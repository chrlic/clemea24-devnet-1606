package appdmetric

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

type logsExporter struct {
	logger              *zap.Logger
	cfg                 *Config
	appDAnalyticsClient AppDAnalyticsClient
}

var DEFAULT_LOGS_ANALYTICS_SCHEMA = AppDAnalyticsSchema{
	Fields: map[string]string{
		"logTimestamp":         "date",
		"logObservedTimestamp": "date",
		"logMessage":           "string",
		"logSeverity":          "string",
		"logSeverityNumber":    "float",
	},
}

func newLogsExporter(cfg *Config, logger *zap.Logger) (*logsExporter, error) {

	appDAnalyticsClient := AppDAnalyticsClient{
		GlobalAccountName: cfg.Analytics.GlobalAccountName,
		ApiToken:          cfg.Analytics.ApiKey,
		EventServiceUrl:   cfg.Analytics.Url,
		logger:            logger,
	}

	tries := 10

	appDAnalyticsClient.init()
	if cfg.Analytics.InitTable {
		err := appDAnalyticsClient.DeleteSchema(cfg.Analytics.LogsTable)
		if err != nil {
			logger.Sugar().Infof("Could not re-initialize table %s", cfg.Analytics.LogsTable)
		}
		tries = TRIES_SCHEMA_INIT_AFTER_REINIT
	}

	var err error
	for i := 0; i < tries; i++ {
		err = appDAnalyticsClient.CreateSchemaIfNotPresent(cfg.Analytics.LogsTable, &DEFAULT_LOGS_ANALYTICS_SCHEMA)
		if err != nil {
			logger.Sugar().Infof("Could not re-initialize table (retry) %s - %v", cfg.Analytics.LogsTable, err)
			time.Sleep(RETRY_INTERVAL_SECONDS * time.Second)
		} else {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("Cannot verify existing schema %s - %v", cfg.Analytics.LogsTable, err)
	}
	currentSchema, err := appDAnalyticsClient.GetSchema(cfg.Analytics.LogsTable)
	if err != nil {
		return nil, fmt.Errorf("Cannot read existing schema %s - %v", cfg.Analytics.LogsTable, err)
	}
	appDAnalyticsClient.Schemas[cfg.Analytics.LogsTable] = *currentSchema

	return &logsExporter{
		logger:              logger,
		cfg:                 cfg,
		appDAnalyticsClient: appDAnalyticsClient,
	}, nil
}

func (e *logsExporter) start(ctx context.Context, host component.Host) error {

	metricCache = map[string]float64{}
	e.logger.Info("Starting appdmetric exporter\n")
	return nil
}

// shutdown will shut down the exporter.
func (e *logsExporter) shutdown(ctx context.Context) error {

	return nil
}

func (e *logsExporter) pushLogsData(ctx context.Context, ld plog.Logs) error {

	logRecords := analyticsRecords{}

	rls := ld.ResourceLogs()
	for i := 0; i < rls.Len(); i++ {
		rl := rls.At(i)
		resource := rl.Resource()
		ills := rl.ScopeLogs()
		for j := 0; j < ills.Len(); j++ {
			ils := ills.At(j)
			scope := ils.Scope()

			logs := ils.LogRecords()
			for k := 0; k < logs.Len(); k++ {
				lr := logs.At(k)
				logRecords = append(logRecords, e.appdLogRecords(&resource, &scope, &lr)...)
			}
		}
	}

	e.appDAnalyticsClient.EnsureSchema(e.cfg.Analytics.LogsTable, logRecords)
	e.appDAnalyticsClient.PostEventsBundled(e.cfg.Analytics.LogsTable, logRecords)

	return nil
}

func (e *logsExporter) appdLogRecords(resource *pcommon.Resource, scope *pcommon.InstrumentationScope, logRecord *plog.LogRecord) analyticsRecords {
	appdLogRecord := analyticsRecord{}

	appdLogRecord["logTimestamp"] = logRecord.Timestamp().AsTime().UnixMilli()
	appdLogRecord["logObservedTimestamp"] = logRecord.ObservedTimestamp().AsTime().UnixMilli()
	appdLogRecord["logMessage"] = logRecord.Body().AsString()
	appdLogRecord["logSeverity"] = logRecord.SeverityText()
	appdLogRecord["logSeverityNumber"] = logRecord.SeverityNumber()

	for name, value := range resource.Attributes().AsRaw() {
		nName := normalizedName(name)
		appdLogRecord[RESOURCE_PREFIX+nName] = fmt.Sprintf("%s", value)
	}
	if scope.Name() != "" {
		appdLogRecord[SCOPE_PREFIX+"name"] = scope.Name()
	}
	if scope.Version() != "" {
		appdLogRecord[SCOPE_PREFIX+"version"] = scope.Version()
	}
	for name, value := range logRecord.Attributes().AsRaw() {
		nName := normalizedName(name)
		appdLogRecord[LABEL_PREFIX+nName] = fmt.Sprintf("%s", value)
	}

	return analyticsRecords{appdLogRecord}
}
