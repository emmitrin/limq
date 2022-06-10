package api

const (
	CodeOk = iota
	CodeAuthenticationError
	CodeUnknownError
	CodeChannelIsFull
	CodeTimeout
)

type hasCode struct {
	Code int `json:"status_code"`
}

type hasMessage struct {
	Message string `json:"message"`
}
