package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// Provider manages the OpenTelemetry tracer provider
type Provider struct {
	tracerProvider *sdktrace.TracerProvider
	tracer         trace.Tracer
	exporter       *SQLiteExporter
}

// Config holds telemetry configuration
type Config struct {
	Enabled     bool
	ServiceName string
	DBPath      string
}

// DefaultConfig returns default telemetry configuration
func DefaultConfig() Config {
	return Config{
		Enabled:     true,
		ServiceName: "gocode-agent",
		DBPath:      "traces.db",
	}
}

// NewProvider creates a new telemetry provider
func NewProvider(config Config) (*Provider, error) {
	if !config.Enabled {
		return &Provider{}, nil
	}

	// Create resource with service information
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create SQLite exporter
	exporter, err := NewSQLiteExporter(config.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// Set global provider
	otel.SetTracerProvider(tp)

	// Create tracer
	tracer := tp.Tracer("gocode-agent")

	return &Provider{
		tracerProvider: tp,
		tracer:         tracer,
		exporter:       exporter,
	}, nil
}

// Tracer returns the configured tracer
func (p *Provider) Tracer() trace.Tracer {
	if p.tracer == nil {
		return trace.NewNoopTracerProvider().Tracer("noop")
	}
	return p.tracer
}

// Shutdown gracefully shuts down the provider
func (p *Provider) Shutdown(ctx context.Context) error {
	if p.tracerProvider == nil {
		return nil
	}

	if err := p.tracerProvider.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown tracer provider: %w", err)
	}

	if p.exporter != nil {
		return p.exporter.Close()
	}

	return nil
}

// ForceFlush forces all pending spans to be exported
func (p *Provider) ForceFlush(ctx context.Context) error {
	if p.tracerProvider == nil {
		return nil
	}

	return p.tracerProvider.ForceFlush(ctx)
}
