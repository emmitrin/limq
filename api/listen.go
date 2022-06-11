package api

import (
	"bytes"
	"context"
	"github.com/valyala/fasthttp"
	"io"
	"net/http"
	"strconv"
	"time"
)

const defaultTimeout = 40

func (stub *Stub) listen(ctx *fasthttp.RequestCtx) {
	key := ctx.UserValue("access_key").(string)

	auth := stub.a.CheckAccessKey(key)
	if !auth.Flags.Active() || len(auth.Tag) == 0 {
		ctx.Error(fastError(CodeAuthenticationError, "access key is suspended or invalid"), http.StatusUnauthorized)
		ctx.SetContentTypeBytes(strApplicationJSON)
		return
	}

	if !auth.Flags.CanListen() {
		ctx.Error(fastError(CodeAuthenticationError, "no listen permissions"), http.StatusForbidden)
		ctx.SetContentTypeBytes(strApplicationJSON)
		return
	}

	timeoutValue := ctx.Request.Header.Peek("X-Timeout")

	seconds, err := strconv.Atoi(string(timeoutValue))
	if err != nil {
		seconds = defaultTimeout
	}

	timeout := time.Duration(seconds) * time.Second
	listenCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	{
		m := stub.q.Listen(listenCtx, auth.Tag)
		if m == nil {
			ctx.SetStatusCode(http.StatusNotModified)
			return
		}

		ctx.SetContentType("application/x-octet-stream")

		_, err := io.Copy(ctx, bytes.NewReader(m.Payload))
		if err != nil {
			ctx.Error(fastError(CodeUnknownError, "can't drop buffer"), http.StatusInternalServerError)
			ctx.SetContentTypeBytes(strApplicationJSON)
			return
		}
	}
}
