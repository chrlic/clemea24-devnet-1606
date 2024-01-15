package ciscoaci

import (
	"fmt"
	"os"

	"github.com/chrlic/otelcol-cust/collector/shared/jsonscraper"
	"go.opentelemetry.io/collector/component"
)

type AciConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Protocol string `mapstructure:"protocol"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Socks5   string `mapstructure:"socks5"`
}

// type AciQueries struct {
// 	Queries []*AciQuery `yaml:"queries"`
// }

// type AciQuery struct {
// 	Name     string       `yaml:"name"`
// 	Rules    AciRule      `yaml:"rules"`
// 	Resource *AciResource `yaml:"resource"`
// 	Scope    *AciScope    `yaml:"scope"`
// }

// type AciResource struct {
// 	Name       string         `yaml:"name"`
// 	Attributes []AciAttribute `yaml:"attributes"`
// }

// type AciScope struct {
// 	Name    string `yaml:"name"`
// 	Version string `yaml:"version"`
// }

// type AciRule struct {
// 	Select             string         `yaml:"select"`
// 	Emit               []AciEmit      `yaml:"emit"`
// 	EmitLogs           []AciLogEmit   `yaml:"emitLogs"`
// 	ForEach            *AciRule       `yaml:"forEach"`
// 	Query              string         `yaml:"query"`
// 	QueryParams        []AciAttribute `yaml:"queryParams"`
// 	ResourceAttributes []AciAttribute `yaml:"resourceAttributes"`
// }

// type AciEmit struct {
// 	Name               string               `yaml:"name"`
// 	Description        string               `yaml:"description"`
// 	Unit               string               `yaml:"unit"`
// 	Type               AciMetricType        `yaml:"type"`
// 	Monotonic          bool                 `yaml:"monotonic"`
// 	Temporality        AciMetricTemporality `yaml:"temporality"`
// 	ValueFrom          string               `yaml:"valueFrom"`
// 	ExpressionOnVal    string               `yaml:"expressionOnVal"`
// 	Attributes         []AciAttribute       `yaml:"attributes"`
// 	ResourceAttributes []AciAttribute       `yaml:"resourceAttributes"`
// }

// type AciLogEmit struct {
// 	Filters            []AciFilter    `yaml:"filters"`       // filters are joined by AND
// 	LogType            string         `yaml:"logType"`       // fault, event, or audit
// 	MessageFrom        string         `yaml:"messageFrom"`   // expression returning the whole message
// 	SeverityFrom       string         `yaml:"severityFrom"`  // expression returning string with ACI severity
// 	TimestampFrom      string         `yaml:"timestampFrom"` // expression returning log entry timestamp
// 	Attributes         []AciAttribute `yaml:"attributes"`
// 	ResourceAttributes []AciAttribute `yaml:"resourceAttributes"`
// }

// type AciFilter struct {
// 	Name string `yaml:"name"`
// 	Is   string `yaml:"is"` // must evaluate to bool
// }

// type AciLogEntry struct {
// 	Affected         string    `json:"affected,omitempty"`
// 	Cause            string    `json:"cause,omitempty"`
// 	ChangeSet        string    `json:"changeSet,omitempty"`
// 	ChildAction      string    `json:"childAction,omitempty"`
// 	ClientTag        string    `json:"clientTag,omitempty"`
// 	Code             string    `json:"code,omitempty"`
// 	Created          time.Time `json:"created,omitempty"`
// 	Description      string    `json:"descr,omitempty"`
// 	Delegated        string    `json:"delegated,omitempty"`
// 	DelegatedFrom    string    `json:"delegatedFrom,omitempty"`
// 	Dn               string    `json:"dn,omitempty"`
// 	ID               string    `json:"id,omitempty"`
// 	Ind              string    `json:"ind,omitempty"`
// 	ModTs            string    `json:"modTs,omitempty"`
// 	Occur            string    `json:"occur,omitempty"`
// 	OriginalSeverity string    `json:"originalSeverity,omitempty"`
// 	PreviousSeverity string    `json:"previousSeverity,omitempty"`
// 	Rule             string    `json:"rule,omitempty"`
// 	SessionID        string    `json:"sessionId,omitempty"`
// 	Severity         string    `json:"severity,omitempty"`
// 	Status           string    `json:"status,omitempty"`
// 	Subject          string    `json:"subject,omitempty"`
// 	Trig             string    `json:"trig,omitempty"`
// 	TxID             string    `json:"txId,omitempty"`
// 	Type             string    `json:"type,omitempty"`
// 	User             string    `json:"user,omitempty"`
// }

// type AciAttribute struct {
// 	Name      string `yaml:"name"`
// 	Value     string `yaml:"value"`
// 	ValueFrom string `yaml:"valueFrom"`
// }

// type AciMetricType string
// type AciMetricTemporality string

// const (
// 	Sum        AciMetricType        = "sum"
// 	Gauge      AciMetricType        = "gauge"
// 	Cumulative AciMetricTemporality = "cumulative"
// 	Delta      AciMetricTemporality = "delta"
// )

type ContextProvider struct {
	Name          string                `mapstructure:"name"`
	Subscriptions []ContextSubscription `mapstructure:"subscriptions"`
}

type ContextSubscription struct {
	Topic string `mapstructure:"topic"`
	Table string `mapstructure:"table"`
}

// Config - represents the receivers' configuration in config.yaml file of the collector
type Config struct {
	Interval int       `mapstructure:"interval"`
	Aci      AciConfig `mapstructure:"aci"`
	// Resource         *AciResource       `mapstructure:"resource"`
	// Scope            *AciScope          `mapstructure:"scope"`
	QueryFiles       []string           `mapstructure:"queries"`
	DbSchemas        []string           `mapstructure:"tableSchemas"`
	ContextProviders []*ContextProvider `mapstructure:"contextProviders"`
	ScraperConfig    jsonscraper.Config
}

// Validate - check validity of the configuration
func (cfg *Config) Validate() error {

	if cfg.Aci.Host == "" {
		return fmt.Errorf("aci.host is mandatory and missing")
	}
	if cfg.Aci.Port == 0 {
		return fmt.Errorf("aci.port is mandatory and missing")
	}
	if cfg.Aci.Protocol == "" {
		return fmt.Errorf("aci.protocol is mandatory and missing")
	}
	if cfg.Aci.User == "" {
		return fmt.Errorf("aci.user is mandatory and missing")
	}
	if cfg.Aci.Password == "" {
		return fmt.Errorf("aci.password is mandatory and missing")
	}
	if len(cfg.QueryFiles) == 0 {
		return fmt.Errorf("at least one query file required")
	}

	cfg.ScraperConfig = jsonscraper.NewScraperConfig()

	// resourcesInQueries := true
	// scopesInQueries := true
	if len(cfg.QueryFiles) > 0 {
		for _, confFile := range cfg.QueryFiles {
			queryConfig, err := os.ReadFile(confFile)
			if err != nil {
				return fmt.Errorf("intersight.queries: cannot read config file %s - %v", confFile, err)
			}

			cfg.ScraperConfig.AddQueryRules(queryConfig)

		}
	}

	// if cfg.Resource == nil && !resourcesInQueries {
	// 	return fmt.Errorf("resource must be specified either globally or in each query")
	// }

	// if cfg.Scope == nil && !scopesInQueries {
	// 	return fmt.Errorf("scope must be specified either globally or in each query")
	// }

	// resourcesInQueries := true
	// scopesInQueries := true
	// if len(cfg.QueryFiles) > 0 {
	// 	for _, confFile := range cfg.QueryFiles {
	// 		queryConfig, err := os.ReadFile(confFile)
	// 		if err != nil {
	// 			return fmt.Errorf("aci.queries: cannot read config file %s - %v", confFile, err)
	// 		}
	// 		aciQueries := &AciQueries{}
	// 		err = yaml.Unmarshal(queryConfig, aciQueries)
	// 		if err != nil {
	// 			return fmt.Errorf("aci.queries: cannot parse config file %s - %v", confFile, err)
	// 		}
	// 		for _, q := range aciQueries.Queries {
	// 			cfg.Queries = append(cfg.Queries, q)
	// 			resourcesInQueries = resourcesInQueries && (q.Resource != nil)
	// 			scopesInQueries = scopesInQueries && (q.Scope != nil)
	// 		}
	// 	}
	// }

	// if cfg.Resource == nil && !resourcesInQueries {
	// 	return fmt.Errorf("resource must be specified either globally or in each query")
	// }

	// if cfg.Scope == nil && !scopesInQueries {
	// 	return fmt.Errorf("scope must be specified either globally or in each query")
	// }

	return nil
}

func createDefaultConfig() component.Config {
	cfg := &Config{}

	return cfg
}
