package ciscoaci

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

const (
	typeStr   = "ciscoaci"
	stability = component.StabilityLevelDevelopment
)

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeStr,
		createDefaultConfig,
		receiver.WithMetrics(createAciReceiver, stability),
		receiver.WithLogs(createAciLogReceiver, stability),
	)
}

func createAciReceiver(
	_ context.Context,
	settings receiver.CreateSettings,
	cc component.Config,
	consumer consumer.Metrics,
) (receiver.Metrics, error) {
	cfg := cc.(*Config)
	return &aciReceiver{
		metricConsumer:   consumer,
		config:           cfg,
		logger:           settings.Logger,
		receiverID:       settings.ID.String(),
		isMetricReceiver: true,
		isLogReceiver:    false,
	}, nil
}

func createAciLogReceiver(_ context.Context, settings receiver.CreateSettings, cc component.Config, consumer consumer.Logs) (receiver.Logs, error) {
	cfg := cc.(*Config)
	return &aciReceiver{
		logConsumer:      consumer,
		config:           cfg,
		logger:           settings.Logger,
		receiverID:       settings.ID.String(),
		isMetricReceiver: false,
		isLogReceiver:    true,
	}, nil
}
