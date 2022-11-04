package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

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

const name = "app2"

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

	// The thing that will create (and handle?) traces
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(newResource()),
	)
	otel.SetTracerProvider(tp)

	// Required for trace information to be added to http calls
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			// Adds/extracts trace info
			propagation.TraceContext{},
			// Adds/extracts baggage (key-value stuff)
			propagation.Baggage{},
		),
	)

	// Will be used to find baggage with key "id"
	idKey := attribute.Key("id")

	rootHandler := func(w http.ResponseWriter, r *http.Request) {
		// Find a trace in the request, if any exists
		ctx := r.Context()
		span := trace.SpanFromContext(ctx)
		bag := baggage.FromContext(ctx)
		// This is just a way to do something with the baggage
		span.AddEvent("handling this...", trace.WithAttributes(idKey.String(bag.Member("id").Value())))

		// Print trace_id so we can reference the trace from the logs
		logger.Printf("request trace_id=%s %s", span.SpanContext().TraceID(), r.URL.String())

		if r.Method != "GET" {
			span.RecordError(fmt.Errorf("disallowed method"))
			span.SetStatus(codes.Error, "disallowed method")
			span.End()
			return
		}

		if len(r.URL.Query()["id"]) == 0 {
			fmt.Fprintf(w, "Hi stranger")
			span.RecordError(fmt.Errorf("no id"))
			span.SetStatus(codes.Error, "no id")
		} else {
			span.AddEvent("sending greeting")
			fmt.Fprintf(w, "Hi %s", r.URL.Query()["id"][0])
		}

		// Record the span
		span.End()
	}

	http.Handle(
		"/",
		// Library magically adds trace
		otelhttp.NewHandler(http.HandlerFunc(rootHandler), "root"),
	)

	logger.Printf("Starting up")
	http.ListenAndServe(":8080", nil)
}
