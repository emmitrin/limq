package message

import "strings"

type Scope int

const (
	ScopeNotifyAll Scope = iota
	ScopeNotifyOne
)

func (s Scope) String() string {
	if s == ScopeNotifyOne {
		return "one"
	}

	return "all"
}

func ParseScope(s string) Scope {
	switch strings.ToLower(s) {
	case "one":
		return ScopeNotifyOne

	default:
		return ScopeNotifyAll
	}
}
