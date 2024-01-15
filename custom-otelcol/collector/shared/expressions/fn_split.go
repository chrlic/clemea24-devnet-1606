package expressions

import (
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

var stringSplitFunction = cel.Function("split",
	cel.Overload("split_string_string",
		[]*cel.Type{cel.StringType, cel.StringType},
		cel.ListType(cel.StringType),
		splitFunctionImpl,
	),
)

var stringSplitMemberFunction = cel.Function("split",
	cel.MemberOverload("string_split_string",
		[]*cel.Type{cel.StringType, cel.StringType},
		cel.ListType(cel.StringType),
		splitFunctionImpl,
	),
)

var splitFunctionImpl = cel.FunctionBinding(func(args ...ref.Val) ref.Val {
	ss := args[0].Value().(string)
	ssep := args[1].Value().(string)
	parts := strings.Split(ss, ssep)
	// fmt.Printf("Got here: %s, %s, %v\n", ss, ssep, parts)
	return types.NewStringList(StringAdapter{}, parts)
})
