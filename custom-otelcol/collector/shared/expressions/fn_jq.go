package expressions

import (
	"fmt"

	"github.com/antchfx/jsonquery"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

func (c *ExpressionEnvironment) JqSetDoc(doc *jsonquery.Node) {
	c.JqDoc = doc
	// c.Logger.Sugar().Debugf("DOC SET NULL: %t", c.JqDoc == nil)
	// c.Logger.Sugar().Debugf("DOC %v", c)
}

func (c *ExpressionEnvironment) jqFunctions() []cel.EnvOption {
	functions := []cel.EnvOption{}

	var jqsFunctionImpl = cel.FunctionBinding(func(args ...ref.Val) ref.Val {
		expr, ok := args[0].Value().(string)
		if !ok {
			return types.NewErr("invalid operand of type '%v' - a string", args[0].Type())
		}
		if c.JqDoc == nil {
			return types.NewErr("jsonquery doc is null")
		}

		valPtr := jsonquery.FindOne(c.JqDoc, expr)
		if valPtr == nil {
			return types.String("")
		}
		valueStr := fmt.Sprintf("%s", valPtr.Value())
		return types.String(valueStr)
	})

	var jqasFunctionImpl = cel.FunctionBinding(func(args ...ref.Val) ref.Val {
		expr, ok := args[0].Value().(string)
		if !ok {
			return types.NewErr("invalid operand of type '%v' - a string", args[0].Type())
		}
		if c.JqDoc == nil {
			return types.NewErr("jsonquery doc is null")
		}

		sliceValue := []string{}
		valSlicePtr := jsonquery.Find(c.JqDoc, expr)
		for _, valPtr := range valSlicePtr {
			valueStr := fmt.Sprintf("%s", valPtr.Value())
			sliceValue = append(sliceValue, valueStr)
			c.Logger.Sugar().Debugf("DOC - cycle %v", sliceValue)
		}
		// c.Logger.Sugar().Debugf("DOC - slice %v", sliceValue)
		return types.NewStringList(StringAdapter{}, sliceValue)
	})

	var jqsFunction = cel.Function("jqs",
		cel.Overload("jqs_string",
			[]*cel.Type{cel.StringType},
			cel.StringType,
			jqsFunctionImpl,
		),
	)

	var jqasFunction = cel.Function("jqas",
		cel.Overload("jqas_string",
			[]*cel.Type{cel.StringType},
			cel.ListType(cel.StringType),
			jqasFunctionImpl,
		),
	)

	functions = append(functions, jqsFunction)
	functions = append(functions, jqasFunction)

	return functions
}
