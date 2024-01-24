package jsonscraper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/antchfx/jsonquery"
	contextdb "github.com/chrlic/otelcol-cust/collector/shared/contextdb"
	expr "github.com/chrlic/otelcol-cust/collector/shared/expressions"
	"go.uber.org/zap"
)

type ScraperClient interface {
	// method = GET, POST
	// url is full URL https://host:port/path?query
	// payload is a body for POST method
	Login() error
	Logout() error
	DoRequest(method string, url string, payload *string) (string, error)
}

type Scraper struct {
	name           string
	logger         *zap.Logger
	config         Config
	interval       int
	scrapperClient ScraperClient
	emitter        Emitter
	db             *contextdb.ContextDb
	expr           *expr.ExpressionEnvironment
}

func NewScraper(name string, logger *zap.Logger, scrapperClient ScraperClient, emitter Emitter, config Config, interval int, db *contextdb.ContextDb) Scraper {

	expr := expr.ExpressionEnvironment{
		Logger: logger,
	}
	expr.InitEnv(logger, db)

	return Scraper{
		logger:         logger,
		config:         config,
		interval:       interval,
		scrapperClient: scrapperClient,
		expr:           &expr,
		name:           name,
		emitter:        emitter,
		db:             db,
	}
}

func (g *Scraper) Run() {

	g.logger.Info("Starting scrapper...\n")
	g.logger.Info("Config: ", zap.Any("config", g.config))

	/*
		err := contextdb.Test(g.logger)
		if err != nil {
			g.logger.Sugar().Errorf("DB Error: %v", err)
		}

		return
	*/

	ticker := time.NewTicker(time.Duration(g.interval) * time.Second)
	quit := make(chan struct{})
	go func() {
		err := g.scrape()
		if err != nil {
			g.logger.Sugar().Errorf("Error scrapping %s: ", g.name, zap.Error(err))
		}
		for {
			select {
			case <-ticker.C:
				go func() {
					err := g.scrape()
					if err != nil {
						g.logger.Sugar().Errorf("Error scrapping %s: ", g.name, zap.Error(err))
					}
				}()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func (g *Scraper) scrape() error {

	defer func() {
		if r := recover(); r != nil {
			g.logger.Sugar().Errorf("***** Recovered in scrape: %v\n%s\n", r, string(debug.Stack()))
		}
	}()

	err := g.scrapperClient.Login()
	if err != nil {
		g.logger.Sugar().Infof("Not logged in - %v", err)
		return err
	}

	for _, q := range g.config.Queries {
		g.scrapeOneQuery(q)
	}
	g.scrapperClient.Logout()

	return nil
}

func (g *Scraper) scrapeOneQuery(query *Query) error {

	scrapeContext := newScaperContext()
	scrapeContext.push()

	// memory leak prevention
	defer func() {
		scrapeContext.cleanup()
	}()

	for _, attr := range query.Resource.Attributes {
		scrapeContext.addRsrcAttr(attr.Name, attr.Value)
	}
	scrapeContext.setScope(query.Scope)

	err := g.runRuleNew(&query.Rules, nil, &scrapeContext)
	if err != nil {
		g.logger.Sugar().Errorf("Error scrapping query %s - %v", query.Name, err)
	}

	return nil

}

func (g *Scraper) runRuleNew(rule *Rule, doc *jsonquery.Node, scContext *scraperContext) error {
	scContext.push()
	defer func() {
		scContext.pop()
	}()

	var currDoc *jsonquery.Node
	var err error

	// evaluate parameters from current doc
	//if doc != nil { // nil is with initial call before any query
	g.evaluateParameters(rule.QueryParameters, doc, scContext)
	//}

	switch rule.Query {
	case "":
		currDoc = doc
	case "LOOP_ITEM":
		currDoc = doc
	default:
		url := g.fillParams(rule.Query, scContext)
		g.logger.Sugar().Debugf("QUERY URL: %s", url)

		if rule.QueryPostData == nil {
			currDoc, err = g.getDataFromService("GET", url, nil)
		} else {
			postData := g.fillParams(*(rule.QueryPostData), scContext)
			currDoc, err = g.getDataFromService("POST", url, &postData)
		}
		if err != nil {
			g.logger.Sugar().Errorf("Cannot get data from service %s - %v", rule.Query, err)
			return err
		}
	}

	// this should only happen if root rule has no query
	if currDoc == nil {
		g.logger.Sugar().Debugf("Nil document where not expected...")
		return nil
	}

	scContext.setDoc(currDoc)

	// now get resource attributes, item attributes, and parameters
	g.evaluateResourceAttributes(rule.ResourceAttributes, currDoc, scContext)
	g.evaluateItemAttributes(rule.ItemAttributes, currDoc, scContext)

	// setup reducers if any
	for _, reducer := range rule.Reducers {
		g.expr.InitReducerMap(reducer)
	}

	// Select returns an arrays of jsonquery Nodes from current document. It comes together with "ForEach",
	// which then processes the items one by one
	if rule.Select != "" && rule.ForEach != nil {
		g.logger.Sugar().Debugf("Going into ForEach - Select %s ForEach: %v", rule.Select, rule.ForEach)

		list := jsonquery.Find(currDoc, rule.Select)
		g.logger.Sugar().Debugf("Selected length %d\n%v", len(list), list)

		for _, subDoc := range list {
			err = g.runRuleNew(rule.ForEach, subDoc, scContext)
			if err != nil {
				g.logger.Sugar().Errorf("Rule processing failed %v: %v - %v", rule.ForEach, scContext, err)
			}
		}
	}

	// process reducer maps if any
	for _, rMap := range rule.ReducerMaps {
		if rMap.Name == "" {
			continue
		}
		if rMap.Value != nil {
			g.expr.AddValueToReducerMap(rMap.Name, *rMap.Value)
			continue
		}
		if rMap.ValueFrom != nil {
			valueAny, err := g.evaluateValueFrom(currDoc, *rMap.ValueFrom, scContext)
			if err != nil {
				g.logger.Sugar().Errorf("Error evaluating expression for reducer map %s: %s - %v", rMap.Name, *rMap.ValueFrom, err)
			}
			// value, ok := valueAny.(float64)
			// if !ok {
			// 	g.logger.Sugar().Errorf("Error return type for expression for reducer map %s: %s - expected float64, actual %T", rMap.Name, *rMap.ValueFrom, valueAny)
			// }
			g.expr.AddValueToReducerMap(rMap.Name, valueAny)
		}
	}

	// process metric and logs emits at this level of rule if present
	if rule.EmitMetric != nil {
		g.processEmitMetrics(rule.EmitMetric, currDoc, scContext)
	}

	if rule.EmitLogs != nil {
		g.processEmitLogs(rule.EmitLogs, currDoc, scContext)
	}

	if rule.EmitDbRecord != nil {
		g.processEmitDbRecord(rule.EmitDbRecord, currDoc, scContext)
	}

	return nil
}

func (g *Scraper) getDataFromService(method string, uri string, payload *string) (*jsonquery.Node, error) {
	response, err := g.scrapperClient.DoRequest(method, uri, payload)
	if err != nil {
		var pld string
		if payload == nil {
			pld = ""
		} else {
			pld = *payload
		}
		return nil, fmt.Errorf("Error in getting data from service %s, method %s, uri %s, payload %s", g.name, method, uri, pld)
	}
	doc, err := jsonquery.Parse(strings.NewReader(response))
	if err != nil {
		return nil, fmt.Errorf("Error in parsing response data from service %s, method %s, uri %s, response %s", g.name, method, uri, response)
	}

	return doc, nil
}

func (g *Scraper) evaluateResourceAttributes(attrs []Attribute, doc *jsonquery.Node, scContext *scraperContext) error {
	for _, a := range attrs {
		if a.Value != "" {
			scContext.addRsrcAttr(a.Name, a.Value)
		} else {
			valueAny, err := g.evaluateValueFrom(doc, a.ValueFrom, scContext)
			if err != nil {
				err = fmt.Errorf("Cannot evaluate expr %s - %v", a.ValueFrom, err)
				g.logger.Sugar().Errorf("%v", err)
				return err
			}
			g.logger.Sugar().Debugf("EVALUATED: %T - %v", valueAny, valueAny)
			scContext.addRsrcAttr(a.Name, valueAny)
		}
	}
	return nil
}

func (g *Scraper) evaluateItemAttributes(attrs []Attribute, doc *jsonquery.Node, scContext *scraperContext) error {
	for _, a := range attrs {
		if a.Value != "" {
			scContext.addItemAttr(a.Name, a.Value)
		} else {
			valueAny, err := g.evaluateValueFrom(doc, a.ValueFrom, scContext)
			if err != nil {
				err = fmt.Errorf("Cannot evaluate expr %s - %v", a.ValueFrom, err)
				g.logger.Sugar().Errorf("%v", err)
				return err
			}
			g.logger.Sugar().Debugf("EVALUATED: %T - %v", valueAny, valueAny)
			scContext.addItemAttr(a.Name, valueAny)
		}
	}
	return nil
}

func (g *Scraper) evaluateParameters(attrs []Attribute, doc *jsonquery.Node, scContext *scraperContext) error {
	for _, a := range attrs {
		if a.Value != "" {
			scContext.addParameter(a.Name, a.Value)
		} else {
			valueAny, err := g.evaluateValueFrom(doc, a.ValueFrom, scContext)
			if err != nil {
				err = fmt.Errorf("Cannot evaluate expr %s - %v", a.ValueFrom, err)
				g.logger.Sugar().Errorf("%v", err)
				return err
			}
			g.logger.Sugar().Debugf("EVALUATED parameter: %T - %v", valueAny, valueAny)
			scContext.addParameter(a.Name, fmt.Sprintf("%s", valueAny))
		}
	}
	return nil
}

func (g *Scraper) processEmitLogs(emitRules []LogEmit, doc *jsonquery.Node, scContext *scraperContext) {
	scContext.push()
	defer func() {
		scContext.pop()
	}()

	g.logger.Sugar().Debugf("Processing log emit rules: %v, doc: %v, ctx: %v, consumer: %v", emitRules, doc, scContext, g.emitter.metricConsumer)

	if g.emitter.logConsumer != nil { // This is a log receiver
		for _, emit := range emitRules {
			if passed, err := g.evaluateFilters(emit.Filters, doc, scContext); !passed || err != nil {
				if err != nil {
					g.logger.Sugar().Errorf("Error evaluating filter - %v", err)
					continue
				}
			} else {
				g.evaluateResourceAttributes(emit.ResourceAttributes, doc, scContext)
				g.evaluateItemAttributes(emit.ItemAttributes, doc, scContext)
				var message string
				var serviceNativeSeverity string
				var timestamp string
				if emit.MessageFrom != "" {
					messageAny, err := g.evaluateValueFrom(doc, emit.MessageFrom, scContext)
					if err != nil {
						g.logger.Sugar().Errorf("Cannot evaluate expr %s - %v", emit.MessageFrom, err)
						return
					}
					message = fmt.Sprintf("%v", messageAny)
				}
				if emit.SeverityFrom != "" {
					serviceNativeSeverityAny, err := g.evaluateValueFrom(doc, emit.SeverityFrom, scContext)
					if err != nil {
						g.logger.Sugar().Errorf("Cannot evaluate expr %s - %v", emit.MessageFrom, err)
						return
					}
					serviceNativeSeverity = fmt.Sprintf("%v", serviceNativeSeverityAny)
				}
				if emit.TimestampFrom != "" {
					timestampAny, err := g.evaluateValueFrom(doc, emit.TimestampFrom, scContext)
					if err != nil {
						g.logger.Sugar().Errorf("Cannot evaluate expr %s - %v", emit.TimestampFrom, err)
						return
					}
					timestamp = fmt.Sprintf("%v", timestampAny)
				}
				g.logger.Sugar().Debugf("Log emit rules: %v, msg: %s, ctx: %v, consumer: %v", emit, message, scContext, g.emitter.metricConsumer)

				g.emitter.EmitLogs(&emit, message, serviceNativeSeverity, timestamp, scContext, g.interval)
			}
		}
	}

}

func (g *Scraper) processEmitMetrics(emitRules []MetricEmit, doc *jsonquery.Node, scContext *scraperContext) {
	scContext.push()
	defer func() {
		scContext.pop()
	}()

	g.logger.Sugar().Debugf("Processing metric emit rules: %v, doc: %v, ctx: %v, consumer: %v", emitRules, doc, scContext, g.emitter.metricConsumer)

	if g.emitter.metricConsumer != nil { // This is a metric receiver
		for _, emit := range emitRules {
			if passed, err := g.evaluateFilters(emit.Filters, doc, scContext); !passed || err != nil {
				if err != nil {
					g.logger.Sugar().Errorf("Error evaluating filter - %v", err)
					continue
				}
			} else {
				g.evaluateResourceAttributes(emit.ResourceAttributes, doc, scContext)
				g.evaluateItemAttributes(emit.ItemAttributes, doc, scContext)
				var value = 0.0
				valueAny, err := g.evaluateValueFrom(doc, emit.ValueFrom, scContext)
				if err != nil {
					g.logger.Sugar().Errorf("Cannot evaluate expr %s - %v", emit.ValueFrom, err)
					return
				}
				valueStr := g.stringifyVal(valueAny)
				if valueStr == "" {
					valueStr = "0"
				}
				value, err = strconv.ParseFloat(valueStr, 64)
				if err != nil {
					g.logger.Sugar().Errorf("Metric value from %s = %s is not number - %v", emit.ValueFrom, valueAny, err)
					return
				}

				g.logger.Sugar().Debugf("Emitting metric emit: %v, val: %v, ctx: %v, interval: %v", emit, value, scContext, g.interval)
				g.emitter.EmitMetrics(&emit, value, scContext, g.interval)
			}
		}
	}
}

func (g *Scraper) processEmitDbRecord(emitRules []DBEmit, doc *jsonquery.Node, scContext *scraperContext) {
	if g.db == nil {
		g.logger.Sugar().Errorf("DB record rule configured, but DB not initialized")
		return
	}

	scContext.push()
	defer func() {
		scContext.pop()
	}()

	g.logger.Sugar().Debugf("Processing db record emit rules: %v, doc: %v, ctx: %v", emitRules, doc, scContext)

	for _, emit := range emitRules {
		if passed, err := g.evaluateFilters(emit.Filters, doc, scContext); !passed || err != nil {
			if err != nil {
				g.logger.Sugar().Errorf("Error evaluating filter - %v", err)
				continue
			}
		} else {
			dbRecord := map[string]any{}
			for _, fld := range emit.Fields {
				var valueAny any
				if fld.Value != "" {
					valueAny = fld.Value
				} else {
					valueAny, err = g.evaluateValueFrom(doc, fld.ValueFrom, scContext)
					if err != nil {
						err = fmt.Errorf("Cannot evaluate expr %s - %v", fld.ValueFrom, err)
						g.logger.Sugar().Errorf("%v", err)
						continue
					}
					g.logger.Sugar().Debugf("EVALUATED: %T - %v", valueAny, valueAny)
				}
				dbRecord[fld.Name] = valueAny
			}

			dbRecordBytes, err := json.Marshal(dbRecord)
			if err != nil {
				g.logger.Sugar().Errorf("Cannot marshall dbRecord %v - %v", dbRecord, err)
				continue
			}
			// fmt.Printf("Emitting DB record %s", string(dbRecordBytes))
			recordNode, err := jsonquery.Parse(bytes.NewReader(dbRecordBytes))
			if err != nil {
				g.logger.Sugar().Errorf("Cannot parse marshalled dbRecord %s - %v", string(dbRecordBytes), err)
				continue
			}

			record := contextdb.ContextRecord{
				Data: recordNode,
			}
			g.db.InsertOrUpdateRecord(emit.DB, &record)
		}
	}

	for _, rule := range emitRules {
		if rule.Dump {
			g.db.Dump(rule.DB)
		}
	}

}

func (g *Scraper) evaluateFilters(filters []Filter, doc *jsonquery.Node, scContext *scraperContext) (bool, error) {
	filtersPassed := true
	for _, f := range filters {
		isValueAny, err := g.evaluateValueFrom(doc, f.Is, &scraperContext{})
		if err != nil {
			return false, fmt.Errorf("Cannot evaluate expr %s - %v", f.Is, err)
		}
		isValue, ok := isValueAny.(bool)
		if !ok {
			return false, fmt.Errorf("Expression evaluated not to bool value %v", isValueAny)
		}
		g.logger.Sugar().Debugf("Filter expression %s evaluated as %t", f.Is, isValue)
		filtersPassed = filtersPassed && isValue
		if !filtersPassed {
			break
		}
	}
	return filtersPassed, nil
}

func (g *Scraper) fillParams(templ string, scContext *scraperContext) string {
	result := templ
	paramsMap := scContext.getParameters()

	for key, value := range paramsMap {
		placeholder := "${" + key + "}"
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}
	return result

	// return g.fillParamsExt(templ, scContext, '{', '}')
}

func (g *Scraper) fillParamsExt(templ string, scContext *scraperContext, startChar byte, endChar byte) (string, error) {

	paramsMap := scContext.getParameters()
	result := ""
	inParam := false
	paramName := ""
	for i := 0; i < len(templ); i++ {
		switch templ[i] {
		case startChar:
			inParam = true
		case endChar:
			inParam = false
			value, ok := paramsMap[paramName]
			if !ok {
				return "", fmt.Errorf("Missing variable %s for template %s", paramName, templ)
			}
			result += fmt.Sprintf("%v", value)
			paramName = ""
		case '\\':
			i++
			result += string(templ[i])
		default:
			if inParam {
				paramName += string(templ[i])
			} else {
				result += string(templ[i])
			}
		}
	}
	return result, nil
}

func (g *Scraper) evalAttribute(attr *Attribute, doc *jsonquery.Node, scrapperContext *scraperContext) string {
	g.logger.Sugar().Debugf("Attribute %v doc %v", attr, doc)
	if attr.ValueFrom != "" && doc != nil {
		valueNode := jsonquery.FindOne(doc, attr.ValueFrom)
		if valueNode == nil {
			g.logger.Sugar().Errorf("Attribute %v returns null value")
			return ""
		} else {
			return fmt.Sprintf("%v", valueNode.Value())
		}
	} else if attr.Value != "" {
		return attr.Value
	} else {
		return ""
	}
}

// scrapeContext related struct/logic

func (g *Scraper) evaluateValueFrom(doc *jsonquery.Node, expr string, scrapeContext *scraperContext) (any, error) {

	defer func() {
		if r := recover(); r != nil {
			g.logger.Sugar().Errorf("Recovered from fatal error %v", r)
		}
	}()

	var value any
	var err error
	switch expr[0] {
	case '=':
		bindings := map[string]interface{}{
			"attr":    scrapeContext.getItemAttrs(),
			"resAttr": scrapeContext.getRsrcAttrs(),
			"params":  scrapeContext.getParameters(),
		}
		value, err = g.expr.EvaluateExpressionWithJqDoc(doc, expr[1:], bindings)
		if err != nil {
			return "", err
		}
		g.logger.Sugar().Debugf("EVALUATE RESULT: %v - %T <= %s", value, value, expr[1:])
	default:
		valRef := jsonquery.FindOne(doc, expr)
		if valRef == nil {
			value = 0
			err = fmt.Errorf("Cannot evaluate expression %s on %v", expr, doc)
		} else {
			value = g.stringifyVal(valRef.Value())
		}
	}

	return value, err
}

func (g *Scraper) stringifyVal(val any) string {
	retval := ""

	switch val.(type) {
	case int, int32, int64:
		retval = fmt.Sprintf("%d", val)
	case float32, float64:
		retval = fmt.Sprintf("%f", val)
	case string:
		retval = val.(string)
	default:
		retval = fmt.Sprintf("%v", val)
	}

	return retval
}
