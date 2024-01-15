package expressions

import "testing"

func TestMapHasFieldFunc(t *testing.T) {
	env := ExpressionEnvironment{}
	err := env.InitEnv(env.initLogger("debug"), nil)
	if err != nil {
		t.Fatalf("Cannot initialize expressions - %v", err)
	}

	args := map[string]interface{}{
		"forSplit": "a/b/c/d/e",
		"forMerge": []string{"x", "y", "z"},
		"attr": map[string]string{
			"a": "has A",
			"b": "has B",
		},
	}

	ret, err := env.EvaluateExpression(`attr.hasField("a")`, args)
	if err != nil {
		t.Fatalf("Cannot compile test expression - %v", err)
	}
	hasField, ok := (*ret).Value().(bool)
	if !ok {
		t.Fatalf("Merge returned invalid type of %T", (*ret).Value())
	}
	expect := true
	if hasField != expect {
		t.Fatalf("expected: %v != actual: %v", expect, hasField)
	}

	ret, err = env.EvaluateExpression(`attr.hasField("c")`, args)
	if err != nil {
		t.Fatalf("Cannot compile test expression - %v", err)
	}
	hasField, ok = (*ret).Value().(bool)
	if !ok {
		t.Fatalf("Merge returned invalid type of %T", (*ret).Value())
	}
	expect = false
	if hasField != expect {
		t.Fatalf("expected: %v != actual: %v", expect, hasField)
	}

	ret, err = env.EvaluateExpression(`hasField(attr, "a")`, args)
	if err != nil {
		t.Fatalf("Cannot compile test expression - %v", err)
	}
	hasField, ok = (*ret).Value().(bool)
	if !ok {
		t.Fatalf("Merge returned invalid type of %T", (*ret).Value())
	}
	expect = true
	if hasField != expect {
		t.Fatalf("expected: %v != actual: %v", expect, hasField)
	}

	ret, err = env.EvaluateExpression(`hasField(attr, "c")`, args)
	if err != nil {
		t.Fatalf("Cannot compile test expression - %v", err)
	}
	hasField, ok = (*ret).Value().(bool)
	if !ok {
		t.Fatalf("Merge returned invalid type of %T", (*ret).Value())
	}
	expect = false
	if hasField != expect {
		t.Fatalf("expected: %v != actual: %v", expect, hasField)
	}

}
