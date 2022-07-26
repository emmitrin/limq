package message

import "strings"

const (
	ScopeNotifyAll = iota
	ScopeNotifyOne
)

func ParseScope(s string) int {
	switch strings.ToLower(s) {
	case "one":
		return ScopeNotifyOne

	default:
		return ScopeNotifyAll
	}
}
