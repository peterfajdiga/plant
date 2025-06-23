package stack

type Stack[T any] struct {
	items []T
}

func New[T any]() *Stack[T] {
	return &Stack[T]{}
}

func (s *Stack[T]) Push(item T) {
	s.items = append(s.items, item)
}

func (s *Stack[T]) Pop() (T, bool) {
	var zero T
	if len(s.items) == 0 {
		return zero, false
	}
	item := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return item, true
}

func (s *Stack[T]) Peek() (T, bool) {
	var zero T
	if len(s.items) == 0 {
		return zero, false
	}
	return s.items[len(s.items)-1], true
}

func (s *Stack[T]) Len() int {
	return len(s.items)
}

func (s *Stack[T]) IsEmpty() bool {
	return len(s.items) == 0
}

func (s *Stack[T]) MustPop() T {
	item, ok := s.Pop()
	if !ok {
		panic("stack: Pop from empty stack")
	}
	return item
}

func (s *Stack[T]) MustPeek() T {
	item, ok := s.Peek()
	if !ok {
		panic("stack: Peek from empty stack")
	}
	return item
}

func (s *Stack[T]) Drop(n int) {
	if n <= 0 {
		return
	}
	if n >= len(s.items) {
		s.items = s.items[:0]
		return
	}
	s.items = s.items[:len(s.items)-n]
}
