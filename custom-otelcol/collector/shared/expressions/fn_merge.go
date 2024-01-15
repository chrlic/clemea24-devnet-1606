package expressions

import (
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
)

var stringMergeFunction = cel.Function("merge",
	cel.Overload("merge_list_list_string",
		[]*cel.Type{cel.ListType(cel.StringType), cel.ListType(cel.IntType), cel.StringType},
		cel.StringType,
		mergeFunctionImpl,
	),
)

var stringMergeMemberFunction = cel.Function("merge",
	cel.MemberOverload("list_merge_list_string",
		[]*cel.Type{cel.ListType(cel.StringType), cel.ListType(cel.IntType), cel.StringType},
		cel.StringType,
		mergeFunctionImpl,
	),
)

var mergeFunctionImpl = cel.FunctionBinding(func(args ...ref.Val) ref.Val {
	result := ""
	parts, ok := args[0].(traits.Lister)
	if !ok {
		return types.NewErr("invalid operand of type '%v' - should be list of strings", args[0].Type())
	}
	indexes, ok := args[1].(traits.Lister)
	if !ok {
		return types.NewErr("invalid operand of type '%v' - should be list of ints", args[1].Type())
	}
	sep, ok := args[2].Value().(string)
	if !ok {
		return types.NewErr("invalid operand of type '%v' - a string", args[2].Type())
	}

	iter := indexes.Iterator()
	for iter.HasNext().Value().(bool) {
		i := iter.Next()
		// fmt.Printf("Got here: %v\n", args)
		index := i.Value().(int64)
		// fmt.Printf("Got here: %d -> %v\n", index, args)
		if index < parts.Size().Value().(int64) {
			part, ok := parts.Get(types.Int(index)).Value().(string)
			if !ok {
				return types.NewErr("invalid list member of type '%v' - should be a string", args[0].Type())
			}
			result += part + sep
		}
	}
	if len(result) > 0 {
		result = result[:len(result)-len(sep)]
	}
	return types.String(result)
})
