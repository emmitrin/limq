package api

import (
	"encoding/json"
	"github.com/valyala/fasthttp"
	"limq/message"
	"net/http"
)

func (stub *Stub) post(ctx *fasthttp.RequestCtx) {
	key := ctx.UserValue("access_key").(string)

	defer ctx.SetContentTypeBytes(strApplicationJSON)

	auth := stub.auth.CheckAccessKey(key)
	if !auth.Flags.Active() || len(auth.Tag) == 0 {
		sendError(ctx, fastError(CodeAuthenticationError, "access key is suspended or invalid"), http.StatusUnauthorized)
		return
	}

	if !auth.Flags.CanPost() {
		sendError(ctx, fastError(CodeAuthenticationError, "no post permissions"), http.StatusForbidden)
		return
	}

	var typ, scope int

	{
		ok := false

		messageTypeRaw := ctx.Request.Header.Peek("x-message-type")
		typ, ok = message.ParseType(string(messageTypeRaw))
		if !ok {
			sendError(ctx, fastError(CodeUnknownMessageType, "unknown message type"), http.StatusBadRequest)
			return
		}
	}

	{
		scopeRaw := string(ctx.Request.Header.Peek("x-scope"))
		scope = message.ParseScope(scopeRaw)
	}

	m := &message.Message{ChannelID: auth.Tag, Type: typ, Scope: scope}

	{
		body := ctx.PostBody()
		m.Payload = make([]byte, len(body))
		copy(m.Payload, body)

		err := stub.bufferedBroker.PostWithMixin(auth.Tag, m)

		if err == nil {
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
