package main

import (
	"context"
	"fmt"
	"log"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-redis/redis/v9"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/nsqio/go-nsq"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

const (
	service     = "backend"
	environment = "development"
	id          = 1
)

func SetupServer() (*Server, error) {

	/************************ REDIS *********************************/

	rdb, err := ConnectRedis()
	if err != nil {
		return nil, err
	}

	/************************** POSTGRES *****************************/

	pgconn, err := ConnectPostgre()
	if err != nil {
		return nil, err
	}

	/************************** NSQ **********************************/

	nsq, err := ConnectNSQ()
	if err != nil {
		return nil, err
	}

	/************************ TRACING *********************************/
	tracer, err := SetupTracerProvider()
	if err != nil {
		return nil, err
	}

	/************************ NATs *********************************/
	// var conn_string string

	// if NATS_URL == "" {
	// 	conn_string = nats.DefaultURL
	// } else {
	// 	conn_string = fmt.Sprintf("%s:4222", NATS_URL)
	// }
	fmt.Println("Vars", NATS_URL)
	nc, err := nats.Connect(fmt.Sprintf("%s:4222", NATS_URL))
	if err != nil {
		return nil, err
	}

	/************************** Chi MUX *********************************/

	mux := CreateRouter()

	/*****************/

	s := &Server{
		redis: rdb,
		pg:    pgconn,
		nsq:   nsq,
		mux:   mux,
		tp:    tracer,
		nats:  nc,
	}

	// Register our TracerProvider as the global so any imported
	// instrumentation in the future will default to using it.
	otel.SetTracerProvider(s.tp)
	// In order to propagate trace context over the wire,
	// a propagator must be registered with the OpenTelemetry API.
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return s, nil
}

func ConnectRedis() (*redis.Client, error) {

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:6379", CacheURL),
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// Testing for a valid connection.
	err := rdb.Set(context.Background(), "TEST", "Connection", 0).Err()
	if err != nil {
		return nil, fmt.Errorf("unable to set redis value: %v", err)

	}
	err = rdb.Del(context.Background(), "TEST").Err()
	if err != nil {
		return nil, fmt.Errorf("unable to delete the redis key: %v", err)
	}

	log.Println("Successfully connected to Redis")
	return rdb, nil
}

func ConnectPostgre() (*pgxpool.Pool, error) {
	conn, err := pgxpool.New(context.Background(), PgURL)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v", err)
	}

	log.Println("Successfully connected to Postgres")

	return conn, nil
}

func ConnectNSQ() (*nsq.Producer, error) {
	config := nsq.NewConfig()
	producer, err := nsq.NewProducer(fmt.Sprintf("%s:4150", NSQD), config)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to NSQ Demon %v", err)
	}

	log.Println("Successfully connected to NSQDemon")
	return producer, nil
}

// CreateRouter creates the router and attaches some default middlewares.
func CreateRouter() *chi.Mux {
	mux := chi.NewRouter()

	// Makes it far easier to implement Middleware for all Routes.
	// A good base middleware stack
	mux.Use(middleware.RequestID) // Injects a request ID into the context of each request
	mux.Use(middleware.RealIP)    // Sets a http.Request's RemoteAddr to either X-Real-IP or X-Forwarded-For
	mux.Use(middleware.Logger)    // Logs the start and end of each request with the elapsed processing time
	mux.Use(middleware.Recoverer) // Gracefully absorb panics and prints the stack trace

	return mux
}

// SetupTracerProvider creates the base stuff. Stolen from
// https://dev.to/aurelievache/learning-go-by-examples-part-10-instrument-your-go-app-with-opentelemetry-and-send-traces-to-jaeger-distributed-tracing-1p4a
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
