package expressions

import (
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

var stringTimeToUnixMillisFunction = cel.Function("toUnixMillis",
	cel.Overload("toUnixMillis_string_int",
		[]*cel.Type{cel.StringType},
		cel.IntType,
		toUnixMillisFunctionImpl,
	),
)

var stringTimeToUnixMillisMemberFunction = cel.Function("toUnixMillis",
	cel.MemberOverload("string_toUnixMillis_int",
		[]*cel.Type{cel.StringType},
		cel.IntType,
		toUnixMillisFunctionImpl,
	),
)

var stringTimeNow = cel.Function("now",
	cel.Overload("now",
		[]*cel.Type{},
		cel.StringType,
		nowFunctionImpl,
	),
)

var millisTimeToStringFunction = cel.Function("fromUnixMillis",
	cel.Overload("fromUnixMillis_int_string",
		[]*cel.Type{cel.IntType},
		cel.StringType,
		fromUnixMillisImpl,
	),
)

var millisTimeToStringMemberFunction = cel.Function("fromUnixMillis",
	cel.MemberOverload("int_fromUnixMillis_string",
		[]*cel.Type{cel.IntType},
		cel.StringType,
		fromUnixMillisImpl,
	),
)

var toUnixMillisFunctionImpl = cel.FunctionBinding(func(args ...ref.Val) ref.Val {
	result := time.Now().UnixMilli()

	timeString, ok := args[0].Value().(string)
	if !ok {
		return types.NewErr("invalid operand of type '%v' - should be a strings", args[0].Type())
	}

	time, err := time.Parse(time.RFC3339, timeString)
	if err != nil {
		return types.NewErr("invalid time string format '%v' - should be RFC3339 compliant ", args[0].Type())
	}

	result = time.UnixMilli()

	return types.Int(result)
})

var nowFunctionImpl = cel.FunctionBinding(func(args ...ref.Val) ref.Val {
	result := time.Now().Format(time.RFC3339)

	return types.String(result)
})

var fromUnixMillisImpl = cel.FunctionBinding(func(args ...ref.Val) ref.Val {
	inputTime := time.UnixMilli(args[0].Value().(int64))
	result := inputTime.Format(time.RFC3339)

	return types.String(result)
})
