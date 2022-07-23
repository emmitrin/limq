package api

import (
	"github.com/valyala/fasthttp"
)

func CorsMiddlewareAny(f func(ctx *fasthttp.RequestCtx)) func(ctx *fasthttp.RequestCtx) {
	return func(ctx *fasthttp.RequestCtx) {
		ctx.Response.Header.Set("access-control-allow-origin", "*")
		ctx.Response.Header.Set("Access-Control-Allow-Headers", "X-Message-Type, X-Timeout")

		f(ctx)
	}
}
