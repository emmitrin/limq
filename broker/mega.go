package broker

import (
	"context"
	"errors"
	"github.com/emmitrin/util"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"
	"limq/message"
	"limq/quota"
	"limq/storage"
	"sync"
)

var (
	ErrNoBufferedMessages = errors.New("no buffered messages")
	ErrMessageIsTooLarge  = errors.New("message is too large")
	ErrMessageIsEmpty     = errors.New("message is empty")
)

const (
	directStreamingBuffer = 128
)

// Mega works in a multicast mode (all receivers can receive the same message).
// If a message is posted onto the broker which has zero subscribers at the time,
// it will be buffered in DBMS
type Mega struct {
	pool   *pgxpool.Pool
	direct map[string]stream
	mman   MixinManager
	mu     *sync.Mutex

	keeper *storage.Keeper
}

func NewMega(pool *pgxpool.Pool, mman MixinManager) *Mega {
	return &Mega{
		mu:     &sync.Mutex{},
		direct: map[string]stream{},
		mman:   mman,
		pool:   pool,
		keeper: storage.NewKeeper(pool),
	}
}

func (aq *Mega) acquire(tag string) stream {
	aq.mu.Lock()
	defer aq.mu.Unlock()

	s, ok := aq.direct[tag]
	if !ok {
		s = newUnbufferedDirectS()
		aq.direct[tag] = s
	}

	return s
}

func (aq *Mega) Publish(m *message.Message) error {
	if len(m.Payload) > quota.MaxMessageSize {
		return ErrMessageIsTooLarge
	}

	if len(m.Payload) == 0 {
		return ErrMessageIsEmpty
	}

	streamHandler := aq.acquire(m.ChannelID)

	online := streamHandler.online()
	if online == 0 {
		return aq.keeper.Put(m)
	}

	switch m.Scope {
	case message.ScopeNotifyAll:
		streamHandler.publish(m)

	case message.ScopeNotifyOne:
		streamHandler.publishOne(m)
	}

	return nil
}

func (aq *Mega) readBuffered(ctx context.Context, tag string) (m *message.Message, err error) {
	conn, err := aq.pool.Acquire(ctx)
	if err != nil {
		zap.L().Error("unable to acquire db conn", zap.Error(err), zap.String("tag", tag))
		return nil, err
	}

	defer conn.Release()

	tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		zap.L().Error("unable to acquire db tx", zap.Error(err), zap.String("tag", tag))
		return nil, err
	}

	row := tx.QueryRow(
		ctx,
		`DELETE FROM messages
			WHERE id = (
				SELECT id FROM messages WHERE tag = $1 ORDER BY ID ASC LIMIT 1
			) RETURNING msg_type, content`,
		tag,
	)

	nm := &message.Message{}

	// manually set scope to one
	// buffered messages are returned only to the race-winner listener, by design
	nm.Scope = message.ScopeNotifyOne

	err = row.Scan(&nm.Type, &nm.Payload)
	if err != nil {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			zap.L().Error("unable to rollback db tx", zap.Error(rollbackErr))
		}

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoBufferedMessages
		}

		zap.L().Error("pgx error", zap.Error(err), zap.String("tag", tag))
		return nil, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		zap.L().Error("unable to commit db tx", zap.Error(err), zap.String("tag", tag))
		return nil, err
	}

	return nm, nil
}

func (aq *Mega) Listen(ctx context.Context, tag string) (m *message.Message) {
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

func (aq *Mega) streamDispatch(ctx context.Context, tag string, target chan *message.Message) {
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

func (aq *Mega) ListenStream(ctx context.Context, tag string) chan *message.Message {
	c := make(chan *message.Message, 10)

	go aq.streamDispatch(ctx, tag, c)

	return c
}

func (aq *Mega) republish(visited *util.Set[string], tag string, m message.Message, publishCurrent bool) {
	if visited.Has(tag) {
		zap.L().Warn("republish for mixed-in broker: circular dependency detected", zap.String("chan_id", tag))
		return
	}

	m.ChannelID = tag

	if publishCurrent {
		err := aq.Publish(&m)

		if err != nil {
			zap.L().Warn("unable to publish to mixed-in broker", zap.String("chan_id", tag), zap.Error(err))
		}
	}

	visited.Add(tag)
	tags := aq.mman.GetForwards(tag)

	for _, t := range tags {
		aq.republish(visited, t, m, true)
	}
}

func (aq *Mega) PublishWithMixin(tag string, m *message.Message) error {
	err := aq.Publish(m)
	if err != nil {
		return err
	}

	if m.Scope != message.ScopeNotifyAll {
		return nil
	}

	go aq.republish(util.NewSet[string](), tag, *m, false)

	return nil
}
