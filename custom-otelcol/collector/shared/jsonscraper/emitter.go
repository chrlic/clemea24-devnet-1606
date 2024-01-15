package jsonscraper

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

type Emitter struct {
	metricConsumer    consumer.Metrics
	logConsumer       consumer.Logs
	ctx               context.Context
	logger            *zap.Logger
	severityConvertor func(string) plog.SeverityNumber
}

func NewEmitter(ctx context.Context, logger *zap.Logger, metricConsumer consumer.Metrics, logConsumer consumer.Logs) Emitter {
	// Only one of metric or log consumers should be non nil
	// if both not nil, log error and set both to nil

	if metricConsumer != nil && logConsumer != nil {
		logger.Sugar().Errorf("Emitter - trying to set as both metric and log receiver, that's not possible")
		return Emitter{
			ctx:               ctx,
			logger:            nil,
			metricConsumer:    nil,
			logConsumer:       nil,
			severityConvertor: DefaultSeverityConvertor,
		}
	}
	return Emitter{
		ctx:               ctx,
		logger:            logger,
		metricConsumer:    metricConsumer,
		logConsumer:       logConsumer,
		severityConvertor: DefaultSeverityConvertor,
	}
}

func (e *Emitter) ConsumeMetrics(metricBundle pmetric.Metrics) {
	e.metricConsumer.ConsumeMetrics(e.ctx, metricBundle)
}

func (e *Emitter) ConsumeLogs(logBundle plog.Logs) {
	e.logConsumer.ConsumeLogs(e.ctx, logBundle)
}

func (e *Emitter) SetSeverityConvertor(convertor func(string) plog.SeverityNumber) {
	e.severityConvertor = convertor
}

func (e *Emitter) EmitMetrics(metric *MetricEmit, value float64, scContext *scraperContext, interval int) {
	metrics := pmetric.NewMetrics()
	resourceMetrics := metrics.ResourceMetrics().AppendEmpty()
	resAttrs := resourceMetrics.Resource().Attributes()

	rsrcAttrs := scContext.getRsrcAttrs()
	for n, v := range rsrcAttrs {
		e.upsertAttribute(&resAttrs, n, v)
	}

	scopeMetrics := resourceMetrics.ScopeMetrics().AppendEmpty()
	scopeMetrics.Scope().SetName(scContext.getScope().Name)
	scopeMetrics.Scope().SetVersion(scContext.getScope().Version)

	scopeMetric := scopeMetrics.Metrics().AppendEmpty()

	scopeMetric.SetName(metric.Name)
	scopeMetric.SetDescription(metric.Description)
	scopeMetric.SetUnit(metric.Unit)

	var dp pmetric.NumberDataPoint

	switch metric.Type {
	case Sum:
		scopeMetric.SetEmptySum()
		scopeMetric.Sum().SetIsMonotonic(true)
		scopeMetric.Sum().SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)

		dp = scopeMetric.Sum().DataPoints().AppendEmpty()
	case Gauge:
		scopeMetric.SetEmptyGauge()
		dp = scopeMetric.Gauge().DataPoints().AppendEmpty()
	}

	now := time.Now()
	startTime := now.Add(-time.Duration(interval))
	dp.SetStartTimestamp(pcommon.NewTimestampFromTime(startTime))
	dp.SetTimestamp(pcommon.NewTimestampFromTime(now))

	dp.SetDoubleValue(value)
	itemAttrs := scContext.getItemAttrs()
	for n, v := range itemAttrs {
		dpAttributes := dp.Attributes()
		e.upsertAttribute(&dpAttributes, n, v)
	}

	e.ConsumeMetrics(metrics)

}

func (e *Emitter) EmitLogs(log *LogEmit, message string, severity string, timestamp string, scContext *scraperContext, interval int) {
	out := plog.NewLogs()
	rls := out.ResourceLogs().AppendEmpty()
	rsrcAttrs := scContext.getRsrcAttrs()
	for attrName, attrValue := range rsrcAttrs {
		e.logger.Sugar().Debugf("Setting rsrc attr... %s -> %v", attrName, attrValue)
		rls.Resource().Attributes().PutStr(attrName, fmt.Sprintf("%v", attrValue))
	}
	logSlice := rls.ScopeLogs().AppendEmpty().LogRecords()
	logRecord := logSlice.AppendEmpty()

	itemAttrs := scContext.getItemAttrs()
	for attrName, attrValue := range itemAttrs {
		logRecord.Attributes().PutStr(attrName, fmt.Sprintf("%v", attrValue))
	}
	logRecord.Body().SetStr(message)
	now := time.Now()

	// timestamp format: 2023-05-30T15:16:26.896+02:00
	otelTimestamp, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		e.logger.Sugar().Errorf("Cannot parse date string %s", timestamp)
		otelTimestamp = now
	}

	logRecord.SetTimestamp(pcommon.NewTimestampFromTime(otelTimestamp))
	logRecord.SetObservedTimestamp(pcommon.NewTimestampFromTime(now))

	otelSeverity := e.severityConvertor(severity)
	logRecord.SetSeverityNumber(otelSeverity)
	logRecord.SetSeverityText(severity)

	e.logger.Sugar().Debugf("Flushing logs... %v", out)
	e.ConsumeLogs(out)
}

func (e *Emitter) upsertAttribute(attributeMap *pcommon.Map, attrName string, attrValue any) {

	switch attrValue.(type) {
	case []string:
		inSlice := attrValue.([]string)
		attrSlice := attributeMap.PutEmptySlice(attrName)
		for _, elem := range inSlice {
			attrSlice.AppendEmpty().SetStr(elem)
		}
	case string:
		attributeMap.PutStr(attrName, attrValue.(string))
	default:
		attributeMap.PutStr(attrName, fmt.Sprintf("%T-toStr-%v", attrValue, attrValue))
	}
}

func DefaultSeverityConvertor(severity string) plog.SeverityNumber {
	otelSeverity := plog.SeverityNumberInfo

	switch strings.ToLower(severity) {
	case "info":
		otelSeverity = plog.SeverityNumberInfo
	case "warning":
		otelSeverity = plog.SeverityNumberWarn
	case "minor":
		otelSeverity = plog.SeverityNumberError
	case "major":
		otelSeverity = plog.SeverityNumberError2
	case "critical":
		otelSeverity = plog.SeverityNumberFatal
	default:
		otelSeverity = plog.SeverityNumberInfo
	}
	return otelSeverity
}
