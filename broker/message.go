package broker

const (
	MessageBinary = iota
	MessageText
)

type Message struct {
	Type      int
	ChannelID string
	Payload   []byte
}
