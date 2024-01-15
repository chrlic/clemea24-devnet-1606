package ciscointersight

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

const (
	typeStr   = "ciscointersight"
	stability = component.StabilityLevelDevelopment
)

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeStr,
		createDefaultConfig,
		receiver.WithMetrics(createIntersightMetricReceiver, stability),
		receiver.WithLogs(createIntersightLogReceiver, stability),
	)
}

func createIntersightMetricReceiver(
	_ context.Context,
	settings receiver.CreateSettings,
	cc component.Config,
	consumer consumer.Metrics,
) (receiver.Metrics, error) {
	cfg := cc.(*Config)
	return &intersightReceiver{
		metricConsumer:   consumer,
		config:           cfg,
		logger:           settings.Logger,
		receiverID:       settings.ID.String(),
		isMetricReceiver: true,
		isLogReceiver:    false,
	}, nil
}

func createIntersightLogReceiver(_ context.Context, settings receiver.CreateSettings, cc component.Config, consumer consumer.Logs) (receiver.Logs, error) {
	cfg := cc.(*Config)
	return &intersightReceiver{
		logConsumer:      consumer,
		config:           cfg,
		logger:           settings.Logger,
		receiverID:       settings.ID.String(),
		isMetricReceiver: false,
		isLogReceiver:    true,
	}, nil
}
