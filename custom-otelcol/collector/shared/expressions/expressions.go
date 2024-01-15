package expressions

import (
	"encoding/json"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/antchfx/jsonquery"
	"github.com/chrlic/otelcol-cust/collector/shared/contextdb"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types/ref"
	"go.uber.org/zap"
)

type ExpressionEnvironment struct {
	env             *cel.Env
	expressionCache map[string]*cel.Program
	duplicatesCache map[string]time.Time
	reducers        map[string][]ref.Val
	Logger          *zap.Logger
	JqDoc           *jsonquery.Node
	db              *contextdb.ContextDb
	mutex           sync.Mutex
}

func (c *ExpressionEnvironment) InitEnv(logger *zap.Logger, db *contextdb.ContextDb) error {
	envOptions := []cel.EnvOption{}
	envOptions = append(envOptions, cel.Variable("forSplit", cel.StringType))
	envOptions = append(envOptions, cel.Variable("expr1", cel.StringType))
	envOptions = append(envOptions, cel.Variable("doc1", cel.AnyType))
	envOptions = append(envOptions, cel.Variable("forMerge", cel.ListType(cel.StringType)))

	envOptions = append(envOptions, cel.Variable("attr", cel.MapType(cel.StringType, cel.AnyType)))
	envOptions = append(envOptions, cel.Variable("params", cel.MapType(cel.StringType, cel.StringType)))
	envOptions = append(envOptions, cel.Variable("resAttr", cel.MapType(cel.StringType, cel.AnyType)))

	envOptions = append(envOptions, c.jqFunctions()...)
	envOptions = append(envOptions, c.seenFunctions()...)
	envOptions = append(envOptions, c.reducerFunctions()...)
	envOptions = append(envOptions, c.dbFunctions()...)
	envOptions = append(envOptions, c.printFunctions()...)
	envOptions = append(envOptions, stringSplitFunction)
	envOptions = append(envOptions, stringSplitMemberFunction)
	envOptions = append(envOptions, stringMergeFunction)
	envOptions = append(envOptions, stringMergeMemberFunction)
	envOptions = append(envOptions, pathSplitFunction)
	envOptions = append(envOptions, pathSplitMemberFunction)
	envOptions = append(envOptions, pathNodesFunction)
	envOptions = append(envOptions, pathNodesMemberFunction)
	envOptions = append(envOptions, pathParseFunction)
	envOptions = append(envOptions, pathParseMemberFunction)
	envOptions = append(envOptions, stringTimeToUnixMillisFunction)
	envOptions = append(envOptions, stringTimeToUnixMillisMemberFunction)
	envOptions = append(envOptions, stringTimeNow)
	envOptions = append(envOptions, millisTimeToStringFunction)
	envOptions = append(envOptions, millisTimeToStringMemberFunction)
	envOptions = append(envOptions, stringGrokFunction)
	envOptions = append(envOptions, stringGrokMemberFunction)
	envOptions = append(envOptions, mapHasFieldFunction)
	envOptions = append(envOptions, mapHasFieldMemberFunction)
	envOptions = append(envOptions, listFlattenFunction)
	envOptions = append(envOptions, listFlattenMemberFunction)

	env, err := cel.NewEnv(envOptions...)
	if err != nil {
		return err
	}

	c.env = env
	c.expressionCache = map[string]*cel.Program{}
	c.duplicatesCache = map[string]time.Time{}
	c.reducers = map[string][]ref.Val{}
	c.db = db
	c.Logger = logger

	c.initLogger("debug")

	return nil
}

func (c *ExpressionEnvironment) initLogger(level string) *zap.Logger {
	sampleJSON := []byte(fmt.Sprintf(`{
		"level" : "%s",
		"encoding": "json",
		"outputPaths":["stdout"],
		"errorOutputPaths":["stderr"],
		"encoderConfig": {
			"messageKey":"message",
			"levelKey":"level",
			"levelEncoder":"lowercase"
		}
	}`, level))

	var cfg zap.Config

	if err := json.Unmarshal(sampleJSON, &cfg); err != nil {
		panic(err)
	}

	logger, _ := cfg.Build()
	c.Logger = logger

	return logger
}

func (c *ExpressionEnvironment) CompileExpression(expr string) (*cel.Program, error) {
	ast, issues := c.env.Compile(expr)
	// Check iss for compilation errors.
	if issues.Err() != nil {
		c.Logger.Sugar().Errorf("cannot compile >%s< - %v", expr, issues.Err())
		return nil, issues.Err()
	}
	prg, err := c.env.Program(ast)
	if err != nil {
		c.Logger.Sugar().Errorf("cannot build code for >%s< - %v", expr, err)
		return nil, err
	}

	c.expressionCache[expr] = &prg

	return &prg, nil
}

func (c *ExpressionEnvironment) EvaluateExpression(expr string, args map[string]interface{}) (*ref.Val, error) {
	defer func() {
		if r := recover(); r != nil {
			c.Logger.Sugar().Errorf("Recovered from fatal error in expression evaluation %v", r)
			c.Logger.Sugar().Infof("%s", string(debug.Stack()))
		}
	}()

	prg, ok := c.expressionCache[expr]
	if !ok {
		var err error
		prg, err = c.CompileExpression(expr)
		if err != nil {
			c.Logger.Sugar().Errorf("cannot compile expression >%s< with args %v - %v", expr, args, err)
			return nil, err
		}
		c.expressionCache[expr] = prg
	}

	out, _, err := (*prg).Eval(args)
	if err != nil {
		c.Logger.Sugar().Errorf("cannot evaluate expression >%s< with args %v - %v", expr, args, err)
		return nil, err
	}

	return &out, nil
}

func (c *ExpressionEnvironment) EvaluateExpressionWithJqDoc(doc *jsonquery.Node, expr string, bindings map[string]interface{}) (any, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.JqSetDoc(doc)
	val, err := c.EvaluateExpression(expr, bindings)
	if err != nil {
		return "", err
	}
	return (*val).Value(), err
}
