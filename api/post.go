package api

import (
	"errors"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"limq/broker"
	"limq/message"
	"net/http"
)

func (stub *Stub) post(ctx *fasthttp.RequestCtx) {
	key := ctx.UserValue("access_key").(string)

	defer ctx.SetContentTypeBytes(strApplicationJSON)

	auth := stub.auth.CheckAccessKey(key)
	if !auth.Flags.Active() || len(auth.Tag) == 0 {
		setError(ctx, http.StatusUnauthorized)
		writeError(ctx, CodeAuthenticationError, "access key is suspended or invalid")

		return
	}

	if !auth.Flags.CanPost() {
		setError(ctx, http.StatusForbidden)
		writeError(ctx, CodeAuthenticationError, "no post permissions")

		return
	}

	var typ, scope int

	{
		ok := false

		messageTypeRaw := ctx.Request.Header.Peek("x-message-type")
		typ, ok = message.ParseType(string(messageTypeRaw))
		if !ok {
			setError(ctx, http.StatusBadRequest)
			writeError(ctx, CodeUnknownMessageType, "unknown message type")

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
			writeJSON(ctx, response)

		} else {
			response := statusCodeWithText{}

			if errors.Is(err, broker.ErrMessageIsEmpty) {
				response.Code = CodeMessageIsEmpty
				response.StatusText = "Message body is empty"

			} else if errors.Is(err, broker.ErrMessageIsTooLarge) {
				response.Code = CodeMessageIsTooBig
				response.StatusText = "Message is too large"

			} else {
				response.Code = CodeUnknownError
				response.StatusText = "Unable to post the message due to server error"

			}

			writeJSON(ctx, response)

			zap.L().Error("unable to post the message", zap.Error(err))
		}
	}
}
