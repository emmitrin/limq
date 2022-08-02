package storage

import "github.com/jackc/pgx/v4/pgxpool"

// Keeper is a core handle for buffered messages persistence
type Keeper struct {
	pool *pgxpool.Pool
}
