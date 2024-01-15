package expressions

import "testing"

func TestMergeFunc(t *testing.T) {
	env := ExpressionEnvironment{}
	err := env.InitEnv(env.initLogger("debug"), nil)
	if err != nil {
		t.Fatalf("Cannot initialize expressions - %v", err)
	}

	args := map[string]interface{}{
		"forSplit": "a/b/c/d/e",
		"forMerge": []string{"x", "y", "z"},
	}
	ret, err := env.EvaluateExpression(`merge(forMerge,[0,2],":")`, args)
	if err != nil {
		t.Fatalf("Cannot compile test expression - %v", err)
	}
	merged, ok := (*ret).Value().(string)
	if !ok {
		t.Fatalf("Merge returned invalid type of %T", (*ret).Value())
	}
	expect := "x:z"
	if merged != expect {
		t.Fatalf("expected: %v != actual: %v", expect, merged)
	}
}

func TestMergeMethod(t *testing.T) {
	env := ExpressionEnvironment{}
	err := env.InitEnv(env.initLogger("debug"), nil)
	if err != nil {
		t.Fatalf("Cannot initialize expressions - %v", err)
	}

	args := map[string]interface{}{
		"forSplit": "a/b/c/d/e",
		"forMerge": []string{"x", "y", "z"},
	}
	ret, err := env.EvaluateExpression(`forMerge.merge([0],":")`, args)
	if err != nil {
		t.Fatalf("Cannot compile test expression - %v", err)
	}
	merged, ok := (*ret).Value().(string)
	if !ok {
		t.Fatalf("Merge returned invalid type of %T", (*ret).Value())
	}
	expect := "x"
	if merged != expect {
		t.Fatalf("expected: %v != actual: %v", expect, merged)
	}
}

func TestSplitMergeChain(t *testing.T) {
	env := ExpressionEnvironment{}
	err := env.InitEnv(env.initLogger("debug"), nil)
	if err != nil {
		t.Fatalf("Cannot initialize expressions - %v", err)
	}

	args := map[string]interface{}{
		"forSplit": "a/b/c/d/e",
		"forMerge": []string{"x", "y", "z"},
	}
	ret, err := env.EvaluateExpression(`split(forSplit,"/").merge([0,3],"::")`, args)
	if err != nil {
		t.Fatalf("Cannot compile test expression - %v", err)
	}
	merged, ok := (*ret).Value().(string)
	if !ok {
		t.Fatalf("Merge returned invalid type of %T", (*ret).Value())
	}
	expect := "a::d"
	if merged != expect {
		t.Fatalf("expected: %v != actual: %v", expect, merged)
	}
}
