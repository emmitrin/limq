package message

type Message struct {
	Type      Type
	Scope     Scope
	ChannelID string
	Payload   []byte
}
