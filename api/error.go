package api

import (
	"github.com/valyala/fasthttp"
	"io"
)

type statusCodeWithText struct {
	hasCode
	hasStatusText
}

func writeError(w io.Writer, statusCode int, reason string) {
	m := statusCodeWithText{}
	m.Code = statusCode
	m.StatusText = reason

	writeJSON(w, m)
}

func setError(ctx *fasthttp.RequestCtx, statusCode int) {
	ctx.SetStatusCode(statusCode)
	ctx.SetContentTypeBytes(strApplicationJSON)
}
