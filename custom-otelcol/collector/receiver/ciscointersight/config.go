package ciscointersight

import (
	"fmt"
	"os"

	"github.com/chrlic/otelcol-cust/collector/shared/jsonscraper"
	"go.opentelemetry.io/collector/component"
)

type ContextProvider struct {
	Name          string                `mapstructure:"name"`
	Subscriptions []ContextSubscription `mapstructure:"subscriptions"`
}

type ContextSubscription struct {
	Topic string `mapstructure:"topic"`
	Table string `mapstructure:"table"`
}

type IntersightConfig struct {
	Host       string `mapstructure:"host"`
	ApiKeyId   string `mapstructure:"apiKeyId"`
	ApiKeyFile string `mapstructure:"apiKeyFile"`
}

// Config - represents the receivers' configuration in config.yaml file of the collector
type Config struct {
	Interval         int                   `mapstructure:"interval"`
	Intersight       IntersightConfig      `mapstructure:"intersight"`
	Resource         *jsonscraper.Resource `mapstructure:"resource"`
	Scope            *jsonscraper.Scope    `mapstructure:"scope"`
	QueryFiles       []string              `mapstructure:"queryFiles"`
	DbSchemas        []string              `mapstructure:"tableSchemas"`
	ContextProviders []*ContextProvider    `mapstructure:"contextProviders"`
	ScraperConfig    jsonscraper.Config
}

// Validate - check validity of the configuration
func (cfg *Config) Validate() error {

	if len(cfg.QueryFiles) == 0 {
		return fmt.Errorf("at least one query file required")
	}

	cfg.ScraperConfig = jsonscraper.NewScraperConfig()

	resourcesInQueries := true
	scopesInQueries := true
	if len(cfg.QueryFiles) > 0 {
		for _, confFile := range cfg.QueryFiles {
			queryConfig, err := os.ReadFile(confFile)
			if err != nil {
				return fmt.Errorf("intersight.queries: cannot read config file %s - %v", confFile, err)
			}

			cfg.ScraperConfig.AddQueryRules(queryConfig)

		}
	}

	if cfg.Resource == nil && !resourcesInQueries {
		return fmt.Errorf("resource must be specified either globally or in each query")
	}

	if cfg.Scope == nil && !scopesInQueries {
		return fmt.Errorf("scope must be specified either globally or in each query")
	}

	return nil
}

func createDefaultConfig() component.Config {
	cfg := &Config{}

	return cfg
}
