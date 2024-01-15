package jsonscraper

type Stack[T any] struct {
	keys []T
}

func NewStack[T any]() *Stack[T] {
	return &Stack[T]{nil}
}

func (stack *Stack[T]) Push(key T) {
	stack.keys = append(stack.keys, key)
}

func (stack *Stack[T]) Top() (T, bool) {
	var x T
	if len(stack.keys) > 0 {
		x = stack.keys[len(stack.keys)-1]
		return x, true
	}
	return x, false
}

func (stack *Stack[T]) SetTop(key T) bool {
	if len(stack.keys) > 0 {
		stack.keys[len(stack.keys)-1] = key
		return true
	}
	return false
}

func (stack *Stack[T]) Pop() (T, bool) {
	var x T
	if len(stack.keys) > 0 {
		x, stack.keys = stack.keys[len(stack.keys)-1], stack.keys[:len(stack.keys)-1]
		return x, true
	}
	return x, false
}

func (stack *Stack[T]) IsEmpty() bool {
	return len(stack.keys) == 0
}

func (stack *Stack[T]) Reduce(new func() T, reducer func(T, T) T) T {
	accum := new()
	for i := 0; i < len(stack.keys); i++ {
		accum = reducer(accum, stack.keys[i])
	}
	return accum
}
