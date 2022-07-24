package api

import (
	"encoding/json"
	"github.com/valyala/fasthttp"
)

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

func sendError(ctx *fasthttp.RequestCtx, msg string, statusCode int) {
	ctx.SetStatusCode(statusCode)
	ctx.SetContentTypeBytes(strApplicationJSON)
	ctx.SetBodyString(msg)
}
