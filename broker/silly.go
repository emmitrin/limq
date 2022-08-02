package broker

import (
	"sync/atomic"
	"time"
)

// Silly is a dummy Go-broker-based message queue implementation utilizing only process' memory
type Silly struct {
	buf     chan []byte
	waiting uint32
}

const inMemoryBufSize = 128

func (i *Silly) BufferSize() int {
	return cap(i.buf)
}

func (i *Silly) Send(b []byte) error {
	for x := uint32(0); x < atomic.LoadUint32(&i.waiting); x++ {
		select {
		case i.buf <- b:
		default:
		}
	}

	return nil
}

func (i *Silly) Read(timeout time.Duration) ([]byte, error) {
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

func (i *Silly) Buffered() bool {
	return true
}

func (i *Silly) Count() int {
	return len(i.buf)
}

func (i *Silly) Clear() {
	i.buf = make(chan []byte, inMemoryBufSize)
}

func NewInMemory() *Silly {
	return &Silly{buf: make(chan []byte, inMemoryBufSize)}
}
