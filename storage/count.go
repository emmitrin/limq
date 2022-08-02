package storage

import (
	"context"
	"github.com/jackc/pgx/v4"
)

func (k *Keeper) Count(ctx context.Context, tx pgx.Tx, tag string) (int, error) {
	result := tx.QueryRow(ctx, "SELECT COUNT(*) FROM messages WHERE tag = $1", tag)

	var count int32
	err := result.Scan(&count)

	if err != nil {
		return 0, err
	}

	return int(count), nil
}
