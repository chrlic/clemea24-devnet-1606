package expressions

import (
	"reflect"
	"testing"
)

func TestGrokFunc(t *testing.T) {
	env := ExpressionEnvironment{}
	err := env.InitEnv(env.initLogger("debug"), nil)
	if err != nil {
		t.Fatalf("Cannot initialize expressions - %v", err)
	}

	args := map[string]interface{}{
		"forSplit": "%{ACIPHYIF}",
		"forMerge": []string{"x", "y", "z"},
	}
	ret, err := env.EvaluateExpression(`grok("topology/pod-1/node-101/sys/phys-[eth1/9]","%{ACIPHYIF}")`, args)
	if err != nil {
		t.Fatalf("Cannot compile test expression - %v", err)
	}
	parts, ok := (*ret).Value().(map[string]string)
	if !ok {
		t.Fatalf("Grok returned invalid type of %T", (*ret).Value())
	}
	expect := map[string]string{
		"node": "node-101",
		"pod":  "pod-1",
		"if":   "eth1/9",
	}
	ok = reflect.DeepEqual(parts, expect)
	if !ok {
		t.Fatalf("expected: %v != actual: %v", expect, parts)
	}
}

func TestGrokMethod(t *testing.T) {
	env := ExpressionEnvironment{}
	err := env.InitEnv(env.initLogger("debug"), nil)
	if err != nil {
		t.Fatalf("Cannot initialize expressions - %v", err)
	}

	args := map[string]interface{}{
		"forSplit": "%{ACIPHYIF}",
		"forMerge": []string{"x", "y", "z"},
	}
	ret, err := env.EvaluateExpression(`"topology/pod-1/node-101/sys/phys-[eth1/9]".grok("%{ACIPHYIF}")`, args)
	if err != nil {
		t.Fatalf("Cannot compile test expression - %v", err)
	}
	parts, ok := (*ret).Value().(map[string]string)
	if !ok {
		t.Fatalf("Grok returned invalid type of %T", (*ret).Value())
	}
	expect := map[string]string{
		"node": "node-101",
		"pod":  "pod-1",
		"if":   "eth1/9",
	}
	ok = reflect.DeepEqual(parts, expect)
	if !ok {
		t.Fatalf("expected: %v != actual: %v", expect, parts)
	}
}
