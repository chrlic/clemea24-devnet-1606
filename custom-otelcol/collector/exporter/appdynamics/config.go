package appdmetric

import (
	"fmt"
	"html/template"
)

type AnalyticsConfig struct {
	Url               string `mapstructure:"url"`
	GlobalAccountName string `mapstructure:"globalAccountName"`
	ApiKey            string `mapstructure:"apiKey"`
	MetricsTable      string `mapstructure:"metricsTable"`
	LogsTable         string `mapstructure:"logsTable"`
	InitTable         bool   `mapstructure:"initTable"`
}

type MatchConditions struct {
	Attribute  *string `mapstructure:"attribute"`
	MetricName *string `mapstructure:"metricName"`
	Equals     *string `mapstructure:"equals"`
	NotEquals  *string `mapstructure:"notEquals"`
	Exists     *bool   `mapstructure:"exists"`
}

type Rule struct {
	Description        string            `mapstructure:"description"`
	MatchConditions    []MatchConditions `mapstructure:"matchConditions"`
	PathTemplate       string            `mapstructure:"pathTemplate"`
	AggregatorType     string            `mapstructure:"aggregatorType"` // AVERAGE | SUM | OBSERVATION. Average = default
	pathTemplateParsed *template.Template
}

type MetricsConfig struct {
	Url              string `mapstructure:"url"`
	Prefix           string `mapstructure:"prefix"`
	Rules            []Rule `mapstructure:"rules"`
	LogMetricRecords bool   `mapstructure:"logMetricRecords"`
}

// Config - represents the exporter configuration in config.yaml file of the collector
type Config struct {
	Metrics   MetricsConfig   `mapstructure:"metrics"`
	Analytics AnalyticsConfig `mapstructure:"analytics"`
}

// Validate - check validity of the configuration
func (cfg *Config) Validate() error {

	for _, rule := range cfg.Metrics.Rules {
		for _, cond := range rule.MatchConditions {
			if cond.Attribute != nil && cond.MetricName != nil {
				return fmt.Errorf("In matchConditions, either attribute or metricName can be specified - attribute = %s, metric = %s", *cond.Attribute, *cond.MetricName)
			}
			if cond.Attribute == nil && cond.MetricName == nil {
				return fmt.Errorf("In matchConditions, either attribute or metricName must be specified")
			}
			if cond.MetricName != nil && (cond.Equals != nil || cond.Exists != nil || cond.NotEquals != nil) {
				return fmt.Errorf("In matchConditions, metricName can be only tested for the metric name - metric %s", *cond.MetricName)
			}
			if cond.Equals != nil && cond.NotEquals != nil {
				return fmt.Errorf("In matchConditions, either equals or notEquals can be specified - equals = %s, notEquals = %s", *cond.Equals, *cond.NotEquals)
			}
			if cond.Attribute != nil && (cond.Equals == nil && cond.NotEquals == nil && cond.Exists == nil) {
				return fmt.Errorf("In matchConditions, either equals or notEquals or exists condition must be specified")
			}
			if cond.Exists != nil && (cond.Equals != nil || cond.NotEquals != nil) {
				return fmt.Errorf("In matchConditions, either exists condition cannot be used with equals or notEquals condition - exists = %t, equals = %s, notEquals = %s", *cond.Exists, *cond.Equals, *cond.NotEquals)
			}
		}
	}

	return nil
}
