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
