package api

import (
	"github.com/fasthttp/router"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/valyala/fasthttp"
	"limq/authenticator"
	"limq/broker"
)

type Stub struct {
	auth           *authenticator.A
	bufferedBroker *broker.Mega
	routes         *router.Router
	ea             *exclusiveAccess
}

func (stub *Stub) Handler() func(ctx *fasthttp.RequestCtx) {
	return stub.routes.Handler
}

var strApplicationJSON = []byte("application/json")

func NewStub(pool *pgxpool.Pool, a *authenticator.A) *Stub {
	s := &Stub{
		auth:           a,
		bufferedBroker: broker.NewMega(pool, a.CreateMixinManager()),
	}

	r := router.New()
	s.routes = r

	r.GET("/listen{access_key}", CorsMiddlewareAny(s.listen))
	r.POST("/publish{access_key}", CorsMiddlewareAny(s.publish))
	r.GET("/subscribe{access_key}", CorsMiddlewareAny(s.listenWS))
	//r.GET("/purge{access_key}", CorsMiddlewareAny(s.purge))

	r.HandleOPTIONS = true
	r.GlobalOPTIONS = CorsMiddlewareAny(func(ctx *fasthttp.RequestCtx) {
		ctx.Response.Header.Set("Allow", "OPTIONS, GET, POST")
	})

	r.NotFound = CorsMiddlewareAny(func(ctx *fasthttp.RequestCtx) {
		ctx.Response.Header.Set("Allow", "OPTIONS, GET, POST")
	})

	s.ea = newEA()

	return s
}
