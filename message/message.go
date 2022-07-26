package message

type Message struct {
	Type      int
	Scope     int
	ChannelID string
	Payload   []byte
}
