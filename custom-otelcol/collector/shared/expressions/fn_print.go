package expressions

import (
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types/ref"
)

func (c *ExpressionEnvironment) printFunctions() []cel.EnvOption {
	functions := []cel.EnvOption{}

	var printWithPrefixFunctionImpl = cel.FunctionBinding(func(args ...ref.Val) ref.Val {
		prefix := args[0].Value().(string)
		arg := args[1].Value()
		c.Logger.Sugar().Debugf("%s: %v", prefix, arg)

		return args[1]
	})

	var printWithPrefixMemberFunctionImpl = cel.FunctionBinding(func(args ...ref.Val) ref.Val {
		prefix := args[1].Value().(string)
		arg := args[0].Value()
		c.Logger.Sugar().Debugf("%s: %v", prefix, arg)

		return args[0]
	})

	var printFunctionImpl = cel.FunctionBinding(func(args ...ref.Val) ref.Val {
		prefix := args[0].Value().(string)
		arg := args[1].Value()
		c.Logger.Sugar().Debugf("%v", prefix, arg)

		return args[1]
	})

	var printWithPrefixFunction = cel.Function("print",
		cel.Overload("print_string_any",
			[]*cel.Type{cel.StringType, cel.AnyType},
			cel.AnyType,
			printWithPrefixFunctionImpl,
		),
	)
	var printWithPrefixMemberFunction = cel.Function("print",
		cel.MemberOverload("any_print_string",
			[]*cel.Type{cel.AnyType, cel.StringType},
			cel.AnyType,
			printWithPrefixMemberFunctionImpl,
		),
	)

	var printFunction = cel.Function("print",
		cel.Overload("print_any",
			[]*cel.Type{cel.AnyType},
			cel.AnyType,
			printFunctionImpl,
		),
	)

	var printMemberFunction = cel.Function("print",
		cel.MemberOverload("any_print",
			[]*cel.Type{cel.AnyType},
			cel.AnyType,
			printFunctionImpl,
		),
	)

	functions = append(functions, printWithPrefixFunction)
	functions = append(functions, printWithPrefixMemberFunction)
	functions = append(functions, printFunction)
	functions = append(functions, printMemberFunction)

	return functions
}
