package ssg

type setStr map[string]struct{}

func (s setStr) insert(v string) bool {
	_, ok := s[v]
	s[v] = struct{}{}

	return ok
}

func (s setStr) contains(v string) bool {
	_, ok := s[v]
	return ok
}

// type set[T comparable] map[T]struct{}

// func (s set[T]) insert(v T) bool {
// 	_, ok := s[v]
// 	s[v] = struct{}{}

// 	return ok
// }

// func (s set[T]) contains(v T) bool {
// 	_, ok := s[v]
// 	return ok
// }
