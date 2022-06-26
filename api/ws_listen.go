package api

import (
	"context"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"net/http"
)

var upgrader = websocket.FastHTTPUpgrader{} // use default options

func (stub *Stub) listenWS(ctx *fasthttp.RequestCtx) {
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

		for {
			m := stub.q.Listen(listenerContext, auth.Tag)
			if m == nil {
				if listenerContext.Err() == nil {
					zap.L().Warn("invalid nil message", zap.String("tag", auth.Tag))
				}

				break
			}

			err := conn.WriteMessage(websocket.BinaryMessage, m.Payload)
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
