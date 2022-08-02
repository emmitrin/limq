package storage

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v4"
	"go.uber.org/zap"
	"limq/message"
	"limq/quota"
)

func (k *Keeper) Put(m *message.Message) error {
	to, cancel := context.WithTimeout(context.Background(), DBTimeout)
	defer cancel()

	// acquire a connection from the pool
	conn, err := k.pool.Acquire(to)
	if err != nil {
		return err
	}

	defer conn.Release()

	// start an isolated transaction
	tx, err := conn.BeginTx(to, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}

	defer tx.Rollback(to)

	// obtain unread messages count
	unread, err := k.Count(to, tx, m.ChannelID)
	if err != nil {
		return err
	}

	if unread == quota.MaxBufferedMessages {
		// quota is reached, delete the oldest message

		err := k.dropOldest(to, tx, m.ChannelID)
		if err != nil {
			return err
		}
	} else if unread > quota.MaxBufferedMessages {
		// quota is already reached

		zap.L().Error("race detected or quota has changed",
			zap.Int("quota", quota.MaxBufferedMessages), zap.Int("count", unread))

		return errors.New("quota is already reached")
	}

	// insert the message
	_, err = tx.Exec(
		context.Background(),
		"INSERT INTO messages (tag, msg_type, content) VALUES ($1, $2, $3)",
		m.ChannelID,
		m.Type,
		m.Payload,
	)

	if err != nil {
		return err
	}

	err = tx.Commit(context.Background())
	if err != nil {
		return err
	}

	return nil
}
