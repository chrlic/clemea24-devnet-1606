package expressions

import (
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
)

var mapHasFieldFunction = cel.Function("hasField",
	cel.Overload("hasfield_list_bool",
		[]*cel.Type{cel.MapType(cel.StringType, cel.AnyType), cel.StringType},
		cel.BoolType,
		hasFieldFunctionImpl,
	),
)

var mapHasFieldMemberFunction = cel.Function("hasField",
	cel.MemberOverload("list_hasfield_bool",
		[]*cel.Type{cel.MapType(cel.StringType, cel.AnyType), cel.StringType},
		cel.BoolType,
		hasFieldFunctionImpl,
	),
)

var hasFieldFunctionImpl = cel.FunctionBinding(func(args ...ref.Val) ref.Val {
	mapInput, ok := args[0].(traits.Mapper)
	if !ok {
		return types.NewErr("invalid operand of type '%v' - should be map[string]any", args[0].Type())
	}
	fieldName, ok := args[1].Value().(string)
	if !ok {
		return types.NewErr("invalid operand of type '%v' - should be a string", args[1].Type())
	}

	result := true
	val := mapInput.Get(types.String(fieldName))
	if val.Type() == types.ErrType {
		result = false
	}

	return types.Bool(result)
})
