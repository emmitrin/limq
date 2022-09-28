package broker

import (
	"limq/message"
	"limq/quota"
	"sync/atomic"
)

type unbufferedDirectStream struct {
	_online uint32
	c       chan *message.Message
}

func (s *unbufferedDirectStream) publishOne(m *message.Message) {
	if s.online() == 0 {
		panic("unbuffered stream publishOne on zero subscribers")
	}

	s.c <- m
}

func (s *unbufferedDirectStream) publish(m *message.Message) {
	if s.online() == 0 {
		panic("unbuffered stream publish on zero subscribers")
	}

	// dummy repeated send
	for i := uint32(0); i < s.online(); i++ {
		s.c <- m
	}
}

func (s *unbufferedDirectStream) subscribe() {
	atomic.AddUint32(&s._online, 1)
}

func (s *unbufferedDirectStream) unsubscribe() {
	atomic.AddUint32(&s._online, ^uint32(0))
}

func (s *unbufferedDirectStream) online() uint32 {
	return atomic.LoadUint32(&s._online)
}

func (s *unbufferedDirectStream) clear() {
	if s.online() != 0 {
		panic("clear is called on unbufferedDirectStream while subscribers count is not zero")
	}

	// todo mux
L:
	for {
		select {
		case <-s.c:
		default:
			break L
		}
	}

}

func newUnbufferedDirectS() stream {
	return &unbufferedDirectStream{c: make(chan *message.Message, quota.MaxBufferedMessages)}
}

func (s *unbufferedDirectStream) ch() chan *message.Message {
	return s.c
}
