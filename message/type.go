package message

import (
	"github.com/fasthttp/websocket"
	"strconv"
	"strings"
)

type Type int

const (
	TypeBinary Type = iota
	TypeText
)

func (t Type) String() string {
	if t == TypeText {
		return "text"
	}

	return "binary"
}

func ParseType(t string) (Type, bool) {
	if len(t) == 0 {
		return TypeBinary, true
	}

	i, err := strconv.Atoi(t)
	if err == nil {
		if i < 2 {
			return i, true
		}
	}

	switch strings.ToLower(t) {
	case "binary", "bin":
		return TypeBinary, true

	case "text", "plain":
		return TypeText, true

	default:
		return 0, false
	}
}

func TypeToWebSocketType(t Type) int {
	switch t {
	case TypeText:
		return websocket.TextMessage

	case TypeBinary:
		fallthrough

	default:
		return websocket.BinaryMessage
	}
}
