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
		sendError(ctx, fastError(CodeAuthenticationError, "access key is suspended or invalid"), http.StatusUnauthorized)
		ctx.SetContentTypeBytes(strApplicationJSON)
		return
	}

	if !auth.Flags.CanListen() {
		sendError(ctx, fastError(CodeAuthenticationError, "no listen permissions"), http.StatusForbidden)
		ctx.SetContentTypeBytes(strApplicationJSON)
		return
	}

	if !stub.ea.start(key) {
		sendError(ctx, fastError(CodeAnotherClientIsOnline, "this access key is being used by another listener right now"), http.StatusConflict)
		return
	}

	defer stub.ea.stop(key)

	err := upgrader.Upgrade(ctx, func(conn *websocket.Conn) {
		listenerContext, cancel := context.WithCancel(context.Background())

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
		zap.L().Error("websocket upgrade", zap.Error(err))
	}
}
