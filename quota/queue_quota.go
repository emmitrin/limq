package quota

const (
	kb = 1 << 10

	MaxMessageSize      = 256 * kb
	MaxBufferedMessages = 256

	MaxSizePerQueue = MaxMessageSize * MaxBufferedMessages
)
