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
	bufferedBroker *broker.AutoBufferedBroker
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
		bufferedBroker: broker.NewAQ(pool, a.CreateMixinManager()),
	}

	r := router.New()
	s.routes = r

	r.GET("/listen{access_key}", s.listen)
	r.POST("/post{access_key}", s.post)
	r.GET("/ws_listen{access_key}", s.listenWS)

	s.ea = newEA()

	return s
}
