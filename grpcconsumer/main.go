package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"proto"

	"github.com/rs/zerolog/log"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"google.golang.org/grpc"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
)

var Jaeger = os.Getenv("JAEGER_URL")

const (
	service     = "GRPC_CONSUMER"
	environment = "development"
	id          = 1
)

type GrpcServer struct {
	proto.UnimplementedGreeterServer
}

func (g GrpcServer) SayHello(ctx context.Context, in *proto.HelloRequest) (*proto.HelloReply, error) {
	log.Info().Msgf("Called from user %s", in.GetName())
	_, span := otel.Tracer("hello-spn").Start(ctx, "span-name")
	defer span.End()

	span.AddEvent("Custom Event")

	return &proto.HelloReply{Message: "Hello again " + in.GetName()}, nil
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

func promMiddleware() grpc.UnaryServerInterceptor {
	// this is called once!
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		// this is called with every request!
		log.Info().Msg("Prometheus Middleware")
		return handler(ctx, req)
	}
}

func logMiddleware() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		log.Info().Msg("Logging before")

		// Calls wanted function
		resp, err = handler(ctx, req)

		log.Info().Msgf("Logging after %s %v", resp, err)
		return resp, err
	}
}
func main() {

	tp, err := SetupTracerProvider()
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	// Register our TracerProvider as the global so any imported
	// instrumentation in the future will default to using it.
	otel.SetTracerProvider(tp)
	// I think this is very important but I dont know why...
	otel.SetTextMapPropagator(propagation.TraceContext{})

	grpcserver := GrpcServer{}

	lis, err := net.Listen("tcp", ":7777")
	if err != nil {
		log.Fatal().Msgf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.StreamInterceptor(
			otelgrpc.StreamServerInterceptor()),
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(
				otelgrpc.UnaryServerInterceptor(),
				logMiddleware(),
				promMiddleware(),
			),
		),
	)

	proto.RegisterGreeterServer(grpcServer, grpcserver)

	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":2112", nil)

	grpcServer.Serve(lis)
}
