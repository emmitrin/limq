package storage

import (
	"context"
	"github.com/jackc/pgx/v4"
)

func (k *Keeper) dropOldest(ctx context.Context, tx pgx.Tx, tag string) error {
	_, err := tx.Exec(ctx,
		`DELETE FROM messages
					WHERE tag = $1
					AND id = (
						SELECT id FROM messages
						WHERE tag = $1
						ORDER BY ID ASC
						LIMIT 1
					)`,
		tag,
	)

	return err
}
