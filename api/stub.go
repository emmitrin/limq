package api

import (
	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
	"limq/authenticator"
	"limq/channel"
)

type Stub struct {
	a *authenticator.A
	q *channel.GlobalQueue
	r *router.Router
}

func (stub *Stub) Handler() func(ctx *fasthttp.RequestCtx) {
	return stub.r.Handler
}

var strApplicationJSON = []byte("application/json")

func NewStub(a *authenticator.A) *Stub {
	s := &Stub{
		a: a,
		q: channel.NewGQ(a.CreateMixinManager()),
	}

	r := router.New()
	s.r = r

	r.GET("/listen{access_key}", s.listen)
	r.POST("/post{access_key}", s.post)
	r.GET("/ws_listen{access_key}", s.listenWS)

	return s
}
