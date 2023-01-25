package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// tracing simply wraps the default provided tracing Handlers to be able to use as middleware.
// func tracing(h http.Handler) http.Handler {
// 	return otelhttp.NewHandler(h, "", otelhttp.WithPublicEndpoint())
// }

// SpecialTracing create a additional tracing for this Path. Allows for more custom stuff.
func (server *Server) SpecialTracing(w http.ResponseWriter, r *http.Request) {
	// Use the global TracerProvider

	ctx, span := server.tp.Tracer("CustomTracer").Start(context.Background(), "SpecialTracing")

	n := rand.Intn(1000)
	time.Sleep(time.Millisecond * time.Duration(n))

	span.SetAttributes(attribute.Key("mykey").String("value"))
	defer span.End()

	wg := sync.WaitGroup{}
	wg.Add(2)
	go SpecialTracingDeeper(ctx, &wg)
	go CallOtherServer(ctx, &wg)

	wg.Wait()

	n = rand.Intn(1000)
	time.Sleep(time.Millisecond * time.Duration(n))

	yourName := os.Getenv("MY_NAME")
	fmt.Fprintf(w, "Hello %q!", yourName)
}

// CallOtherServer sends a GET request to another server.
// Injects the TracerID with a traceparent Header.
// Included are two versions to inject the Header.
func CallOtherServer(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	url := fmt.Sprintf("http://%s", TracingApp)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request error: %w", err)
	}

	// Approach 1
	// This adds the traceparent as a header.
	otelhttptrace.Inject(ctx, req)

	client := http.Client{}
	_, err = client.Do(req)
	if err != nil {
		return err
	}

	// // Approach 2
	// client = http.Client{
	// 	// Wrap the Transport with one that starts a span and injects the span context
	// 	// into the outbound request headers.
	// 	Transport: otelhttp.NewTransport(http.DefaultTransport),
	// 	Timeout:   10 * time.Second,
	// }
	// _, err = client.Do(req)
	// if err != nil {
	// 	return err
	// }
	return nil
}

// SpecialTracingDeeper creates a child Span of the Parent Span.
func SpecialTracingDeeper(ctx context.Context, wg *sync.WaitGroup) bool {
	defer wg.Done()

	tr := otel.Tracer("Custom Tracer")

	_, span := tr.Start(ctx, "SpecialTracingDeeper")
	n := rand.Intn(1000)
	time.Sleep(time.Millisecond * time.Duration(n))
	val := rand.Intn(2) != 0

	span.SetAttributes(attribute.Key("val").Bool(val))
	defer span.End()
	return val
}
