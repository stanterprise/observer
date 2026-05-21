// Package telemetry provides OpenTelemetry initialisation helpers for Observer
// services. It wires up a Prometheus exporter so that each service exposes a
// /metrics HTTP endpoint that can be scraped by Prometheus directly or via a
// Prometheus Operator ServiceMonitor.
//
// Usage (in a service main):
//
//	shutdown, err := telemetry.Setup(ctx, "observer-api", logger)
//	if err != nil {
//	    logger.Warn("telemetry setup failed – metrics disabled", "error", err)
//	} else {
//	    defer shutdown(context.Background())
//	}
//
//	stopMetrics := telemetry.StartMetricsServer(envOr("METRICS_PORT", "9090"), logger)
//	defer stopMetrics(context.Background())
package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Setup initialises the global OTel MeterProvider with a Prometheus exporter
// and starts Go runtime metric collection (goroutines, GC, memory).
// The returned shutdown function must be called before the process exits.
func Setup(ctx context.Context, serviceName string, logger *slog.Logger) (func(context.Context) error, error) {
	if logger == nil {
		logger = slog.Default()
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(serviceName)),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithProcessPID(),
	)
	if err != nil {
		return nil, fmt.Errorf("create otel resource: %w", err)
	}

	exporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("create prometheus exporter: %w", err)
	}

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(exporter),
	)

	otel.SetMeterProvider(provider)

	// Collect Go runtime metrics (goroutines, GC pause, heap allocations …)
	// with a 15-second minimum read interval to reduce overhead.
	if err := runtime.Start(runtime.WithMinimumReadMemStatsInterval(15 * time.Second)); err != nil {
		// Non-fatal: the rest of the metrics pipeline still works.
		logger.Warn("runtime metrics collection failed to start", "error", err)
	}

	logger.Info("telemetry initialised", "service", serviceName)

	return provider.Shutdown, nil
}

// Meter returns a named OTel Meter from the global MeterProvider.
// If telemetry was not set up, this returns a no-op Meter so callers do not
// need to guard against nil.
func Meter(name string) metric.Meter {
	return otel.GetMeterProvider().Meter(name)
}

// StartMetricsServer starts a minimal HTTP server that exposes:
//   - GET /metrics – Prometheus scrape endpoint
//   - GET /health  – simple liveness probe
//
// The server runs in a background goroutine.  The returned function shuts it
// down gracefully when called.
func StartMetricsServer(port string, logger *slog.Logger) func(context.Context) error {
	if logger == nil {
		logger = slog.Default()
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("metrics server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("metrics server stopped unexpectedly", "error", err)
		}
	}()

	return srv.Shutdown
}
