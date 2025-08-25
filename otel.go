package main

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/metric"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// Global telemetry instances
var (
	appTracer trace.Tracer
	appMeter  metric.Meter
	appLogger *slog.Logger
)

// setupTracing configures OpenTelemetry tracing with OTLP gRPC exporter.
func setupTracing(ctx context.Context, res *resource.Resource, otlpEndpoint string) (*sdktrace.TracerProvider, error) {
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(otlpEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	return tp, nil
}

// setupMetrics configures OpenTelemetry metrics with OTLP gRPC exporter.
func setupMetrics(ctx context.Context, res *resource.Resource, otlpEndpoint string) (*sdkmetric.MeterProvider, error) {
	metricExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(otlpEndpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	return mp, nil
}

// setupLogging configures OpenTelemetry logging with OTLP gRPC exporter and structured logging.
func setupLogging(ctx context.Context, res *resource.Resource, otlpEndpoint, serviceName string) (*sdklog.LoggerProvider, error) {
	logExporter, err := otlploggrpc.New(ctx,
		otlploggrpc.WithEndpoint(otlpEndpoint),
		otlploggrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		sdklog.WithResource(res),
	)
	global.SetLoggerProvider(lp)

	// Create structured logger that will send logs to OTLP
	otelHandler := otelslog.NewHandler(serviceName)
	appLogger = slog.New(otelHandler)

	return lp, nil
}

// setupInstrumentation initializes OpenTelemetry with tracing, metrics, and logging.
// Returns a cleanup function that should be called before application shutdown.
func setupInstrumentation(serviceName string) func() {
	ctx := context.Background()

	// Get OTLP endpoint from environment or use default
	otlpEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otlpEndpoint == "" {
		otlpEndpoint = "localhost:4317"
	}

	// Create resource with service identification
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		slog.Error("failed to create resource", "error", err)
		panic(err)
	}

	// Setup tracing
	tp, err := setupTracing(ctx, res, otlpEndpoint)
	if err != nil {
		slog.Error("failed to setup tracing", "error", err)
		panic(err)
	}
	appTracer = otel.Tracer(serviceName)

	// Setup metrics
	mp, err := setupMetrics(ctx, res, otlpEndpoint)
	if err != nil {
		slog.Error("failed to setup metrics", "error", err)
		panic(err)
	}
	appMeter = otel.Meter(serviceName)

	// Setup logging
	lp, err := setupLogging(ctx, res, otlpEndpoint, serviceName)
	if err != nil {
		slog.Error("failed to setup logging", "error", err)
		panic(err)
	}

	appLogger.Info("OpenTelemetry instrumentation initialized",
		"service", serviceName,
		"endpoint", otlpEndpoint)

	// Return cleanup function
	return func() {
		appLogger.Info("Shutting down OpenTelemetry instrumentation")

		if err := tp.Shutdown(ctx); err != nil {
			slog.Error("failed to shutdown tracer provider", "error", err)
		}
		if err := mp.Shutdown(ctx); err != nil {
			slog.Error("failed to shutdown meter provider", "error", err)
		}
		if err := lp.Shutdown(ctx); err != nil {
			slog.Error("failed to shutdown logger provider", "error", err)
		}
	}
}

// GetTracer returns the global tracer instance.
// Call setupInstrumentation first.
func GetTracer() trace.Tracer {
	return appTracer
}

// GetMeter returns the global meter instance.
// Call setupInstrumentation first.
func GetMeter() metric.Meter {
	return appMeter
}

// GetLogger returns the global structured logger instance.
// Call setupInstrumentation first.
func GetLogger() *slog.Logger {
	return appLogger
}
