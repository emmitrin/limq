package broker

import (
	"sync/atomic"
	"time"
)

// InMemory is a dummy Go-broker-based message queue implementation utilizing only process' memory
type InMemory struct {
	buf     chan []byte
	waiting uint32
}

const inMemoryBufSize = 128

func (i *InMemory) BufferSize() int {
	return cap(i.buf)
}

func (i *InMemory) Send(b []byte) error {
	for x := uint32(0); x < atomic.LoadUint32(&i.waiting); x++ {
		select {
		case i.buf <- b:
		default:
		}
	}

	return nil
}

func (i *InMemory) Read(timeout time.Duration) ([]byte, error) {
	var t *time.Timer

	if timeout == -1 {
		t = &time.Timer{C: make(chan time.Time)}
	} else {
		t = time.NewTimer(timeout)
	}

	atomic.AddUint32(&i.waiting, 1)

	select {
	case <-t.C:
		return nil, ErrTimeout

	case data := <-i.buf:
		if timeout != -1 && !t.Stop() {
			<-t.C
		}

		return data, nil
	}
}

func (i *InMemory) Buffered() bool {
	return true
}

func (i *InMemory) Count() int {
	return len(i.buf)
}

func (i *InMemory) Clear() {
	i.buf = make(chan []byte, inMemoryBufSize)
}

func NewInMemory() *InMemory {
	return &InMemory{buf: make(chan []byte, inMemoryBufSize)}
}
