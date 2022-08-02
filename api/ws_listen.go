package api

import (
	"context"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"limq/message"
	"net/http"
)

var upgrader = websocket.FastHTTPUpgrader{
	CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
		return true // possibly unsafe
	},
}

func (stub *Stub) listenWS(ctx *fasthttp.RequestCtx) {
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

	err := upgrader.Upgrade(ctx, func(conn *websocket.Conn) {
		listenerContext, cancel := context.WithCancel(context.Background())

		defer stub.ea.stop(key)

		go func() {
			defer cancel()

			for {
				// bypass any incoming messages

				_, _, err := conn.ReadMessage()
				if err != nil {
					if _, ok := err.(*websocket.CloseError); !ok {
						zap.L().Error("ws error", zap.Error(err), zap.String("tag", auth.Tag))
					}

					return
				}
			}
		}()

		channel := stub.bufferedBroker.ListenStream(listenerContext, auth.Tag)

		for m := range channel {
			if m == nil {
				if listenerContext.Err() == nil {
					zap.L().Warn("invalid nil message", zap.String("tag", auth.Tag))
				}

				break
			}

			err := conn.WriteMessage(message.TypeToWebSocketType(m.Type), m.Payload)
			if err != nil {
				zap.L().Warn("unable to write message", zap.String("tag", auth.Tag))
				break
			}
		}
	})

	if err != nil {
		stub.ea.stop(key)
		zap.L().Error("websocket upgrade", zap.Error(err))
	}
}
