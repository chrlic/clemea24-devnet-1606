package expressions

import (
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
)

var listFlattenFunction = cel.Function("flatten",
	cel.Overload("flatten_list",
		[]*cel.Type{cel.ListType(cel.AnyType)},
		cel.ListType(cel.StringType),
		listFlattenFunctionImpl,
	),
)

var listFlattenMemberFunction = cel.Function("flatten",
	cel.MemberOverload("list_flatten",
		[]*cel.Type{cel.ListType(cel.AnyType)},
		cel.ListType(cel.StringType),
		listFlattenFunctionImpl,
	),
)

var listFlattenFunctionImpl = cel.FunctionBinding(func(args ...ref.Val) ref.Val {
	lists, ok := args[0].(traits.Lister)
	if !ok {
		return types.NewErr("invalid operand of type '%v' - should be []any", args[0].Type())
	}

	slice := []any{}
	iter := lists.Iterator()
	for iter.HasNext().Value().(bool) {
		i := iter.Next()
		slice = append(slice, i.Value())
	}
	sliceStr := flattenSliceStr(slice)
	return types.NewStringList(StringAdapter{}, sliceStr)
})
