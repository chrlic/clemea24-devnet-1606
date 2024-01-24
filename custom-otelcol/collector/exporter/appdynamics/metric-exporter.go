package appdmetric

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

type metricsExporter struct {
	logger              *zap.Logger
	cfg                 *Config
	appDAnalyticsClient AppDAnalyticsClient
}

type analyticsRecords []analyticsRecord
type analyticsRecord map[string]any

type MetricRecord struct {
	MetricName     string `json:"metricName"`
	AggregatorType string `json:"aggregatorType"`
	Value          int64  `json:"value"`
}

const (
	RESOURCE_PREFIX = "rsrc_"
	SCOPE_PREFIX    = "scp_"
	LABEL_PREFIX    = "lbl_"
)

const PROFILER_FILE = "appdynamics-exporter.pprof"

var DEFAULT_METRICS_ANALYTICS_SCHEMA = AppDAnalyticsSchema{
	Fields: map[string]string{
		"metricName":           "string",
		"metricUnit":           "string",
		"metricType":           "string",
		"metricStartTimestamp": "date",
		"metricTimestamp":      "date",
		"metricValue":          "float",
	},
}

const (
	TRIES_SCHEMA_INIT_AFTER_REINIT = 10 // to workaround delay between delete table and having that table name available again
	RETRY_INTERVAL_SECONDS         = 30
)

var metricCache map[string]float64
var metricCacheUpdates map[string]time.Time
var metricCacheMutex = sync.Mutex{}

func newMetricsExporter(cfg *Config, logger *zap.Logger) (*metricsExporter, error) {

	appDAnalyticsClient := AppDAnalyticsClient{
		GlobalAccountName: cfg.Analytics.GlobalAccountName,
		ApiToken:          cfg.Analytics.ApiKey,
		EventServiceUrl:   cfg.Analytics.Url,
		logger:            logger,
	}

	logger.Sugar().Debugf("Analytics client config: %v", appDAnalyticsClient)

	tries := 10

	appDAnalyticsClient.init()
	if cfg.Analytics.InitTable {
		err := appDAnalyticsClient.DeleteSchema(cfg.Analytics.MetricsTable)
		if err != nil {
			logger.Sugar().Infof("Could not re-initialize table %s", cfg.Analytics.MetricsTable)
		}
		tries = TRIES_SCHEMA_INIT_AFTER_REINIT
	}

	var err error
	for i := 0; i < tries; i++ {
		err = appDAnalyticsClient.CreateSchemaIfNotPresent(cfg.Analytics.MetricsTable, &DEFAULT_METRICS_ANALYTICS_SCHEMA)
		if err != nil {
			logger.Sugar().Infof("Could not re-initialize table (retry) %s - %v", cfg.Analytics.MetricsTable, err)
			time.Sleep(RETRY_INTERVAL_SECONDS * time.Second)
		} else {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("Cannot verify existing schema %s - %v", cfg.Analytics.MetricsTable, err)
	}
	currentSchema, err := appDAnalyticsClient.GetSchema(cfg.Analytics.MetricsTable)
	if err != nil {
		return nil, fmt.Errorf("Cannot read existing schema %s - %v", cfg.Analytics.MetricsTable, err)
	}
	appDAnalyticsClient.Schemas[cfg.Analytics.MetricsTable] = *currentSchema

	return &metricsExporter{
		logger:              logger,
		cfg:                 cfg,
		appDAnalyticsClient: appDAnalyticsClient,
	}, nil
}

func (e *metricsExporter) start(ctx context.Context, host component.Host) error {

	metricCache = map[string]float64{}
	metricCacheUpdates = map[string]time.Time{}
	e.logger.Info("Starting appdmetric exporter\n")
	return nil

	// // Start a process:
	// cmd := exec.Command("sleep", "5")
	// if err := cmd.Start(); err != nil {
	// 	log.Fatal(err)
	// }

	// // Kill it:
	// if err := cmd.Process.Kill(); err != nil {
	// 	log.Fatal("failed to kill process: ", err)
	// }
}

// shutdown will shut down the exporter.
func (e *metricsExporter) shutdown(ctx context.Context) error {

	return nil
}

func (e *metricsExporter) pushMetricsData(ctx context.Context, md pmetric.Metrics) error {

	defer func() {
		if r := recover(); r != nil {
			e.logger.Sugar().Errorf("***** Recovered in pushMetricsData: %v\n%s\n", r, string(debug.Stack()))
		}
	}()

	// profiler_output, err := os.OpenFile(PROFILER_FILE, os.O_WRONLY|os.O_CREATE, 0600)
	// if err != nil {
	// 	e.logger.Error("Cannot open profiler file:", zap.Error(err))
	// }

	// pprof.WriteHeapProfile(profiler_output)

	metricRecords := analyticsRecords{}

	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		// metaData := internal.MetricsMetaData{}
		resourceMetrics := md.ResourceMetrics().At(i)
		// res := metrics.Resource()
		// metaData.ResAttr = attributesToMap(res.Attributes())
		// metaData.ResURL = metrics.SchemaUrl()
		for j := 0; j < resourceMetrics.ScopeMetrics().Len(); j++ {
			scopeMetrics := resourceMetrics.ScopeMetrics().At(j).Metrics()
			// metaData.ScopeURL = metrics.ScopeMetrics().At(j).SchemaUrl()
			// metaData.ScopeInstr = metrics.ScopeMetrics().At(j).Scope()
			for k := 0; k < scopeMetrics.Len(); k++ {
				metric := scopeMetrics.At(k)
				var errs error
				switch metric.Type() {
				case pmetric.MetricTypeGauge:
					resource := resourceMetrics.Resource()
					scope := resourceMetrics.ScopeMetrics().At(j).Scope()
					metricRecords = append(metricRecords, e.appdMetricRecords(&resource, &scope, &metric)...)
					e.logger.Sugar().Debugf("Received: %s -> %f / %v",
						metric.Name(),
						metric.Gauge().DataPoints().At(0).DoubleValue(),
						metric.Gauge().DataPoints().At(0).Attributes().AsRaw(),
					)
				case pmetric.MetricTypeSum:
					resource := resourceMetrics.Resource()
					scope := resourceMetrics.ScopeMetrics().At(j).Scope()
					metricRecords = append(metricRecords, e.appdMetricRecords(&resource, &scope, &metric)...)
				case pmetric.MetricTypeHistogram:
					resource := resourceMetrics.Resource()
					scope := resourceMetrics.ScopeMetrics().At(j).Scope()
					metricRecords = append(metricRecords, e.appdHistogramMetricRecords(&resource, &scope, &metric)...)
					// return fmt.Errorf("Unsupported metric of type MetricTypeHistogram")
				case pmetric.MetricTypeExponentialHistogram:
					return fmt.Errorf("Unsupported metric of type MetricTypeExponentialHistogram")
				case pmetric.MetricTypeSummary:
					return fmt.Errorf("Unsupported metric of type MetricTypeSummary")
				default:
					return fmt.Errorf("unsupported metric of type %v", metric.Type())
				}
				if errs != nil {
					return errs
				}
			}
		}
	}

	if e.cfg.Analytics.Url != "" {
		e.appDAnalyticsClient.EnsureSchema(e.cfg.Analytics.MetricsTable, metricRecords)
		e.appDAnalyticsClient.PostEventsBundled(e.cfg.Analytics.MetricsTable, metricRecords)
	}

	metricTreeRecords := e.getMetricTreeRecords(e.cfg.Metrics, metricRecords)
	if len(metricTreeRecords) > 0 {
		e.postMetricTreeRecords(metricTreeRecords)
	}

	return nil
}

func (e *metricsExporter) appdMetricRecords(resource *pcommon.Resource, scope *pcommon.InstrumentationScope, metric *pmetric.Metric) analyticsRecords {
	metricRecords := analyticsRecords{}

	var dataPoints pmetric.NumberDataPointSlice
	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		dataPoints = metric.Gauge().DataPoints()
		// e.logger.Debug("DP: ", zap.Any("dp len", dataPoints.Len()), zap.Any("dpp", metric.Gauge().DataPoints().At(0)))
	case pmetric.MetricTypeSum:
		dataPoints = metric.Sum().DataPoints()
		// e.logger.Debug("DP: ", zap.Any("dp len", dataPoints.Len()), zap.Any("dpp", metric.Sum().DataPoints().At(0)))
	default:
		return metricRecords
	}

	for i := 0; i < dataPoints.Len(); i++ {
		dp := dataPoints.At(i)

		metricRecord := analyticsRecord{}

		// copy resource attributes
		for name, value := range resource.Attributes().AsRaw() {
			nName := normalizedName(name)
			metricRecord[RESOURCE_PREFIX+nName] = fmt.Sprintf("%s", value)
		}
		if scope.Name() != "" {
			metricRecord[SCOPE_PREFIX+"name"] = scope.Name()
		}
		if scope.Version() != "" {
			metricRecord[SCOPE_PREFIX+"version"] = scope.Version()
		}
		metricRecord["metricName"] = metric.Name()
		metricRecord["metricUnit"] = metric.Unit()
		metricRecord["metricType"] = metric.Type().String()
		metricRecord["metricStartTimestamp"] = dp.StartTimestamp().AsTime().UnixMilli()
		metricRecord["metricTimestamp"] = dp.Timestamp().AsTime().UnixMilli()
		metricRecord["metricValue"] = dp.DoubleValue()
		dpAttributes := dp.Attributes().AsRaw()
		for name, value := range dpAttributes {
			nName := normalizedName(name)
			metricRecord[LABEL_PREFIX+nName] = fmt.Sprintf("%s", value)
		}
		if metric.Type() == pmetric.MetricTypeSum {
			if metric.Sum().AggregationTemporality() == pmetric.AggregationTemporalityCumulative {
				cacheKey := getCacheKey(resource.Attributes().AsRaw(), metric.Name())
				cacheKey = getCacheKey(dp.Attributes().AsRaw(), cacheKey)
				metricCacheMutex.Lock()
				if cacheValue, ok := metricCache[cacheKey]; ok {
					if cacheUpdate, hasRecord := metricCacheUpdates[cacheKey]; hasRecord {
						if time.Now().Compare(cacheUpdate.Add(90*time.Second)) > 0 {
							// missed reporting interval - skip this one and start from scratch
							metricCache[cacheKey] = dp.DoubleValue()
							metricCacheUpdates[cacheKey] = time.Now()
						} else {
							value := dp.DoubleValue() - cacheValue
							metricCache[cacheKey] = dp.DoubleValue()
							metricCacheUpdates[cacheKey] = time.Now()
							metricRecord["metricValue"] = value
							// Sum, Cummulative -> difference over interval
							metricRecords = append(metricRecords, metricRecord)
						}
					} else {
						// cache-update-miss -> new entry + skip this reporting interval
						metricCache[cacheKey] = dp.DoubleValue()
						metricCacheUpdates[cacheKey] = time.Now()
					}
				} else {
					// cache-miss -> new entry + skip this reporting interval
					metricCache[cacheKey] = dp.DoubleValue()
					metricCacheUpdates[cacheKey] = time.Now()
				}
				metricCacheMutex.Unlock()
			} else {
				// Sum, Delta -> direct metric value
				metricRecords = append(metricRecords, metricRecord)
			}
		} else {
			// Gauge -> direct metric value
			metricRecords = append(metricRecords, metricRecord)
		}
	}

	return metricRecords
}

func (e *metricsExporter) appdHistogramMetricRecords(resource *pcommon.Resource, scope *pcommon.InstrumentationScope, metric *pmetric.Metric) analyticsRecords {
	metricRecords := analyticsRecords{}

	var dataPoints pmetric.HistogramDataPointSlice
	switch metric.Type() {
	case pmetric.MetricTypeHistogram:
		dataPoints = metric.Histogram().DataPoints()
		// e.logger.Debug("DP: ", zap.Any("dp len", dataPoints.Len()), zap.Any("dpp", metric.Histogram().DataPoints().At(0)))
	case pmetric.MetricTypeExponentialHistogram:
		// dataPoints = metric.ExponentialHistogram().DataPoints()
		// e.logger.Debug("DP: ", zap.Any("dp len", dataPoints.Len()), zap.Any("dpp", metric.ExponentialHistogram().DataPoints().At(0)))
	default:
		return metricRecords
	}

	for i := 0; i < dataPoints.Len(); i++ {
		dp := dataPoints.At(i)

		buckets := dp.BucketCounts()
		bounds := dp.ExplicitBounds()

		// padding of bounds from left by spaces to make the bucket names sortable
		boundMaxLen := len(strconv.FormatFloat(bounds.At(bounds.Len()-1), 'f', -1, 64))
		lowFormatString := fmt.Sprintf("(-inf,%% %ds]", boundMaxLen)
		midFormatString := fmt.Sprintf("(%% %ds,%% %ds]", boundMaxLen, boundMaxLen)
		highFormatString := fmt.Sprintf("(%% %ds,+inf)", boundMaxLen)

		for bckt := 0; bckt < buckets.Len(); bckt++ {
			metricRecord := analyticsRecord{}
			bucketName := ""
			if bckt == 0 {
				bucketName = fmt.Sprintf(lowFormatString, strconv.FormatFloat(bounds.At(bckt), 'f', -1, 64))
			} else if bckt == buckets.Len()-1 {
				bucketName = fmt.Sprintf(highFormatString, strconv.FormatFloat(bounds.At(bckt-1), 'f', -1, 64))
			} else {
				bucketName = fmt.Sprintf(midFormatString, strconv.FormatFloat(bounds.At(bckt-1), 'f', -1, 64), strconv.FormatFloat(bounds.At(bckt), 'f', -1, 64))
			}
			metricRecord["bucket"] = bucketName
			bucketValue := buckets.At(bckt)

			// copy resource attributes
			for name, value := range resource.Attributes().AsRaw() {
				nName := normalizedName(name)
				metricRecord[RESOURCE_PREFIX+nName] = fmt.Sprintf("%s", value)
			}
			if scope.Name() != "" {
				metricRecord[SCOPE_PREFIX+"name"] = scope.Name()
			}
			if scope.Version() != "" {
				metricRecord[SCOPE_PREFIX+"version"] = scope.Version()
			}
			metricRecord["metricName"] = metric.Name()
			metricRecord["metricUnit"] = metric.Unit()
			metricRecord["metricType"] = metric.Type().String()
			metricRecord["metricStartTimestamp"] = dp.StartTimestamp().AsTime().UnixMilli()
			metricRecord["metricTimestamp"] = dp.Timestamp().AsTime().UnixMilli()

			metricRecord["metricValue"] = bucketValue
			dpAttributes := dp.Attributes().AsRaw()
			for name, value := range dpAttributes {
				nName := normalizedName(name)
				metricRecord[LABEL_PREFIX+nName] = fmt.Sprintf("%s", value)
			}
			if metric.Histogram().AggregationTemporality() == pmetric.AggregationTemporalityCumulative {
				cacheKey := getCacheKey(resource.Attributes().AsRaw(), metric.Name())
				cacheKey = getCacheKey(dp.Attributes().AsRaw(), cacheKey)
				metricCacheMutex.Lock()
				if cacheValue, ok := metricCache[cacheKey]; ok {
					if cacheUpdate, hasRecord := metricCacheUpdates[cacheKey]; hasRecord {
						if time.Now().Compare(cacheUpdate.Add(90*time.Second)) > 0 {
							// missed reporting interval - skip this one and start from scratch
							metricCache[cacheKey] = float64(bucketValue)
							metricCacheUpdates[cacheKey] = time.Now()
						} else {
							value := float64(bucketValue) - cacheValue
							metricCache[cacheKey] = float64(bucketValue)
							metricCacheUpdates[cacheKey] = time.Now()
							metricRecord["metricValue"] = value
							// Histogram bucket, Cummulative -> difference over interval
							metricRecords = append(metricRecords, metricRecord)
						}
					} else {
						// cache-update-miss -> new entry + skip this reporting interval
						metricCache[cacheKey] = float64(bucketValue)
						metricCacheUpdates[cacheKey] = time.Now()
					}
				} else {
					// cache-miss -> new entry + skip this reporting interval
					metricCache[cacheKey] = float64(bucketValue)
					metricCacheUpdates[cacheKey] = time.Now()
				}
				metricCacheMutex.Unlock()
			} else {
				// Histogram bucket, Delta -> direct metric value
				metricRecords = append(metricRecords, metricRecord)
			}
		}
	}
	return metricRecords
}

func (e *metricsExporter) getMetricTreeRecords(metricsConfig MetricsConfig, metricRecords []analyticsRecord) []MetricRecord {
	rules := metricsConfig.Rules
	metricTreeRecords := []MetricRecord{}

	for _, metric := range metricRecords {
		for _, rule := range rules {
			if e.matchesConditions(metric, rule.MatchConditions) {
				metricPath, err := e.getMetricPath(metric, &rule)
				if err != nil {
					e.logger.Sugar().Errorf("Cannot process metric path template %s - %v", rule.PathTemplate, err)
					break
				}
				valueFloat := metric["metricValue"] // type float
				value := int64(math.Round(valueFloat.(float64)))
				aggregatorType := rule.AggregatorType
				if aggregatorType == "" {
					aggregatorType = "AVERAGE"
				}
				metricTreeRecords = append(metricTreeRecords, MetricRecord{
					MetricName:     metricsConfig.Prefix + "|" + metricPath,
					AggregatorType: aggregatorType,
					Value:          value,
				})
				// first match -> skip rest
				break
			}
		}
	}

	return metricTreeRecords
}

func (e *metricsExporter) matchesConditions(metric analyticsRecord, conditions []MatchConditions) bool {

	for _, condition := range conditions {
		if condition.MetricName != nil {
			metricName := metric["metricName"]
			return metricName == *condition.MetricName
		}
		if condition.Exists != nil {
			shouldExist := *condition.Exists
			if condition.Attribute != nil {
				_, exists := metric[*condition.Attribute]
				return shouldExist == exists
			}
			// never should get here, but to be sure...
			return true
		}
		if condition.Equals != nil {
			valueToCheck, ok := metric[*condition.Attribute]
			if !ok {
				return false
			}
			return *condition.Equals == valueToCheck
		}
		if condition.NotEquals != nil {
			valueToCheck, ok := metric[*condition.Attribute]
			if !ok {
				return false
			}
			return *condition.NotEquals != valueToCheck
		}
	}
	// should get here only if list of conditions is empty
	return true
}

type TemplateParams struct {
	A map[string]any
}

func (e *metricsExporter) getMetricPath(metric analyticsRecord, rule *Rule) (string, error) {

	tmpl := rule.pathTemplateParsed
	var err error
	if tmpl == nil {
		tmpl, err = template.New(fmt.Sprintf("tmpl-%s", rule.Description)).Parse(rule.PathTemplate)
		if err != nil {
			return "", fmt.Errorf("Cannot parse pathTemplate %s - %v", rule.PathTemplate, err)
		}
		rule.pathTemplateParsed = tmpl
	}

	templateParams := TemplateParams{
		A: metric,
	}
	buff := new(bytes.Buffer)
	err = tmpl.Execute(buff, templateParams)
	if err != nil {
		return "", err
	}
	return buff.String(), nil
}

func (e *metricsExporter) postMetricTreeRecords(metricTreeRecords []MetricRecord) {
	buff, err := json.Marshal(metricTreeRecords)
	if err != nil {
		e.logger.Sugar().Errorf("Error formatting metric tree records: %v", err)
		return
	}
	records := string(buff)

	if e.cfg.Metrics.LogMetricRecords {
		e.logger.Sugar().Infof("METRIC TREE RECORDS: %s", records)
	}

	if e.cfg.Metrics.Url != "" {
		request, err := http.NewRequest("POST", e.cfg.Metrics.Url, bytes.NewBuffer(buff))
		if err != nil {
			e.logger.Sugar().Errorf("Error building metric tree records to machine agent: %v", err)
			return
		}
		request.Header.Add("Content-Type", "application/json")

		client := &http.Client{}
		response, err := client.Do(request)
		if err != nil {
			e.logger.Sugar().Errorf("Error posting metric tree records to machine agent: %v", err)
			return
		}
		defer response.Body.Close()
		if response.StatusCode >= 400 {
			e.logger.Sugar().Errorf("Error posting metric tree records to machine agent - status: %d", response.StatusCode)
			return
		}
	}
}

func (e *metricsExporter) appdMetric() {

}

func getCacheKey(attrs map[string]any, appendTo string) string {
	cacheKey := appendTo
	attrsNames := []string{}
	for name := range attrs {
		attrsNames = append(attrsNames, name)
	}
	sort.Strings(attrsNames)
	for _, name := range attrsNames {
		cacheKey += "\x01" + fmt.Sprintf("%v", attrs[name])
	}
	return cacheKey
}

func normalizedName(in string) string {
	// string.replace(/[&\/\\#,+()$~%.'":*?<>{}]/g,'_');
	return strings.ReplaceAll(in, ".", "_")
}
