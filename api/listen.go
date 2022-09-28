package api

import (
	"bytes"
	"context"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strconv"
	"time"
)

const defaultTimeout = 25

func (stub *Stub) listen(ctx *fasthttp.RequestCtx) {
	key := ctx.UserValue("access_key").(string)

	auth := stub.auth.CheckAccessKey(key)
	if !auth.Flags.Active() || len(auth.Tag) == 0 {
		setError(ctx, http.StatusUnauthorized)
		writeError(ctx, CodeAuthenticationError, "access key is suspended or invalid")

		return
	}

	if !auth.Flags.CanListen() {
		setError(ctx, http.StatusForbidden)
		writeError(ctx, CodeAuthenticationError, "no listen permissions")

		return
	}

	if !stub.ea.start(key) {
		setError(ctx, http.StatusConflict)
		writeError(ctx, CodeAnotherClientIsOnline, "this access key is being used by another listener right now")

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
		ctx.Response.Header.Set("X-Message-Scope", m.Scope.String())
		ctx.Response.Header.Set("X-Message-Type", m.Type.String())

		_, err := io.Copy(ctx, bytes.NewReader(m.Payload))
		if err != nil {
			zap.L().Error("can't drop buffer", zap.String("chan_id", m.ChannelID), zap.Error(err))
			return
		}
	}
}
