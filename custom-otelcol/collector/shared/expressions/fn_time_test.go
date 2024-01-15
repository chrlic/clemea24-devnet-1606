package expressions

import (
	"testing"
	"time"
)

func TestToUnixMillisFunc(t *testing.T) {
	env := ExpressionEnvironment{}
	err := env.InitEnv(env.initLogger("debug"), nil)
	if err != nil {
		t.Fatalf("Cannot initialize expressions - %v", err)
	}

	args := map[string]interface{}{
		"forSplit": "2023-05-30T15:16:26.896+02:00",
	}
	ret, err := env.EvaluateExpression(`toUnixMillis(forSplit)`, args)
	if err != nil {
		t.Fatalf("Cannot compile test expression - %v", err)
	}
	millis, ok := (*ret).Value().(int64)
	if !ok {
		t.Fatalf("Merge returned invalid type of %T", (*ret).Value())
	}
	expect := int64(1685452586896)
	if millis != expect {
		t.Fatalf("expected: %v != actual: %v", expect, millis)
	}
}

func TestToUnixMillisMethod(t *testing.T) {
	env := ExpressionEnvironment{}
	err := env.InitEnv(env.initLogger("debug"), nil)
	if err != nil {
		t.Fatalf("Cannot initialize expressions - %v", err)
	}

	args := map[string]interface{}{
		"forSplit": "2023-05-30T15:16:26.896+02:00",
	}
	ret, err := env.EvaluateExpression(`forSplit.toUnixMillis()`, args)
	if err != nil {
		t.Fatalf("Cannot compile test expression - %v", err)
	}
	millis, ok := (*ret).Value().(int64)
	if !ok {
		t.Fatalf("Merge returned invalid type of %T", (*ret).Value())
	}
	expect := int64(1685452586896)
	if millis != expect {
		t.Fatalf("expected: %v != actual: %v", expect, millis)
	}
}

func TestTimeNow(t *testing.T) {
	env := ExpressionEnvironment{}
	err := env.InitEnv(env.initLogger("debug"), nil)

	args := map[string]interface{}{}
	ret, err := env.EvaluateExpression(`now()`, args)
	if err != nil {
		t.Fatalf("Cannot compile test expression - %v", err)
	}
	nowString, ok := (*ret).Value().(string)
	if !ok {
		t.Fatalf("Merge returned invalid type of %T", (*ret).Value())
	}
	_, err = time.Parse(time.RFC3339, nowString)
	if err != nil {
		t.Fatalf("invalid date string returned: %s", nowString)
	}
}

func TestFromUnixMillisFunc(t *testing.T) {
	env := ExpressionEnvironment{}
	err := env.InitEnv(env.initLogger("debug"), nil)

	timeNow := time.Now()

	args := map[string]interface{}{
		"doc1": timeNow.UnixMilli(),
	}

	ret, err := env.EvaluateExpression(`fromUnixMillis(doc1)`, args)
	if err != nil {
		t.Fatalf("Cannot compile test expression - %v", err)
	}
	timeString, ok := (*ret).Value().(string)
	if !ok {
		t.Fatalf("Merge returned invalid type of %T", (*ret).Value())
	}
	expect := timeNow.Format(time.RFC3339)
	if timeString != expect {
		t.Fatalf("expected: %v != actual: %v", expect, timeString)
	}
}

func TestFromUnixMillisMethod(t *testing.T) {
	env := ExpressionEnvironment{}
	err := env.InitEnv(env.initLogger("debug"), nil)

	timeNow := time.Now()

	args := map[string]interface{}{
		"doc1": timeNow.UnixMilli(),
	}

	ret, err := env.EvaluateExpression(`doc1.fromUnixMillis()`, args)
	if err != nil {
		t.Fatalf("Cannot compile test expression - %v", err)
	}

	timeString, ok := (*ret).Value().(string)
	if !ok {
		t.Fatalf("Merge returned invalid type of %T", (*ret).Value())
	}
	expect := timeNow.Format(time.RFC3339)
	if timeString != expect {
		t.Fatalf("expected: %v != actual: %v", expect, timeString)
	}
}
