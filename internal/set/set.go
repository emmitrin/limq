package set

import (
	"bytes"
	"encoding/gob"
	"math/rand"
	"sync/atomic"
	"time"
)

var randomizer *rand.Rand // init on produce.go

func init() {
	src := rand.NewSource(time.Now().Unix())
	randomizer = rand.New(src)
}

// Zero-sized value used as a placeholder
type _dummy = struct{}

var dummy = _dummy{}

type Set[T comparable] struct {
	m map[T]_dummy

	// Snap is a cached version of the set,
	// useful when some action requires a sequence of elements,
	// not their map representation
	snap []T

	locked int32
}

const (
	unlockedSet = iota
	lockedSet
)

// Internal check with no blocking
func (s *Set[T]) containsUnsafe(v T) bool {
	_, c := s.m[v]
	return c
}

func (s *Set[T]) unlocked() bool {
	return atomic.LoadInt32(&s.locked) == unlockedSet
}

// Makes an internal snapshot only if it's necessary, i.e. the previous one is empty
func (s *Set[T]) snapshot() {
	if s.snap == nil {
		s.mustSnapshot()
	}
}

// Makes an internal snapshot of a values for further usage.
func (s *Set[T]) mustSnapshot() {
	cd := make([]T, s.Len())
	cx := 0
	for k := range s.m {
		cd[cx] = k
		cx++
	}
	s.snap = cd
}

// Has determines that a key is in the set.
func (s *Set[T]) Has(v T) bool {
	return s.containsUnsafe(v)
}

// Add adds a string value to the set. If the string is already in the set, does nothing.
// Returns exactly was the value added at this time or not.
func (s *Set[T]) Add(v T) bool {
	if !s.unlocked() {
		return false
	}

	c := s.containsUnsafe(v)

	if !c && s.snap != nil {
		s.snap = append(s.snap, v)
	}

	s.m[v] = dummy
	return !c
}

func (s *Set[T]) Len() int {
	return len(s.m)
}

func (s *Set[T]) Delete(v T) {
	if !s.unlocked() {
		return
	}

	delete(s.m, v)
	s.snap = nil // invalidate cache
}

// Range calls f sequentially for each value present in the set.
// If f returns false, range stops the iteration.
// Range makes a snapshot of the values at the start.
// Values added during f execution will not be iterated.
func (s *Set[T]) Range(f func(T) bool) {
	for k := range s.m {
		if !f(k) {
			break
		}
	}
}

// Pick picks a random element from the set and returns it.
// If the set is empty, Pick panics.
func (s *Set[T]) Pick() T {
	s.snapshot()

	if len(s.snap) == 0 {
		panic("calling Pick() on empty Set")
	}

	return s.snap[randomizer.Intn(len(s.snap))]
}

func (s *Set[T]) ToSlice() []T {
	ms := make([]T, len(s.m))
	x := 0
	for k := range s.m {
		ms[x] = k
		x++
	}

	return ms
}

// Clear removes everything from the set.
// It is more convenient than creating a new one, if necessary.
func (s *Set[T]) Clear() {
	if !s.unlocked() {
		return
	}

	s.snap = nil
	s.m = make(map[T]_dummy)
}

// Equal reports if two *Set[T]s are equal by values, or they aren't.
func (s *Set[T]) Equal(s1 *Set[T]) bool {
	if s == s1 { // equal by pointer
		return true
	}

	if len(s.m) != len(s1.m) {
		return false
	}

	for k := range s.m {
		_, ok := s1.m[k]
		if !ok {
			return false
		}
	}

	return true
}

// Intersection returns the set with the elements that contains the items that exist in both sets.
func (s *Set[T]) Intersection(s1 *Set[T]) *Set[T] {
	in := NewSet[T](nil)

	s.Range(func(s T) bool {
		if s1.Has(s) {
			in.Add(s)
		}
		return true
	})

	return in
}

// Difference returns a set containing the difference between two sets.
func (s *Set[T]) Difference(s1 *Set[T]) *Set[T] {
	in := NewSet[T](nil)

	s.Range(func(s T) bool {
		if !s1.Has(s) {
			in.Add(s)
		}
		return true
	})

	s1.Range(func(v T) bool {
		if !s.Has(v) {
			in.Add(v)
		}
		return true
	})

	return in
}

// Copy creates an independent copy of the set.
// It is equal to New*Set[T](s.ToSlice()), but this is more efficient.
func (s *Set[T]) Copy() *Set[T] {
	s.snapshot()

	return NewSet[T](s.snap)
}

// If the set is locked, any write/delete/clear operation on it will result in nothing.
// For example, Add(...) won't add a string when the set is locked.
// Returns current locking state (true - locked).
// This operation is goroutine-safe and does not require any synchronization stuff.
func (s *Set[T]) ToggleLock() bool {
	old := atomic.LoadInt32(&s.locked)
	atomic.StoreInt32(&s.locked, old^lockedSet)
	return old == unlockedSet
}

// Gob encoding implementation
func (s *Set[T]) GobEncode() ([]byte, error) {
	encb := new(bytes.Buffer)
	enc := gob.NewEncoder(encb)
	err := enc.Encode(s.ToSlice())
	if err != nil {
		return nil, err
	}

	return encb.Bytes(), nil
}

// Gob decoding implementation
func (s *Set[T]) GobDecode(b []byte) error {
	if !s.unlocked() {
		return nil
	}

	s.Clear() // or maybe not?
	defer s.snapm()
	dec := gob.NewDecoder(bytes.NewReader(b))
	return dec.Decode(&s.snap)
}

//
func (s *Set[T]) snapm() {
	for _, x := range s.snap {
		s.m[x] = dummy
	}
}

// Creates a new set from the values of a given slice.
// Left nil if you want to create an empty set.
func NewSet[T comparable](fromSlice []T) *Set[T] {
	s := new(Set[T])
	s.m = make(map[T]_dummy)

	if fromSlice != nil {
		s.snap = make([]T, len(fromSlice))
		copy(s.snap, fromSlice)
	}

	for _, k := range fromSlice {
		s.Add(k)
	}

	return s
}
