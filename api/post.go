package api

import (
	"encoding/json"
	"github.com/valyala/fasthttp"
	"limq/channel"
	"net/http"
)

func (stub *Stub) post(ctx *fasthttp.RequestCtx) {
	key := ctx.UserValue("access_key").(string)

	ctx.SetContentTypeBytes(strApplicationJSON)

	auth := stub.a.CheckAccessKey(key)
	if !auth.Flags.Active() || len(auth.Tag) == 0 {
		ctx.Error(fastError(CodeAuthenticationError, "access key is suspended or invalid"), http.StatusUnauthorized)
		return
	}

	if !auth.Flags.CanPost() {
		ctx.Error(fastError(CodeAuthenticationError, "no post permissions"), http.StatusForbidden)
		return
	}

	m := &channel.Message{ChannelID: auth.Tag}

	{
		body := ctx.PostBody()
		m.Payload = make([]byte, len(body))
		copy(m.Payload, body)

		ok := stub.q.PostImmediately(m)

		if ok {
			response := struct{ hasCode }{}
			json.NewEncoder(ctx).Encode(response)
		} else {
			response := statusCodeWithText{}
			response.Code = CodeChannelIsFull
			response.Message = "Channel is already full. Enable a message listener to offload messages"
			json.NewEncoder(ctx).Encode(response)
		}
	}
}
