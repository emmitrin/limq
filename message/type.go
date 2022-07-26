package message

import (
	"github.com/fasthttp/websocket"
	"strconv"
	"strings"
)

const (
	TypeBinary = iota
	TypeText
)

func ParseType(t string) (int, bool) {
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

func TypeToWebSocketType(t int) int {
	switch t {
	case TypeText:
		return websocket.TextMessage

	case TypeBinary:
		fallthrough

	default:
		return websocket.BinaryMessage
	}
}
