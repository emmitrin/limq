package channel

import (
	"context"
	"go.uber.org/zap"
	"limq/internal/set"
	"sync"
	"time"
)

const (
	GlobalQueueMaxBufferPerChannel = 512
	GlobalQueueMaxMessageSize      = 50 * 1024
)

// GlobalQueue works in a multicast mode and holds buffered data in the process' memory
type GlobalQueue struct {
	mu   *sync.Mutex
	wm   map[string]stream
	mman MixinManager
}

func NewGQ(mman MixinManager) *GlobalQueue {
	return &GlobalQueue{
		mu:   &sync.Mutex{},
		wm:   map[string]stream{},
		mman: mman,
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

func (gq *GlobalQueue) Listen(ctx context.Context, tag string) (m *Message) {
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

func (gq *GlobalQueue) QueueSize(tag string) int {
	gq.mu.Lock()
	defer gq.mu.Unlock()

	s, ok := gq.wm[tag]
	if !ok {
		return 0
	}

	return len(s.ch())
}

func (gq *GlobalQueue) repost(visited *set.Set[string], tag string, m Message, postToThis bool) {
	if visited.Has(tag) {
		zap.L().Warn("repost for mixed-in channel: circular dependency detected", zap.String("chan_id", tag))
		return
	}

	m.ChannelID = tag

	if postToThis {
		ok := gq.PostImmediately(&m)

		if !ok {
			zap.L().Warn("unable to post to mixed-in channel", zap.String("chan_id", tag))
		}
	}

	visited.Add(tag)
	tags := gq.mman.GetForwards(tag)

	for _, t := range tags {
		gq.repost(visited, t, m, true)
	}
}

func (gq *GlobalQueue) PostImmediatelyWithMixins(tag string, m *Message) (ok bool) {
	ok = gq.PostImmediately(m)

	go gq.repost(set.NewSet[string](nil), tag, *m, false)

	return
}
