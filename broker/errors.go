package broker

import "errors"

var (
	ErrTimeout = errors.New("message polling timeout")
)
