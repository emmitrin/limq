package api

import "encoding/json"

type statusCodeWithText struct {
	hasCode
	hasMessage
}

func fastError(statusCode int, reason string) string {
	m := statusCodeWithText{}
	m.Code = statusCode
	m.Message = reason

	val, _ := json.Marshal(m)

	return string(val)
}
