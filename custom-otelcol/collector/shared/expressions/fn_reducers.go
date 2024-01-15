package expressions

import (
	"reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
)

func (c *ExpressionEnvironment) InitReducerMap(reducerName string) {
	c.reducers[reducerName] = []ref.Val{}
}

func (c *ExpressionEnvironment) GetReducerMap(reducerName string) []ref.Val {
	reducerMap, ok := c.reducers[reducerName]
	if !ok {
		reducerMap = []ref.Val{}
	}
	return reducerMap
}

func (c *ExpressionEnvironment) AddValueToReducerMap(reducerName string, value any) error {
	reducerMap, ok := c.reducers[reducerName]
	if !ok {
		reducerMap = []ref.Val{}
	}
	rt := reflect.TypeOf(value)
	switch rt.Kind() {
	case reflect.Slice:
		for _, val := range value.([]any) {
			reducerMap = append(reducerMap, c.anyScalarToCelType(val))
		}

	default:
		reducerMap = append(reducerMap, c.anyScalarToCelType(value))
	}

	c.reducers[reducerName] = reducerMap

	return nil
}

func (c *ExpressionEnvironment) anyToCelType(value any) ref.Val {
	rt := reflect.TypeOf(value)
	switch rt.Kind() {
	case reflect.Slice:
		parts := []ref.Val{}
		for _, val := range value.([]any) {
			parts = append(parts, c.anyScalarToCelType(val))
		}
		return types.NewRefValList(AnyAdapter{}, parts)
	default:
		return c.anyScalarToCelType(value)
	}
}

func (c *ExpressionEnvironment) anyScalarToCelType(value any) ref.Val {
	switch value.(type) {
	case float64:
		return types.Double(value.(float64))
	case float32:
		return types.Double(value.(float32))
	case int:
		return types.Int(value.(int))
	case int16:
		return types.Int(value.(int16))
	case int32:
		return types.Int(value.(int32))
	case int64:
		return types.Int(value.(int64))
	case string:
		return types.String(value.(string))
	case bool:
		return types.Bool(value.(bool))
	}
	return types.Double(0)
}

func (c *ExpressionEnvironment) reducerFunctions() []cel.EnvOption {
	functions := []cel.EnvOption{}

	var sumReducerFunctionImpl = cel.FunctionBinding(func(args ...ref.Val) ref.Val {
		result := 0.0
		values, ok := args[0].(traits.Lister)
		if !ok {
			return types.NewErr("invalid operand of type '%v' - should be a list of double", args[0].Type())
		}

		iter := values.Iterator()
		for iter.HasNext().Value().(bool) {
			i := iter.Next()
			// fmt.Printf("Got here: %v\n", args)
			value := i.Value().(float64)
			result += value
		}

		return types.Double(result)
	})

	var countReducerFunctionImpl = cel.FunctionBinding(func(args ...ref.Val) ref.Val {

		values, ok := args[0].(traits.Lister)
		if !ok {
			return types.NewErr("invalid operand of type '%v' - should be a list of double", args[0].Type())
		}

		return values.Size()
	})

	var avgReducerFunctionImpl = cel.FunctionBinding(func(args ...ref.Val) ref.Val {
		result := 0.0
		accum := 0.0
		counter := 0.0
		values, ok := args[0].(traits.Lister)
		if !ok {
			return types.NewErr("invalid operand of type '%v' - should be a list of double", args[0].Type())
		}

		iter := values.Iterator()
		for iter.HasNext().Value().(bool) {
			i := iter.Next()
			// fmt.Printf("Got here: %v\n", args)
			value := i.Value().(float64)
			accum += value
			counter++
		}

		if counter != 0 {
			result = accum / counter
		}

		return types.Double(result)
	})

	var reducerMapFunctionImpl = cel.FunctionBinding(func(args ...ref.Val) ref.Val {

		reducerName, ok := args[0].Value().(string)
		if !ok {
			return types.NewErr("invalid operand of type '%v' - should be a string", args[0].Type())
		}

		reducerMap := c.GetReducerMap(reducerName)

		return types.NewDynamicList(AnyAdapter{}, reducerMap)
	})

	var sumReducerFunction = cel.Function("sumReducer",
		cel.Overload("sumReducer_list_double",
			[]*cel.Type{cel.ListType(cel.DoubleType)},
			cel.DoubleType,
			sumReducerFunctionImpl,
		),
	)

	var sumReducerMemberFunction = cel.Function("sumReducer",
		cel.MemberOverload("list_sumReducer_double",
			[]*cel.Type{cel.ListType(cel.DoubleType)},
			cel.DoubleType,
			sumReducerFunctionImpl,
		),
	)

	var countReducerFunction = cel.Function("countReducer",
		cel.Overload("countReducer_list_double",
			[]*cel.Type{cel.ListType(cel.DoubleType)},
			cel.IntType,
			countReducerFunctionImpl,
		),
	)

	var countReducerMemberFunction = cel.Function("countReducer",
		cel.MemberOverload("list_countReducer_double",
			[]*cel.Type{cel.ListType(cel.DoubleType)},
			cel.IntType,
			countReducerFunctionImpl,
		),
	)

	var avgReducerFunction = cel.Function("avgReducer",
		cel.Overload("avgReducer_list_double",
			[]*cel.Type{cel.ListType(cel.DoubleType)},
			cel.DoubleType,
			avgReducerFunctionImpl,
		),
	)

	var avgReducerMemberFunction = cel.Function("avgReducer",
		cel.MemberOverload("list_avgReducer_double",
			[]*cel.Type{cel.ListType(cel.DoubleType)},
			cel.DoubleType,
			avgReducerFunctionImpl,
		),
	)

	// var joinReducerMemberFunction = cel.Function("joinReducer",
	// 	cel.MemberOverload("list_avgReducer_double",
	// 		[]*cel.Type{cel.ListType(cel.DoubleType)},
	// 		cel.DoubleType,
	// 		avgReducerFunctionImpl,
	// 	),
	// )

	var reducerMapFunction = cel.Function("reducerMap",
		cel.Overload("reducerMap_string_list",
			[]*cel.Type{cel.StringType},
			cel.ListType(cel.AnyType),
			reducerMapFunctionImpl,
		),
	)

	functions = append(functions, sumReducerFunction)
	functions = append(functions, countReducerFunction)
	functions = append(functions, avgReducerFunction)
	functions = append(functions, sumReducerMemberFunction)
	functions = append(functions, countReducerMemberFunction)
	functions = append(functions, avgReducerMemberFunction)
	functions = append(functions, reducerMapFunction)

	return functions
}
