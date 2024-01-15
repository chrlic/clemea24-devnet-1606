package expressions

import (
	"reflect"
	"sort"
	"testing"
)

func TestAciPathFunc(t *testing.T) {
	env := ExpressionEnvironment{}
	err := env.InitEnv(env.initLogger("debug"), nil)
	if err != nil {
		t.Fatalf("Cannot initialize expressions - %v", err)
	}

	args := map[string]interface{}{
		"forSplit": "topology/pod-1/node-201/sys/phys-[eth1/33]",
	}
	ret, err := env.EvaluateExpression("acipath(forSplit)", args)
	if err != nil {
		t.Fatalf("Cannot compile test expression - %v", err)
	}

	parts, ok := (*ret).Value().(map[string]string)
	if !ok {
		t.Fatalf("Split returned invalid type of %T", (*ret).Value())
	}
	expect := map[string]string{
		"path":   "topology/pod-1/node-201/sys/phys",
		"target": "eth1/33",
		"node":   "node-201",
		"nodeId": "201",
		"pod":    "pod-1",
		"podId":  "1",
	}
	ok = reflect.DeepEqual(parts, expect)
	if !ok {
		t.Fatalf("expected: %v != actual: %v", expect, parts)
	}
}

func TestAciPathMethod(t *testing.T) {
	env := ExpressionEnvironment{}
	err := env.InitEnv(env.initLogger("debug"), nil)
	if err != nil {
		t.Fatalf("Cannot initialize expressions - %v", err)
	}

	args := map[string]interface{}{
		"forSplit": "topology/pod-1/node-201/sys/phys-[eth1/33]",
	}
	ret, err := env.EvaluateExpression(`forSplit.acipath()`, args)
	if err != nil {
		t.Fatalf("Cannot compile test expression - %v", err)
	}
	parts, ok := (*ret).Value().(map[string]string)
	if !ok {
		t.Fatalf("Acipath returned invalid type of %T", (*ret).Value())
	}
	expect := map[string]string{
		"path":   "topology/pod-1/node-201/sys/phys",
		"target": "eth1/33",
		"node":   "node-201",
		"nodeId": "201",
		"pod":    "pod-1",
		"podId":  "1",
	}
	ok = reflect.DeepEqual(parts, expect)
	if !ok {
		t.Fatalf("expected: %v != actual: %v", expect, parts)
	}
}

func TestAciNodePathFunc(t *testing.T) {
	env := ExpressionEnvironment{}
	err := env.InitEnv(env.initLogger("debug"), nil)
	if err != nil {
		t.Fatalf("Cannot initialize expressions - %v", err)
	}

	args := map[string]interface{}{
		"forMerge": []string{
			"topology/pod-1/paths-201/sys/phys-[eth1/33]",
			"topology/pod-1/paths-202/sys/phys-[eth1/33]",
			"topology/pod-1/protpaths-203-204/sys/phys-[ACI_CHANNEL]",
			"topology/pod-1/pathgrp-[192.168.130.21]",
		},
	}
	ret, err := env.EvaluateExpression(`acipathnodes(forMerge)`, args)
	if err != nil {
		t.Fatalf("Cannot compile test expression - %v", err)
	}
	parts, ok := (*ret).Value().([]string)
	if !ok {
		t.Fatalf("Acipath returned invalid type of %T", (*ret).Value())
	}
	sort.Strings(parts)
	expect := []string{
		"topology/pod-1/node-201",
		"topology/pod-1/node-202",
		"topology/pod-1/node-203",
		"topology/pod-1/node-204",
	}
	ok = reflect.DeepEqual(parts, expect)
	if !ok {
		t.Fatalf("expected: %v != actual: %v", expect, parts)
	}
}
