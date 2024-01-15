package expressions

import (
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
)

func (c *ExpressionEnvironment) seenFunctions() []cel.EnvOption {
	functions := []cel.EnvOption{}

	var notSeenFunctionImpl = cel.FunctionBinding(func(args ...ref.Val) ref.Val {

		parts, ok := args[0].(traits.Lister)
		if !ok {
			return types.NewErr("invalid operand of type '%v' - should be list of strings", args[0].Type())
		}

		id := ""
		iter := parts.Iterator()
		for iter.HasNext().Value().(bool) {
			i := iter.Next()
			// fmt.Printf("Got here: %v\n", args)
			part := i.Value().(string)
			id = id + part + "\x01"
		}

		_, ok = c.duplicatesCache[id]

		c.duplicatesCache[id] = time.Now()

		return types.Bool(!ok)
	})

	var notSeenFunction = cel.Function("notSeen",
		cel.Overload("notSeen_list_string",
			[]*cel.Type{cel.ListType(cel.StringType)},
			cel.BoolType,
			notSeenFunctionImpl,
		),
	)

	functions = append(functions, notSeenFunction)

	return functions
}
