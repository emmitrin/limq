package api

import (
	"github.com/fasthttp/router"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/valyala/fasthttp"
	"limq/authenticator"
	"limq/broker"
)

type Stub struct {
	a  *authenticator.A
	br *broker.AutoBufferedBroker
	r  *router.Router
}

func (stub *Stub) Handler() func(ctx *fasthttp.RequestCtx) {
	return stub.r.Handler
}

var strApplicationJSON = []byte("application/json")

func NewStub(pool *pgxpool.Pool, a *authenticator.A) *Stub {
	s := &Stub{
		a:  a,
		br: broker.NewAQ(pool, a.CreateMixinManager()),
	}

	r := router.New()
	s.r = r

	r.GET("/listen{access_key}", s.listen)
	r.POST("/post{access_key}", s.post)
	r.GET("/ws_listen{access_key}", s.listenWS)

	return s
}
