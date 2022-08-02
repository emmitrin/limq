package broker

import (
	"context"
	"go.uber.org/zap"
	"limq/internal/set"
	"limq/message"
	"limq/quota"
	"sync"
	"time"
)

// InMemory works in a multicast mode and holds buffered data in the process' memory
type InMemory struct {
	mu   *sync.Mutex
	wm   map[string]stream
	mman MixinManager
}

func NewGQ(mman MixinManager) *InMemory {
	return &InMemory{
		mu:   &sync.Mutex{},
		wm:   map[string]stream{},
		mman: mman,
	}
}

func (gq *InMemory) acquire(tag string) stream {
	gq.mu.Lock()
	defer gq.mu.Unlock()

	s, ok := gq.wm[tag]
	if !ok {
		s = newUnbufferedDirectS()
		gq.wm[tag] = s
	}

	return s
}

func (gq *InMemory) PostWithTimeout(m *message.Message, to time.Duration) (ok bool) {
	if len(m.Payload) > quota.MaxMessageSize {
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

func (gq *InMemory) PostImmediately(m *message.Message) (ok bool) {
	if len(m.Payload) > quota.MaxMessageSize {
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
			// broker is already fed, reject
			// todo check situations when only half of the online listeners received the msg
			return false

		case queue <- m:
		}
	}

	return true
}

func (gq *InMemory) Listen(ctx context.Context, tag string) (m *message.Message) {
	streamHandler := gq.acquire(tag)

	// todo make peer identification to solve the re-post issue
	streamHandler.subscribe()
	defer streamHandler.unsubscribe()

	queue := streamHandler.ch()

	select {
	case <-ctx.Done():
		return nil

	case val := <-queue:
		return val
	}
}

func (gq *InMemory) QueueSize(tag string) int {
	gq.mu.Lock()
	defer gq.mu.Unlock()

	s, ok := gq.wm[tag]
	if !ok {
		return 0
	}

	return len(s.ch())
}

func (gq *InMemory) repost(visited *set.Set[string], tag string, m message.Message, postToThis bool) {
	if visited.Has(tag) {
		zap.L().Warn("repost for mixed-in broker: circular dependency detected", zap.String("chan_id", tag))
		return
	}

	m.ChannelID = tag

	if postToThis {
		ok := gq.PostImmediately(&m)

		if !ok {
			zap.L().Warn("unable to post to mixed-in broker", zap.String("chan_id", tag))
		}
	}

	visited.Add(tag)
	tags := gq.mman.GetForwards(tag)

	for _, t := range tags {
		gq.repost(visited, t, m, true)
	}
}

func (gq *InMemory) PostImmediatelyWithMixins(tag string, m *message.Message) (ok bool) {
	ok = gq.PostImmediately(m)

	go gq.repost(set.NewSet[string](), tag, *m, false)

	return
}
