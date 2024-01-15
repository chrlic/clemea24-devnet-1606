package expressions

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/antchfx/jsonquery"
	"github.com/chrlic/otelcol-cust/collector/shared/contextdb"
	"go.uber.org/zap"
)

const SCHEMA_FILE = "test-db-schema.yaml"

func TestDbGetFirst(t *testing.T) {
	// dbGetFirst

	env := ExpressionEnvironment{}
	err := env.InitEnv(env.initLogger("debug"), nil)
	if err != nil {
		t.Fatalf("Cannot initialize expressions - %v", err)
	}

}

func TestDbGetAll(t *testing.T) {
	// dbGetAll

	env := ExpressionEnvironment{}
	err := env.InitEnv(env.initLogger("debug"), nil)
	if err != nil {
		t.Fatalf("Cannot initialize expressions - %v", err)
	}

}

func prepDataSet(t *testing.T, env *ExpressionEnvironment) {
	logger := &zap.Logger{}
	schemaConfig, err := os.ReadFile(SCHEMA_FILE)
	if err != nil {
		t.Errorf("cannot read db schema yaml %s - %v", SCHEMA_FILE, err)
	}

	dbSchemaAbstract, err := contextdb.ParseDbJsonSchema(schemaConfig)
	if err != nil {
		t.Errorf("cannot parse db schema file %s - %v", SCHEMA_FILE, err)
	}

	dbSchema, err := contextdb.GetDbSchema(dbSchemaAbstract)
	if err != nil {
		t.Errorf("cannot convert schema to memdb schema %s - %v", SCHEMA_FILE, err)
	}

	err = env.db.Init(dbSchema, logger)
	if err != nil {
		t.Errorf("cannot init DB %s - %v", SCHEMA_FILE, err)
	}

	t.Logf("Successfully initialized database %v", dbSchemaAbstract)

	var appdData = []map[string]interface{}{
		{"application": "Mockup-App", "tier": "Mock-Tier-1", "node": "node1", "ipv4": []string{"10.133.10.150", "10.134.10.150"}},
		{"application": "Mockup-App", "tier": "Mock-Tier-1", "node": "node2", "ipv4": []string{"10.133.10.151", "10.134.10.151"}},
		{"application": "Mockup-App", "tier": "Mock-Tier-2", "node": "node3", "ipv4": []string{"10.133.10.152", "10.134.10.152"}},
		{"application": "Mockup-App", "tier": "Mock-Tier-2", "node": "node4", "ipv4": []string{"10.133.10.153", "10.134.10.153"}},
		{"application": "Mockup-App", "tier": "Mock-Tier-3", "node": "node5", "ipv4": []string{"10.133.10.154", "10.134.10.154"}},

		{"application": "Mockup-Cont", "tier": "Cont-Tier-1", "node": "cont1", "ipv4": []string{"10.10.10.150"}},
		{"application": "Mockup-Cont", "tier": "Cont-Tier-1", "node": "cont2", "ipv4": []string{"10.10.10.151"}},
		{"application": "Mockup-Cont", "tier": "Cont-Tier-2", "node": "cont3", "ipv4": []string{"10.10.10.152"}},
		{"application": "Mockup-Cont", "tier": "Cont-Tier-2", "node": "cont4", "ipv4": []string{"10.10.10.153"}},
		{"application": "Mockup-Cont", "tier": "Cont-Tier-3", "node": "cont5", "ipv4": []string{"10.10.10.154"}},
	}

	var k8sData = []map[string]interface{}{
		{"nodeName": "wrk1", "nodeIP": "10.133.10.160", "podName": "rand12345678", "ipv4": "10.10.10.150"},
		{"nodeName": "wrk1", "nodeIP": "10.133.10.160", "podName": "rand87654321", "ipv4": "10.10.10.151"},
		{"nodeName": "wrk1", "nodeIP": "10.133.10.160", "podName": "rand11111111", "ipv4": "10.10.10.152"},
		{"nodeName": "wrk1", "nodeIP": "10.133.10.160", "podName": "rand22222222", "ipv4": "10.10.10.153"},
		{"nodeName": "wrk1", "nodeIP": "10.133.10.160", "podName": "rand33333333", "ipv4": "10.10.10.154"},
	}

	for _, a := range appdData {
		jsonDoc, _ := json.Marshal(a)
		jsonQueryDoc, _ := jsonquery.Parse(bytes.NewReader(jsonDoc))
		rec := contextdb.ContextRecord{Data: jsonQueryDoc}
		err = env.db.InsertOrUpdateRecord("appd", &rec)
		if err != nil {
			t.Logf("cannot store %s to table %s - %v", jsonDoc, "appd", err)
		}
	}

	for _, a := range k8sData {
		jsonDoc, _ := json.Marshal(a)
		jsonQueryDoc, _ := jsonquery.Parse(bytes.NewReader(jsonDoc))
		rec := contextdb.ContextRecord{Data: jsonQueryDoc}
		err = env.db.InsertOrUpdateRecord("k8s-pods", &rec)
		if err != nil {
			t.Logf("cannot store %s to table %s - %v", jsonDoc, "appd", err)
		}
	}
}
