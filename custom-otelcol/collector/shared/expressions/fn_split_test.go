package expressions

import (
	"reflect"
	"testing"
)

func TestSplitFunc(t *testing.T) {
	env := ExpressionEnvironment{}
	err := env.InitEnv(env.initLogger("debug"), nil)
	if err != nil {
		t.Fatalf("Cannot initialize expressions - %v", err)
	}

	args := map[string]interface{}{
		"forSplit": "a/b/c/d/e",
		"forMerge": []string{"x", "y", "z"},
	}
	ret, err := env.EvaluateExpression(`split(forSplit,"/")`, args)
	if err != nil {
		t.Fatalf("Cannot compile test expression - %v", err)
	}
	parts, ok := (*ret).Value().([]string)
	if !ok {
		t.Fatalf("Split returned invalid type of %T", (*ret).Value())
	}
	expect := []string{"a", "b", "c", "d", "e"}
	ok = reflect.DeepEqual(parts, expect)
	if !ok {
		t.Fatalf("expected: %v != actual: %v", expect, parts)
	}
}

func TestSplitMethod(t *testing.T) {
	env := ExpressionEnvironment{}
	err := env.InitEnv(env.initLogger("debug"), nil)
	if err != nil {
		t.Fatalf("Cannot initialize expressions - %v", err)
	}

	args := map[string]interface{}{
		"forSplit": "a/b/c/d/e",
		"forMerge": []string{"x", "y", "z"},
	}
	ret, err := env.EvaluateExpression(`forSplit.split("/")`, args)
	if err != nil {
		t.Fatalf("Cannot compile test expression - %v", err)
	}
	parts, ok := (*ret).Value().([]string)
	if !ok {
		t.Fatalf("Split returned invalid type of %T", (*ret).Value())
	}
	expect := []string{"a", "b", "c", "d", "e"}
	ok = reflect.DeepEqual(parts, expect)
	if !ok {
		t.Fatalf("expected: %v != actual: %v", expect, parts)
	}
}
