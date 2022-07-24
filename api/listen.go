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

	auth := stub.auth.CheckAccessKey(key)
	if !auth.Flags.Active() || len(auth.Tag) == 0 {
		sendError(ctx, fastError(CodeAuthenticationError, "access key is suspended or invalid"), http.StatusUnauthorized)
		return
	}

	if !auth.Flags.CanListen() {
		sendError(ctx, fastError(CodeAuthenticationError, "no listen permissions"), http.StatusForbidden)
		return
	}

	if !stub.ea.start(key) {
		sendError(ctx, fastError(CodeAnotherClientIsOnline, "this access key is being used by another listener right now"), http.StatusForbidden)
		return
	}

	defer stub.ea.stop(key)

	timeoutValue := ctx.Request.Header.Peek("X-Timeout")

	seconds, err := strconv.Atoi(string(timeoutValue))
	if err != nil {
		seconds = defaultTimeout
	}

	timeout := time.Duration(seconds) * time.Second
	listenCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	{
		m := stub.bufferedBroker.Listen(listenCtx, auth.Tag)
		if m == nil {
			ctx.SetStatusCode(http.StatusNotModified)
			return
		}

		ctx.SetContentType("application/x-octet-stream")

		_, err := io.Copy(ctx, bytes.NewReader(m.Payload))
		if err != nil {
			sendError(ctx, fastError(CodeUnknownError, "can't drop buffer"), http.StatusInternalServerError)
			return
		}
	}
}
