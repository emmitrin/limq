package authenticator

import "strconv"

const (
	AccessRead        = 1 << 0
	AccessWrite       = 1 << 1
	AccessInfoEnabled = 1 << 2
	AccessSuspended   = 1 << 8
)

type AccessLevel int

func (al AccessLevel) CanListen() bool          { return al&AccessRead != 0 }
func (al AccessLevel) CanPost() bool            { return al&AccessWrite != 0 }
func (al AccessLevel) InfoRequestEnabled() bool { return al&AccessInfoEnabled != 0 }
func (al AccessLevel) Active() bool             { return al&AccessSuspended == 0 }

func parseAccessLevel(raw string) AccessLevel {
	i, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}

	return AccessLevel(i)
}
