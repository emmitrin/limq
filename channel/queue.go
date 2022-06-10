package channel

import (
	"sync"
	"time"
)

const (
	GlobalQueueMaxBufferPerChannel = 512
	GlobalQueueMaxMessageSize      = 50 * 1024
)

// GlobalQueue works in a multicast mode and holds buffered data in the process' memory
type GlobalQueue struct {
	mu *sync.Mutex
	wm map[string]stream
}

func NewGQ() *GlobalQueue {
	return &GlobalQueue{
		mu: &sync.Mutex{},
		wm: map[string]stream{},
	}
}

func (gq *GlobalQueue) acquire(tag string) stream {
	gq.mu.Lock()
	defer gq.mu.Unlock()

	s, ok := gq.wm[tag]
	if !ok {
		s = newStatefulMulticastStream()
		gq.wm[tag] = s
	}

	return s
}

func (gq *GlobalQueue) PostWithTimeout(m *Message, to time.Duration) (ok bool) {
	if len(m.Payload) > GlobalQueueMaxMessageSize {
		return false
	}

	t := time.NewTimer(to)

	streamHandler := gq.acquire(m.ChannelID)

	queue := streamHandler.ch()
	online := streamHandler.online()
	if online == 0 {
		online = 1
	}

	for i := uint32(0); i < online; i++ {
		select {
		case <-t.C:
			return false // todo check situations when only half of the online listeners received the msg

		case queue <- m:
		}
	}

	t.Stop()
	return true
}

func (gq *GlobalQueue) PostImmediately(m *Message) (ok bool) {
	if len(m.Payload) > GlobalQueueMaxMessageSize {
		return false
	}

	streamHandler := gq.acquire(m.ChannelID)

	queue := streamHandler.ch()
	online := streamHandler.online()
	if online == 0 {
		online = 1
	}

	for i := uint32(0); i < online; i++ {
		select {
		default:
			// channel is already fed, reject
			// todo check situations when only half of the online listeners received the msg
			return false

		case queue <- m:
		}
	}

	return true
}

func (gq *GlobalQueue) ListenWithTimeout(tag string, to time.Duration) (m *Message) {
	t := time.NewTimer(to)

	streamHandler := gq.acquire(tag)

	// todo make peer identification to solve the re-post issue
	streamHandler.subscribe()
	defer streamHandler.unsubscribe()

	queue := streamHandler.ch()

	select {
	case <-t.C:
		return nil

	case val := <-queue:
		t.Stop()

		return val
	}
}

func (gq *GlobalQueue) QueueSize(tag string) int {
	gq.mu.Lock()
	defer gq.mu.Unlock()

	s, ok := gq.wm[tag]
	if !ok {
		return 0
	}

	return len(s.ch())
}
