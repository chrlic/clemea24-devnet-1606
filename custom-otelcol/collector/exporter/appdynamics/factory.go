package appdmetric

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

const (
	typeStr       = "appdynamics"
	stability     = component.StabilityLevelAlpha
	defaultMAUrl  = "http://localhost:8293/api/v1/metrics"
	defaultPrefix = "Custom Metrics"
	defaultTable  = "otlpmetrics"
)

// NewFactory creates a factory for OTLP exporter.
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		typeStr,
		createDefaultConfig,
		exporter.WithMetrics(createMetricsExporter, stability),
		exporter.WithLogs(createLogsExporter, stability),
	)
}

func createMetricsExporter(
	ctx context.Context,
	set exporter.CreateSettings,
	cfg component.Config,
) (exporter.Metrics, error) {

	c := cfg.(*Config)
	exporter, err := newMetricsExporter(c, set.Logger)
	if err != nil {
		return nil, fmt.Errorf("cannot configure %s metrics exporter: %w", typeStr, err)
	}

	return exporterhelper.NewMetricsExporter(
		ctx,
		set,
		cfg,
		exporter.pushMetricsData,
		exporterhelper.WithStart(exporter.start),
		exporterhelper.WithShutdown(exporter.shutdown),
	)
}

func createLogsExporter(
	ctx context.Context,
	set exporter.CreateSettings,
	cfg component.Config,
) (exporter.Logs, error) {

	c := cfg.(*Config)
	exporter, err := newLogsExporter(c, set.Logger)
	if err != nil {
		return nil, fmt.Errorf("cannot configure %s logs exporter: %w", typeStr, err)
	}

	return exporterhelper.NewLogsExporter(
		ctx,
		set,
		cfg,
		exporter.pushLogsData,
		exporterhelper.WithStart(exporter.start),
		exporterhelper.WithShutdown(exporter.shutdown),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		Metrics: MetricsConfig{
			Url:    defaultMAUrl,
			Prefix: defaultPrefix,
			Rules:  []Rule{},
		},
		Analytics: AnalyticsConfig{
			MetricsTable: defaultTable,
		},
	}
}
