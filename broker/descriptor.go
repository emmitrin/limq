package broker

import "time"

// Descriptor is the core interface for broker manipulation
type Descriptor interface {
	Send(b []byte) error
	Read(timeout time.Duration) ([]byte, error)
	Buffered() bool
}

type Buffer interface {
	Count() int
	Clear()
	BufferSize() int
}

type BufferedDescriptor interface {
	Descriptor
	Buffer
}
