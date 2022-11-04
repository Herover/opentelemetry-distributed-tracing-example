package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptrace"
	"os"

	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

const name = "app1"

// newExporter returns a console exporter.
func newExporter(ctx context.Context) (sdktrace.SpanExporter, error) {
	client := otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint("tempo:4318"), // otlp http port
		otlptracehttp.WithInsecure(),             // Defaults to https
	)
	return otlptrace.New(ctx, client)
}

// newResource returns a resource describing this application.
func newResource() *resource.Resource {
	// Mostly just meta info that's static to this application instance
	r, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(name),
			semconv.ServiceVersionKey.String("v0.1.0"),
			attribute.String("environment", "demo"),
		),
	)
	return r
}

func main() {
	// Just use a default logger for now
	logger := log.New(os.Stdout, "", 0)

	// Could also just call context.Background() whenever instead of defining this
	rootCtx := context.Background()

	exp, err := newExporter(rootCtx)
	if err != nil {
		logger.Fatal(err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(newResource()),
	)

	// The thing that will create (and handle?) traces
	otel.SetTracerProvider(tp)

	// Required for trace information to be added to http calls
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	rootHandler := func(w http.ResponseWriter, r *http.Request) {
		// Continue span that library will create for handler
		ctx := r.Context()
		span := trace.SpanFromContext(ctx)
		// ctx, span := otel.Tracer(name).Start(r.Context(), "request")
		// If we created our own span, then we would need to end it manually, but otelhttp does it for us in this case
		// defer span.End()
		ctx = httptrace.WithClientTrace(ctx, otelhttptrace.NewClientTrace(ctx))

		// Use otelhttp transport to add headers to http request
		client := http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		}

		// Print something with a traceID that we can find in logs and relate to trace
		logger.Printf("request traceID=%s %s", span.SpanContext().TraceID(), r.URL.String())

		if r.Method != "GET" {
			return
		}

		if len(r.URL.Query()["id"]) == 0 {
			w.WriteHeader(400)
			fmt.Fprintf(w, "no id")

			span.RecordError(fmt.Errorf("no id"))
			span.SetStatus(codes.Error, "no id")
			return
		}

		// Add something to the baggage
		bag, _ := baggage.Parse("id=" + r.URL.Query()["id"][0])
		ctx = baggage.ContextWithBaggage(ctx, bag)

		requestURL := fmt.Sprintf("http://app2:8080/?id=%s", r.URL.Query()["id"][0])

		logger.Printf("url trace_id=%s %s", span.SpanContext().TraceID(), requestURL)

		// USe context with all the otel parts for http request
		req, _ := http.NewRequestWithContext(ctx, "GET", requestURL, nil)

		// Add trace info to req
		otelhttptrace.Inject(ctx, req)

		// Actually do the thing we want to do
		res, err := client.Do(req)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "Error: %s", err.Error())
			logger.Printf("request error trace_id=%s %s", span.SpanContext().TraceID(), err.Error())

			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())

			return
		}

		logger.Printf("client: status code: %d trace_id=%s", res.StatusCode, span.SpanContext().TraceID())

		// Echo response to client
		body, _ := io.ReadAll(res.Body)
		fmt.Fprint(w, string(body))
	}

	http.Handle("/",
		// Let otelhttp add trace to handler
		otelhttp.NewHandler(
			http.HandlerFunc(rootHandler),
			"root",
		),
	)

	logger.Printf("Starting up")
	http.ListenAndServe(":8080", nil)
}
