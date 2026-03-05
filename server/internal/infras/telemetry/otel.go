package telemetry

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/justblue/samsa/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Telemetry holds OpenTelemetry configuration and cleanup function
type Telemetry struct {
	Cancel func()
}

// SetupOTLPExporter bootstraps the OpenTelemetry pipeline.
// If it does not return an error, make sure to call shutdown for proper cleanup.
// Reference: https://github.com/open-telemetry/opentelemetry-go/blob/main/example/dice/otel.go
func SetupOTLPExporter(ctx context.Context, c *config.Config) func() {
	res, err := resource.New(ctx,
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(c.OpenTelemetry.ServiceName),
			semconv.ServiceVersionKey.String(c.OpenTelemetry.ServiceVersion),
		),
	)
	if err != nil {
		log.Println(fmt.Errorf("creating resource, %v", err))
	}

	conn, err := dialGrpc(ctx, c)
	if err != nil {
		log.Println(fmt.Errorf("connecting to otel-collecter: %w", err))
	}

	tracerProvider, err := newTraceProvider(ctx, res, conn, c)
	if err != nil {
		log.Println(fmt.Errorf("creating trace provider, %v", err))
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	otel.SetTracerProvider(tracerProvider)

	meterProvider, err := newMeterProvider(ctx, res, conn)
	if err != nil {
		log.Println(fmt.Errorf("creating meter provider, %v", err))
	}

	otel.SetMeterProvider(meterProvider)

	log.Println("otlp connected.")

	return func() {
		cxt, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		if err := tracerProvider.Shutdown(cxt); err != nil {
			otel.Handle(err)
		}
		// pushes any last exports to the receiver
		if err := meterProvider.Shutdown(cxt); err != nil {
			otel.Handle(err)
		}
	}
}

func dialGrpc(ctx context.Context, c *config.Config) (*grpc.ClientConn, error) {
	log.Printf("dialing %s\n", c.OpenTelemetry.Endpoint)

	base, maxBackoff := time.Second, time.Minute
	backoff := base

	for {
		// Attempt to dial using the provided context. WithBlock makes DialContext wait
		// until the connection is ready or the context is done.
		conn, err := grpc.NewClient(c.OpenTelemetry.Endpoint,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err == nil {
			log.Println("gRPC connected.")
			return conn, nil
		}

		log.Printf("failed to connect to gRPC: %v", err)

		// If the context is done, return the context's error instead of retrying.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Apply exponential backoff with jitter before retrying.
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
		jitter := rand.Int63n(int64(backoff * 3)) //nolint
		sleep := base + time.Duration(jitter)
		time.Sleep(sleep)
		log.Println("retrying to connect to gRPC...")

		// Exponentially increase backoff for next attempt.
		if backoff < maxBackoff {
			backoff <<= 1
		}
	}
}

func newTraceProvider(ctx context.Context, res *resource.Resource, conn *grpc.ClientConn, c *config.Config) (*trace.TracerProvider, error) {
	traceExp, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, err
	}

	bsp := trace.NewBatchSpanProcessor(traceExp)
	tracerProvider := trace.NewTracerProvider(
		trace.WithSampler(trace.TraceIDRatioBased(c.OpenTelemetry.SamplerRatio)),
		trace.WithResource(res),
		trace.WithSpanProcessor(bsp),
	)

	return tracerProvider, nil
}

func newMeterProvider(ctx context.Context, res *resource.Resource, conn *grpc.ClientConn) (*metric.MeterProvider, error) {
	metricExporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, err
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(
			metricExporter,
			metric.WithInterval(5*time.Second),
		)),
	)
	return meterProvider, nil
}
