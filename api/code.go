package api

const (
	CodeOk = iota
	CodeAuthenticationError
	CodeUnknownError
	CodeChannelIsFull
	CodeTimeout
	CodeAnotherClientIsOnline
	CodeUnknownMessageType
	CodeMessageIsEmpty
	CodeMessageIsTooBig
)

type hasCode struct {
	Code int `json:"status_code"`
}

type hasStatusText struct {
	StatusText string `json:"status_text"`
}
