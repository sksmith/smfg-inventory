package db

import (
	"context"
)

type Transaction interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
	RollbackUnlessCommitted(ctx context.Context) error
}