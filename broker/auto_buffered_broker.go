package broker

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"
	"limq/internal/set"
	"limq/message"
	"sync"
	"time"
)

var (
	ErrNoBufferedMessages = errors.New("no buffered messages")
)

const (
	directStreamingBuffer = 128
)

// AutoBufferedBroker works in a multicast mode (all receivers can receive the same message).
// If a message is posted onto the broker which has zero subscribers at the time,
// it will be buffered in DBMS
type AutoBufferedBroker struct {
	pool   *pgxpool.Pool
	direct map[string]stream
	mman   MixinManager
	mu     *sync.Mutex
}

func NewAQ(pool *pgxpool.Pool, mman MixinManager) *AutoBufferedBroker {
	return &AutoBufferedBroker{
		mu:     &sync.Mutex{},
		direct: map[string]stream{},
		mman:   mman,
		pool:   pool,
	}
}

func (aq *AutoBufferedBroker) acquire(tag string) stream {
	aq.mu.Lock()
	defer aq.mu.Unlock()

	s, ok := aq.direct[tag]
	if !ok {
		s = newUnbufferedDirectS()
		aq.direct[tag] = s
	}

	return s
}

func (aq *AutoBufferedBroker) storeToBufferPersist(m *message.Message) error {
	to, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	conn, err := aq.pool.Acquire(to)
	if err != nil {
		zap.L().Error("unable to acquire db conn; message is lost", zap.Error(err), zap.String("tag", m.ChannelID))
		return err
	}

	defer conn.Release()

	_, err = conn.Exec(
		context.Background(),
		"INSERT INTO messages (tag, msg_type, content) VALUES ($1, $2, $3)",
		m.ChannelID,
		m.Type,
		m.Payload,
	)

	if err != nil {
		zap.L().Error("message is lost due to pgxpool error", zap.Error(err), zap.String("tag", m.ChannelID))
		return err
	}

	return nil
}

func (aq *AutoBufferedBroker) Post(m *message.Message) error {
	if len(m.Payload) > GlobalQueueMaxMessageSize {
		return errors.New("message is too big")
	}

	streamHandler := aq.acquire(m.ChannelID)

	online := streamHandler.online()
	if online == 0 {
		return aq.storeToBufferPersist(m)
	}

	switch m.Scope {
	case message.ScopeNotifyAll:
		streamHandler.post(m)

	case message.ScopeNotifyOne:
		streamHandler.postOne(m)
	}

	return nil
}

func (aq *AutoBufferedBroker) readBuffered(ctx context.Context, tag string) (m *message.Message, err error) {
	conn, err := aq.pool.Acquire(ctx)
	if err != nil {
		zap.L().Error("unable to acquire db conn", zap.Error(err), zap.String("tag", tag))
		return nil, err
	}

	defer conn.Release()

	row := conn.QueryRow(
		ctx,
		`DELETE FROM messages
			WHERE id = (
				SELECT id FROM messages WHERE tag = $1 ORDER BY ID ASC LIMIT 1
			) RETURNING msg_type, content`,
		tag,
	)

	nm := &message.Message{}

	err = row.Scan(&nm.Type, &nm.Payload)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoBufferedMessages
		}

		zap.L().Error("pgx error", zap.Error(err), zap.String("tag", tag))
		return nil, err
	}

	return nm, nil
}

func (aq *AutoBufferedBroker) Listen(ctx context.Context, tag string) (m *message.Message) {
	// dispatch buffered messages
	m, err := aq.readBuffered(ctx, tag)
	if err != nil && !errors.Is(err, ErrNoBufferedMessages) {
		return nil
	}

	if m != nil {
		return m
	}

	streamHandler := aq.acquire(tag)

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

func (aq *AutoBufferedBroker) streamDispatch(ctx context.Context, tag string, target chan *message.Message) {
	streamHandler := aq.acquire(tag)

	queue := streamHandler.ch()

	// todo make this smarter
	direct := make(chan *message.Message, directStreamingBuffer)

	go func() {
		streamHandler.subscribe()
		defer streamHandler.unsubscribe()

		for {
			select {
			case <-ctx.Done():
				close(direct)
				return

			case val := <-queue:
				direct <- val
			}
		}
	}()

	for {
		bufferedMessage, err := aq.readBuffered(ctx, tag)
		if errors.Is(err, ErrNoBufferedMessages) {
			break
		}

		if err != nil {
			zap.L().Error("listen streaming mode: dispatch buffered error", zap.String("tag", tag), zap.Error(err))
			break
		}

		target <- bufferedMessage
	}

	for m := range direct {
		target <- m
	}

	close(target)
}

func (aq *AutoBufferedBroker) ListenStream(ctx context.Context, tag string) chan *message.Message {
	c := make(chan *message.Message, 10)

	go aq.streamDispatch(ctx, tag, c)

	return c
}

func (aq *AutoBufferedBroker) repost(visited *set.Set[string], tag string, m message.Message, publishCurrent bool) {
	if visited.Has(tag) {
		zap.L().Warn("repost for mixed-in broker: circular dependency detected", zap.String("chan_id", tag))
		return
	}

	m.ChannelID = tag

	if publishCurrent {
		err := aq.Post(&m)

		if err != nil {
			zap.L().Warn("unable to post to mixed-in broker", zap.String("chan_id", tag), zap.Error(err))
		}
	}

	visited.Add(tag)
	tags := aq.mman.GetForwards(tag)

	for _, t := range tags {
		aq.repost(visited, t, m, true)
	}
}

func (aq *AutoBufferedBroker) PostWithMixin(tag string, m *message.Message) error {
	err := aq.Post(m)
	if err != nil {
		return err
	}

	if m.Scope != message.ScopeNotifyAll {
		return nil
	}

	go aq.repost(set.NewSet[string](nil), tag, *m, false)

	return nil
}
