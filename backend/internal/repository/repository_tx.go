package repository

import (
	"context"
	"errors"

	dbent "github.com/Wei-Shaw/sub2api/ent"
)

// withRepositoryTransaction reuses a caller-owned transaction or creates and
// commits a local one. The bool reports whether this call committed locally,
// allowing callers to publish cache state only after a durable commit.
func withRepositoryTransaction(
	ctx context.Context,
	defaultClient *dbent.Client,
	fn func(context.Context, *dbent.Client) error,
) (bool, error) {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return false, fn(ctx, tx.Client())
	}

	tx, err := defaultClient.Tx(ctx)
	if errors.Is(err, dbent.ErrTxStarted) {
		return false, fn(ctx, defaultClient)
	}
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	if err := fn(txCtx, tx.Client()); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}
