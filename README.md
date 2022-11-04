# opentelemetry-distributed-tracing-example

A slightly messy example system using distributed traces, made in golang using [OpenTelemetry](https://opentelemetry.io/docs/instrumentation/go/) for creating traces in the code, [Grafana Tempo](https://grafana.com/oss/tempo/) with local storage as tracing backend, [Loki](https://grafana.com/oss/loki/) for logs with their Docker logging driver, [Prometheus](https://prometheus.io/docs/introduction/overview/) to collect metrics, and [Grafana](https://grafana.com/grafana/) to show the results.

To run it make sure docker, docker-compose and the loki docker log driver is installed, run `docker-compose up -d`, go to http://127.0.0.1:8080/?id=asdf and perhaps http://127.0.0.1:8080/ a couple of times, open grafana at http://localhost:3000/explore?orgId=1, select the Loki data source, make a query like `{container_name=~"tracing_app1_1|tracing_app2_1"}`, and look for the `traceID` fields with Tempo buttons.
