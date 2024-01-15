package appdmetric

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"
	"reflect"
	"sync"
	"time"

	"go.uber.org/zap"
)

type AppDAnalyticsClient struct {
	client                *http.Client
	jar                   http.CookieJar
	x_csrf_token          string
	EventServiceUrl       string
	GlobalAccountName     string
	ApiToken              string
	Schemas               map[string]AppDAnalyticsSchema
	bundleCache           analyticsRecords
	bundleCacheLastUpdate time.Time
	bundleCacheLock       sync.Mutex
	bundleSize            int
	bundleFlushTimeoutSec int
	bundleTimer           *time.Timer
	bundleCancelChannel   chan bool
	logger                *zap.Logger
}

type AppDAnalyticsSchema struct {
	Name   string            `json:"eventType"`
	Fields map[string]string `json:"schema"`
}

type AppDAnalyticsSchemaPatches []AppDAnalyticsSchemaPatch

type AppDAnalyticsSchemaPatch struct {
	Add    map[string]string `json:"add"`
	Rename map[string]string `json:"rename"`
}

func (c *AppDAnalyticsClient) GetSchema(name string) (*AppDAnalyticsSchema, error) {

	url := c.EventServiceUrl + "/events/schema/" + name
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		c.logger.Sugar().Errorf("Error creating request for get schema: %v\n", err)
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.appd.events+json;v=2")
	req.Header.Set("X-Events-API-Key", c.ApiToken)
	req.Header.Set("X-Events-API-AccountName", c.GlobalAccountName)

	resp, err := c.client.Do(req)
	if err != nil {
		c.logger.Sugar().Errorf("Error getting schema: %v\n", err)
		return nil, err
	}
	if resp.StatusCode >= 400 {
		c.logger.Sugar().Errorf("Error reading schema get response status: %d\n", resp.StatusCode)
		return nil, fmt.Errorf("Error reading schema get response status: %d\n", resp.StatusCode)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.logger.Sugar().Errorf("Error reading schema get response: %v\n", err)
		return nil, err
	}

	c.logger.Sugar().Debugf("Schema Get response %d: \n%s\n", resp.StatusCode, string(body))
	schema := AppDAnalyticsSchema{}

	err = json.Unmarshal(body, &schema)

	return &schema, nil
}

func (c *AppDAnalyticsClient) CreateSchema(name string, schema *AppDAnalyticsSchema) error {

	url := c.EventServiceUrl + "/events/schema/" + name

	schemaJson, _ := json.MarshalIndent(schema, "", "  ")
	c.logger.Sugar().Infof("Creating schema:  %s -> %s\n", url, schema)

	req, err := http.NewRequest("POST", url, bytes.NewReader([]byte(schemaJson)))
	if err != nil {
		c.logger.Sugar().Errorf("Error creating request for create schema: %v\n", err)
		return err
	}

	req.Header.Set("Content-Type", "application/vnd.appd.events+json;v=2")
	req.Header.Set("Accept", "application/vnd.appd.events+json;v=2")
	req.Header.Set("X-Events-API-Key", c.ApiToken)
	req.Header.Set("X-Events-API-AccountName", c.GlobalAccountName)

	resp, err := c.client.Do(req)
	if err != nil {
		c.logger.Sugar().Errorf("Error creating schema: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.logger.Sugar().Errorf("Error reading schema create response: %v\n", err)
		return err
	}

	c.logger.Sugar().Debugf("Schema Create response: \n%s\n", string(body))
	return nil
}

func (c *AppDAnalyticsClient) CreateSchemaIfNotPresent(name string, schema *AppDAnalyticsSchema) error {
	_, err := c.GetSchema(name)
	if err != nil {
		return c.CreateSchema(name, schema)
	}
	return nil
}

func (c *AppDAnalyticsClient) NeededPatch(name string, curr *AppDAnalyticsSchema, need *AppDAnalyticsSchema) (bool, *AppDAnalyticsSchemaPatches) {

	missing := map[string]string{}
	for n, v := range need.Fields {
		if _, ok := curr.Fields[n]; !ok {
			missing[n] = v
		}
	}

	patches := AppDAnalyticsSchemaPatches{}
	patches = append(patches, AppDAnalyticsSchemaPatch{
		Add:    missing,
		Rename: map[string]string{},
	})

	return len(missing) > 0, &patches
}

func (c *AppDAnalyticsClient) UpdateSchema(name string, curr *AppDAnalyticsSchema, need *AppDAnalyticsSchema) (bool, error) {

	patchNeeded, patches := c.NeededPatch(name, curr, need)

	if !patchNeeded {
		return false, nil
	}

	patchesJson, err := json.Marshal(patches)
	if err != nil {
		c.logger.Sugar().Errorf("Error marshalling patching schema: %v\n", err)
		return false, err
	}

	url := c.EventServiceUrl + "/events/schema/" + name
	req, err := http.NewRequest("PATCH", url, bytes.NewReader([]byte(patchesJson)))
	if err != nil {
		c.logger.Sugar().Errorf("Error creating request for patching schema: %v\n", err)
		return false, err
	}

	req.Header.Set("X-Events-API-Key", c.ApiToken)
	req.Header.Set("X-Events-API-AccountName", c.GlobalAccountName)
	req.Header.Set("Content-Type", "application/vnd.appd.events+json;v=2")
	req.Header.Set("Accept", "application/vnd.appd.events+json;v=2")

	resp, err := c.client.Do(req)
	if err != nil {
		c.logger.Sugar().Errorf("Error creating schema: %v\n", err)
		return false, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.logger.Sugar().Errorf("Error reading schema patch response: %v\n", err)
		return false, err
	}

	c.logger.Sugar().Debugf("Schema Patch response: \n%s\n", string(body))
	return true, nil

	/*
		PATCH http://analytics.api.example.com/events/schema/{schemaName} HTTP/1.1
		X-Events-API-AccountName:<global_account_name>
		X-Events-API-Key:<api_key>
		Content-type: application/vnd.appd.events+json;v=2
		Accept: application/vnd.appd.events+json;v=2

		[
		{
			"add": {
			"newfield": "integer"
			},
			"rename": {
			"oldname": "newname",
			"oldname2": "newname2"
			}
		}
		]
	*/
}

func (c *AppDAnalyticsClient) DeleteSchema(name string) error {

	url := c.EventServiceUrl + "/events/schema/" + name
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		c.logger.Sugar().Errorf("Error creating request for delete schema: %v\n", err)
		return err
	}

	req.Header.Set("X-Events-API-Key", c.ApiToken)
	req.Header.Set("X-Events-API-AccountName", c.GlobalAccountName)

	resp, err := c.client.Do(req)
	if err != nil {
		c.logger.Sugar().Errorf("Error deleting schema: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.logger.Sugar().Errorf("Error reading schema delete response: %v\n", err)
		return err
	}

	c.logger.Sugar().Debugf("Schema Delete response: \n%s\n", string(body))

	return nil
}

func (c *AppDAnalyticsClient) PostEventsBundled(name string, v analyticsRecords) {
	c.bundleCacheLock.Lock()
	// defer c.bundleCacheLock.Unlock()

	if c.bundleTimer != nil {
		c.bundleTimer.Stop()
		select {
		case c.bundleCancelChannel <- true:
		default:
		}
	}

	c.bundleCache = append(c.bundleCache, v...)
	if len(c.bundleCache) > c.bundleSize {
		c.PostEvents(name, c.bundleCache)
		c.bundleCache = analyticsRecords{}
	}
	c.bundleCacheLastUpdate = time.Now()
	c.bundleCacheLock.Unlock()
	c.bundleTimer = time.NewTimer(5 * time.Second)

	go func() {
		select {
		// case <-c.bundleTimer.C:
		case <-time.After(5 * time.Second):
			c.FlushBundle(name)
		case <-c.bundleCancelChannel:
		}
	}()
}

func (c *AppDAnalyticsClient) FlushBundle(name string) {
	c.bundleCacheLock.Lock()
	defer c.bundleCacheLock.Unlock()

	c.PostEvents(name, c.bundleCache)
	c.bundleCache = analyticsRecords{}
}

func (c *AppDAnalyticsClient) PostEvents(name string, v interface{}) error {
	url := c.EventServiceUrl + "/events/publish/" + name
	var events interface{}

	if reflect.TypeOf(v).Kind() == reflect.Slice {
		events = v
	} else {
		events = []interface{}{v}
	}

	buff, err := json.Marshal(events)
	if err != nil {
		c.logger.Sugar().Errorf("Error marshalling event array: %v\n", err)
		return err
	}

	c.logger.Sugar().Debugf("Posting events:  %s -> %s\n", url, string(buff))

	req, err := http.NewRequest("POST", url, bytes.NewReader(buff))
	if err != nil {
		c.logger.Sugar().Errorf("Error creating request for events post: %v\n", err)
		return err
	}

	req.Header.Set("Content-Type", "application/vnd.appd.events+json;v=2")
	req.Header.Set("Accept", "application/vnd.appd.events+json;v=2")
	req.Header.Set("X-Events-API-Key", c.ApiToken)
	req.Header.Set("X-Events-API-AccountName", c.GlobalAccountName)

	resp, err := c.client.Do(req)
	if err != nil {
		c.logger.Sugar().Errorf("Error posting events: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.logger.Sugar().Errorf("Error reading events post response: %v\n", err)
		return err
	}

	c.logger.Sugar().Debugf("Post Events response: \n%s\n", string(body))
	return nil
}

func (c *AppDAnalyticsClient) EnsureSchema(name string, metrics analyticsRecords) {

	fields := map[string]string{}
	for _, rec := range metrics {
		for fld := range rec {
			fields[fld] = "string"
		}
	}
	currSchema := c.Schemas[name]

	updated, err := c.UpdateSchema(name, &(currSchema), &AppDAnalyticsSchema{Fields: fields})
	if err != nil {
		c.logger.Sugar().Debugf("Cannot Update Schema %s - %v\n", name, err)
	}
	if updated {
		currentSchema, err := c.GetSchema(name)
		if err != nil {
			c.logger.Sugar().Debugf("Cannot read existing schema %s - %v", name, err)
		}
		c.Schemas[name] = *currentSchema

	}
}

func (c *AppDAnalyticsClient) RunQuery(name string, v interface{}) error {

	return nil
}

func (c *AppDAnalyticsClient) getSchemaFromStruct(v interface{}) string {

	schemaFields := map[string]string{}
	schema := map[string]map[string]string{}

	vt := reflect.TypeOf(v)
	for i := 0; i < vt.NumField(); i++ {
		field := vt.Field(i)
		name, appdType := c.parseField(field)
		schemaFields[name] = appdType
	}

	schema["schema"] = schemaFields
	buff, _ := json.Marshal(schema)

	return string(buff)
}

func (c *AppDAnalyticsClient) parseField(field reflect.StructField) (string, string) {
	appdType := "string"
	name := ""
	ok := false
	if name, ok = field.Tag.Lookup("json"); ok {
		// fmt.Printf("Fld: %s, Type: %s %v\n", name, field.Type.Kind().String(), field.Type)
		switch field.Type.Kind().String() {
		case "bool":
			appdType = "boolean"
		case "int", "int8", "int16", "int32", "int64",
			"uint", "uint8", "uint16", "uint32", "uint64", "uintptr":
			appdType = "integer"
		case "float32", "float64":
			appdType = "float"
		case "complex64", "complex128", "Array":
			appdType = "object"
		case "chan", "Func", "Interface":
			appdType = "string"
		case "map":
			appdType = "object"
		case "pointer":
			appdType = "integer"
		case "slice":
			appdType = "object"
		case "string":
			appdType = "string"
		case "struct":
			if field.Type.String() == "time.Time" {
				appdType = "date"
			} else {
				appdType = "object"
			}
		case "unsafePointer":
			appdType = "integer"
		default:
			appdType = "string"
		}
	} else {
		fmt.Println("(not specified)")
	}
	return name, appdType
}

func (c *AppDAnalyticsClient) init() {
	var netTransport = http.DefaultTransport.(*http.Transport).Clone()
	netTransport.Dial = (&net.Dialer{
		Timeout: 30 * time.Second,
	}).Dial

	netTransport.TLSHandshakeTimeout = 30 * time.Second
	netTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	//if useProxy {
	//	netTransport.Proxy = http.ProxyURL(proxyUrl)
	//}

	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatalf("Got error while creating cookie jar %s", err.Error())
	}

	// TODO: Let the consumer define the http.Client
	timeout := time.Duration(60 * time.Second)
	httpClient := &http.Client{
		Timeout:   timeout,
		Transport: netTransport,
		Jar:       jar,
	}

	c.client = httpClient
	c.Schemas = make(map[string]AppDAnalyticsSchema)
	c.bundleCache = analyticsRecords{}
	c.bundleCacheLastUpdate = time.Now()
	c.bundleCacheLock = sync.Mutex{}
	c.bundleSize = 20
	c.bundleFlushTimeoutSec = 1
	c.bundleCancelChannel = make(chan bool)

}
