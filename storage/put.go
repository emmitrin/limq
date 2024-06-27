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

	success := false

	defer func() {
		if !success {
			if rollbackErr := tx.Rollback(to); rollbackErr != nil {
				zap.L().Error("unable to rollback db tx", zap.Error(rollbackErr))
			}
		}
	}()

	// obtain unread messages count
	unread, err := k.Count(to, tx, m.ChannelID)
	if err != nil {
		return err
	}

	err = k.dropExcessUnread(to, tx, m, unread)
	if err != nil {
		return err
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

	err = tx.Commit(to)
	if err != nil {
		return err
	}

	success = true
	return nil
}

func (k *Keeper) dropExcessUnread(ctx context.Context, tx pgx.Tx, m *message.Message, unread int) error {
	if unread == quota.MaxBufferedMessages {
		// quota is reached, delete the oldest message

		err := k.dropOldest(ctx, tx, m.ChannelID)
		if err != nil {
			return err
		}
	} else if unread > quota.MaxBufferedMessages {
		// quota is already reached

		zap.L().Error("race detected or quota has changed",
			zap.Int("quota", quota.MaxBufferedMessages), zap.Int("count", unread))

		return errors.New("quota is already reached")
	}

	return nil
}
