package channel

import "sync/atomic"

type stream interface {
	ch() chan *Message

	subscribe()
	unsubscribe()
	online() uint32
}

type statefulMulticastStream struct {
	_online uint32
	c       chan *Message
}

func (s *statefulMulticastStream) subscribe() {
	atomic.AddUint32(&s._online, 1)
}

func (s *statefulMulticastStream) unsubscribe() {
	atomic.AddUint32(&s._online, ^uint32(0))
}

func (s *statefulMulticastStream) online() uint32 {
	return atomic.LoadUint32(&s._online)
}

func newStatefulMulticastStream() stream {
	return &statefulMulticastStream{c: make(chan *Message, GlobalQueueMaxBufferPerChannel)}
}

func (s *statefulMulticastStream) ch() chan *Message {
	return s.c
}
