package set

import (
	"math/rand"
	"time"
)

var randomizer *rand.Rand // init on produce.go

func init() {
	src := rand.NewSource(time.Now().Unix())
	randomizer = rand.New(src)
}

func first[T1, T2 any](v1 T1, v2 T2) T1  { return v1 }
func second[T1, T2 any](v1 T1, v2 T2) T2 { return v2 }

// Zero-sized value used as a placeholder
type _dummy = struct{}

var dummy = _dummy{}

type Set[T comparable] struct {
	m map[T]_dummy

	// Snap is a cached version of the set,
	// useful when some action requires a sequence of elements,
	// not their map representation
	snap []T
}

func (s *Set[T]) Add(val T) bool {
	_, ok := s.m[val]
	s.m[val] = dummy

	if s.snap != nil {
		s.snap = append(s.snap, val)
	}

	return !ok
}

func (s *Set[T]) Has(val T) bool {
	_, ok := s.m[val]
	return ok
}

func (s *Set[T]) Delete(val T) {
	delete(s.m, val)
	s.snap = nil
}

func (s *Set[T]) checkSnap() {
	if s.snap == nil {
		s.hardSnap()
	} else if len(s.snap) != len(s.m) {
		s.hardSnap()
	}
}

func (s *Set[T]) hardSnap() {
	s.snap = make([]T, 0, len(s.m))

	for k := range s.m {
		s.snap = append(s.snap, k)
	}
}

func (s *Set[T]) Clear() {
	s.m = map[T]_dummy{}
	s.snap = nil
}

func (s *Set[T]) Slice() []T {
	slice := make([]T, 0, len(s.m))

	for k := range s.m {
		slice = append(slice, k)
	}

	return nil
}

func (s *Set[T]) Len() int {
	return len(s.m)
}

func (s *Set[T]) Equal(s1 *Set[T]) bool {
	if s.Len() != s1.Len() {
		return false
	}

	for k := range s.m {
		if !s1.Has(k) {
			return false
		}
	}

	return true
}

func (s *Set[T]) Range(f func(T) bool) {
	for k := range s.m {
		if !f(k) {
			return
		}
	}
}

func (s *Set[T]) Copy() *Set[T] {
	s.checkSnap()

	return NewSet[T](s.snap...)
}

func (s *Set[T]) Pick() T {
	if s.Len() == 0 {
		panic("Pick() on empty Set!")
	}

	s.checkSnap()

	return s.snap[randomizer.Intn(len(s.snap))]
}

func NewSet[T comparable](init ...T) *Set[T] {
	s := &Set[T]{m: map[T]_dummy{}}

	for _, val := range init {
		s.Add(val)
	}

	return s
}
