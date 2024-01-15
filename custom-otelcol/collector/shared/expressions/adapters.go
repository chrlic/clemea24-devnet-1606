package expressions

import (
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

type StringAdapter struct {
	ref.TypeAdapter
}

func (s StringAdapter) NativeToValue(str any) ref.Val {
	val := types.String(str.(string))
	return val
}

type DoubleAdapter struct {
	ref.TypeAdapter
}

func (s DoubleAdapter) NativeToValue(input any) ref.Val {
	return input.(ref.Val)
}

type AnyAdapter struct {
	ref.TypeAdapter
}

func (s AnyAdapter) NativeToValue(input any) ref.Val {
	return input.(ref.Val)
}
