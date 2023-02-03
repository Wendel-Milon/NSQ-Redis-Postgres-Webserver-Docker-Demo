package main

import (
	"fmt"
	"html"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

const (
	service     = "tracingApp"
	environment = "development"
	id          = 1
)

var Jaeger = os.Getenv("JAEGER_URL") //Port is 14268

// SetupTracerProvider creates the Jaeger exporter.
func SetupTracerProvider() (*tracesdk.TracerProvider, error) {
	url := fmt.Sprintf("http://%s/api/traces", Jaeger)
	// Create the Jaeger exporter
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(url)))
	if err != nil {
		return nil, err
	}
	tp := tracesdk.NewTracerProvider(
		// Always be sure to batch in production.
		tracesdk.WithBatcher(exp),
		// Record information about this application in a Resource.
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(service),
			attribute.String("environment", environment),
			attribute.Int64("ID", id),
		)),
	)
	return tp, nil
}

func main() {

	rand.Seed(time.Now().Unix())

	tp, err := SetupTracerProvider()
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	// Register our TracerProvider as the global so any imported
	// instrumentation in the future will default to using it.
	otel.SetTracerProvider(tp)
	// I think this is very important but I dont know why...
	otel.SetTextMapPropagator(propagation.TraceContext{})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		propagators := otel.GetTextMapPropagator()

		// This seems to be the correct way to extract headers.
		ctx := propagators.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		_, span := otel.Tracer("Something").Start(ctx, "Default Path")
		defer span.End()

		n := rand.Intn(1000)
		time.Sleep(time.Millisecond * time.Duration(n))

		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
		log.Info().Msgf("Message received!")
	})

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal().Err(http.ListenAndServe(":8001", nil)).Msg("")

}
