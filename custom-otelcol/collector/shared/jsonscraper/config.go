package jsonscraper

import (
	"fmt"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Queries []*Query `yaml:"queries"`
}

type Query struct {
	Name     string    `yaml:"name"`
	Rules    Rule      `yaml:"rules"`
	Resource *Resource `yaml:"resource"`
	Scope    *Scope    `yaml:"scope"`
}

type Resource struct {
	Name       string      `yaml:"name"`
	Attributes []Attribute `yaml:"attributes"`
}

type Scope struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

type Rule struct {
	Select             string       `yaml:"select"`
	EmitMetric         []MetricEmit `yaml:"emitMetric"`
	EmitLogs           []LogEmit    `yaml:"emitLogs"`
	EmitDbRecord       []DBEmit     `yaml:"emitDbRecord"`
	ForEach            *Rule        `yaml:"forEach"`
	Query              string       `yaml:"query"`
	QueryParameters    []Attribute  `yaml:"queryParameters"`
	QueryPostData      *string      `yaml:"queryPostData"`
	ResourceAttributes []Attribute  `yaml:"resourceAttributes"`
	ItemAttributes     []Attribute  `yaml:"itemAttributes"`
	Reducers           []string     `yaml:"reducers"`
	ReducerMaps        []ReducerMap `yaml:"reducerMaps"`
}

type MetricEmit struct {
	Name               string            `yaml:"name"`
	Description        string            `yaml:"description"`
	Filters            []Filter          `yaml:"filters"` // filters are joined by AND
	Unit               string            `yaml:"unit"`
	Type               MetricType        `yaml:"type"`
	Monotonic          bool              `yaml:"monotonic"`
	Temporality        MetricTemporality `yaml:"temporality"`
	ValueFrom          string            `yaml:"valueFrom"`
	ItemAttributes     []Attribute       `yaml:"itemAttributes"`
	ResourceAttributes []Attribute       `yaml:"resourceAttributes"`
	// ExpressionOnVal    string            `yaml:"expressionOnVal"`
	// TODO - check if the above can be removed ^^^
}

type DBEmit struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"description"`
	Filters     []Filter    `yaml:"filters"` // filters are joined by AND
	DB          string      `yaml:"db"`
	Dump        bool        `yaml:"dump"`
	Fields      []Attribute `yaml:"fields"`
}

type LogEmit struct {
	Filters            []Filter    `yaml:"filters"`       // filters are joined by AND
	LogType            string      `yaml:"logType"`       // fault, event, or audit
	MessageFrom        string      `yaml:"messageFrom"`   // expression returning the whole message
	SeverityFrom       string      `yaml:"severityFrom"`  // expression returning string with services' severity
	TimestampFrom      string      `yaml:"timestampFrom"` // expression returning log entry timestamp
	ItemAttributes     []Attribute `yaml:"itemAttributes"`
	ResourceAttributes []Attribute `yaml:"resourceAttributes"`
}

type ReducerMap struct {
	Name      string   `yaml:"name"`
	Value     *float64 `yaml:"value"`
	ValueFrom *string  `yaml:"valueFrom"`
}

type Filter struct {
	Name string `yaml:"name"`
	Is   string `yaml:"is"` // must evaluate to bool
}

type Attribute struct {
	Name      string `yaml:"name"`
	Value     string `yaml:"value"`
	ValueFrom string `yaml:"valueFrom"`
}

type MetricType string
type MetricTemporality string

const (
	Sum        MetricType        = "sum"
	Gauge      MetricType        = "gauge"
	Cumulative MetricTemporality = "cumulative"
	Delta      MetricTemporality = "delta"
)

func NewScraperConfig() Config {
	return Config{
		Queries: []*Query{},
	}
}

func (c *Config) AddQueryRules(rules []byte) error {
	rulesParsed := &Config{}
	err := yaml.Unmarshal(rules, rulesParsed)
	if err != nil {
		return fmt.Errorf("config queries: cannot parse rule config file %s - %v", string(rules), err)
	}
	for _, q := range rulesParsed.Queries {
		c.Queries = append(c.Queries, q)
	}
	return nil
}
