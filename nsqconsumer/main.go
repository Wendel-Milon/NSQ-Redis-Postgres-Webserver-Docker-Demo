package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/nsqio/go-nsq"
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
	service     = "NSQ_Consumer"
	environment = "development"
	id          = 1
)

var NSQ_LOOKUP = os.Getenv("NSQ_LOOKUP")
var NSQ_CHAN = os.Getenv("NSQ_CHAN")
var NSQ_TOPIC = os.Getenv("NSQ_TOPIC")
var Jaeger = os.Getenv("JAEGER_URL") //Port is 14268

func main() {

	// For the random Delay.
	rand.Seed(time.Now().Unix())

	tp, err := SetupTracerProvider()
	if err != nil {
		log.Fatal().Err(err).Msgf("")
	}

	// Register our TracerProvider as the global so any imported
	// instrumentation in the future will default to using it.
	otel.SetTracerProvider(tp)
	// I think this is very important but I dont know why...
	otel.SetTextMapPropagator(propagation.TraceContext{})

	log.Info().Msgf("Starting consuming for Lookup:", NSQ_LOOKUP, "; Topic:", NSQ_TOPIC, "; Channel:", "NSQ_CHAN")
	ConsumeMessage()
}

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

type myMessageHandler struct{}

// TODO this is stupid and will break someday....
type Message struct {
	Traceparent string
}

func (m Message) Get(key string) string {
	if key == "traceparent" {
		return m.Traceparent
	}
	return ""
}

func (m Message) Set(key string, value string) {
}

func (m Message) Keys() []string {
	return []string{"traceparent"}
}

// HandleMessage implements the Handler interface.
func (h *myMessageHandler) HandleMessage(m *nsq.Message) error {

	if len(m.Body) == 0 {
		// Returning nil will automatically send a FIN command to NSQ to mark the message as processed.
		// In this case, a message with an empty body is simply ignored/discarded.
		return nil
	}

	message := Message{}

	err := json.Unmarshal(m.Body, &message)
	if err != nil {
		log.Info().Err(err).Msgf("")
	}
	// log.Info().Msgf("Success!", message)

	propgator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})

	// carrier := propagation.TextMapCarrier{}

	parentCtx := propgator.Extract(context.Background(), message)
	_, childSpan := otel.Tracer("foo").Start(parentCtx, "child-span-name")
	defer childSpan.End()

	// log.Info().Msgf(parentCtx)
	// log.Info().Msgf(childSpan)

	// do whatever actual message processing is desired
	n := rand.Intn(5)
	time.Sleep(time.Second * time.Duration(n))

	log.Printf("%s\n", message.Traceparent)

	// Returning a non-nil error will automatically send a REQ command to NSQ to re-queue the message.
	return nil
}

// ConsumeMessage does currently not work!!
// This is (guessed) because it polls the NSQLookup which works,
// The Lookup returns the dockerinternal IP Addres to which the client has no acess too.
// To work this either has all be in Docker or nothing at all....
func ConsumeMessage() {

	// Instantiate a consumer that will subscribe to the provided channel.
	config := nsq.NewConfig()
	consumer, err := nsq.NewConsumer(NSQ_TOPIC, NSQ_CHAN, config)
	if err != nil {
		log.Fatal().Err(err).Msgf("")
	}

	// Set the Handler for messages received by this Consumer. Can be called multiple times.
	// See also AddConcurrentHandlers.
	consumer.AddHandler(&myMessageHandler{})

	// Use nsqlookupd to discover nsqd instances.
	// See also ConnectToNSQD, ConnectToNSQDs, ConnectToNSQLookupds.
	err = consumer.ConnectToNSQLookupd(fmt.Sprintf("%s:4161", NSQ_LOOKUP))
	if err != nil {
		log.Fatal().Err(err).Msgf("")
	}

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":2112", nil)
	}()

	// wait for signal to exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Gracefully stop the consumer.
	consumer.Stop()

	// os.Exit(1)

}
