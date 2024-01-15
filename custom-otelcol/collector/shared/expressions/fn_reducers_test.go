package expressions

import "testing"

func TestReducerSumFunc(t *testing.T) {
	REDUCER_NAME := "testReducer"
	env := ExpressionEnvironment{}
	err := env.InitEnv(env.initLogger("debug"), nil)
	if err != nil {
		t.Fatalf("Cannot initialize expressions - %v", err)
	}

	args := map[string]interface{}{}

	env.InitReducerMap(REDUCER_NAME)
	env.AddValueToReducerMap(REDUCER_NAME, 4.0)
	env.AddValueToReducerMap(REDUCER_NAME, 8.0)
	env.AddValueToReducerMap(REDUCER_NAME, 10.0)
	ret, err := env.EvaluateExpression(`sumReducer(reducerMap("`+REDUCER_NAME+`"))`, args)
	if err != nil {
		t.Fatalf("Cannot compile test expression - %v", err)
	}
	sum, ok := (*ret).Value().(float64)
	if !ok {
		t.Fatalf("Merge returned invalid type of %T", (*ret).Value())
	}
	expect := 22.0
	if sum != expect {
		t.Fatalf("expected: %v != actual: %v", expect, sum)
	}
}

func TestReducerSumMacroFunc(t *testing.T) {
	REDUCER_NAME := "testReducer"
	env := ExpressionEnvironment{}
	err := env.InitEnv(env.initLogger("debug"), nil)
	if err != nil {
		t.Fatalf("Cannot initialize expressions - %v", err)
	}

	args := map[string]interface{}{}

	env.InitReducerMap(REDUCER_NAME)
	env.AddValueToReducerMap(REDUCER_NAME, 4.0)
	env.AddValueToReducerMap(REDUCER_NAME, 8.0)
	env.AddValueToReducerMap(REDUCER_NAME, 10.0)
	ret, err := env.EvaluateExpression(`reducerMap("`+REDUCER_NAME+`").map(x, x+1.0).sumReducer()`, args)
	if err != nil {
		t.Fatalf("Cannot compile test expression - %v", err)
	}
	sum, ok := (*ret).Value().(float64)
	if !ok {
		t.Fatalf("Merge returned invalid type of %T", (*ret).Value())
	}
	expect := 25.0
	if sum != expect {
		t.Fatalf("expected: %v != actual: %v", expect, sum)
	}
}

func TestReducerCountFunc(t *testing.T) {
	REDUCER_NAME := "testReducer"
	env := ExpressionEnvironment{}
	err := env.InitEnv(env.initLogger("debug"), nil)
	if err != nil {
		t.Fatalf("Cannot initialize expressions - %v", err)
	}

	args := map[string]interface{}{}

	env.InitReducerMap(REDUCER_NAME)
	env.AddValueToReducerMap(REDUCER_NAME, 4.0)
	env.AddValueToReducerMap(REDUCER_NAME, 8.0)
	env.AddValueToReducerMap(REDUCER_NAME, 10.0)
	ret, err := env.EvaluateExpression(`countReducer(reducerMap("`+REDUCER_NAME+`"))`, args)
	if err != nil {
		t.Fatalf("Cannot compile test expression - %v", err)
	}
	sum, ok := (*ret).Value().(int64)
	if !ok {
		t.Fatalf("Merge returned invalid type of %T", (*ret).Value())
	}
	expect := int64(3)
	if sum != expect {
		t.Fatalf("expected: %v != actual: %v", expect, sum)
	}
}

func TestReducerAvgFunc(t *testing.T) {
	REDUCER_NAME := "testReducer"
	env := ExpressionEnvironment{}
	err := env.InitEnv(env.initLogger("debug"), nil)
	if err != nil {
		t.Fatalf("Cannot initialize expressions - %v", err)
	}

	args := map[string]interface{}{}

	env.InitReducerMap(REDUCER_NAME)
	env.AddValueToReducerMap(REDUCER_NAME, 4.0)
	env.AddValueToReducerMap(REDUCER_NAME, 8.0)
	env.AddValueToReducerMap(REDUCER_NAME, 9.0)
	ret, err := env.EvaluateExpression(`avgReducer(reducerMap("`+REDUCER_NAME+`"))`, args)
	if err != nil {
		t.Fatalf("Cannot compile test expression - %v", err)
	}
	sum, ok := (*ret).Value().(float64)
	if !ok {
		t.Fatalf("Merge returned invalid type of %T", (*ret).Value())
	}
	expect := 7.0
	if sum != expect {
		t.Fatalf("expected: %v != actual: %v", expect, sum)
	}
}
