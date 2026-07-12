package repository

import (
	"context"

	"github.com/niflaot/pixels/pkg/postgres"
)

// transactionRunner runs one messenger mutation atomically.
type transactionRunner func(context.Context, func(context.Context, postgres.Executor) error) error

// Repository persists messenger records in PostgreSQL.
type Repository struct {
	// executor runs PostgreSQL statements.
	executor postgres.Executor
	// withinTx runs atomic relationship mutations.
	withinTx transactionRunner
}

// New creates a messenger repository.
func New(executor postgres.Executor) *Repository {
	repository := &Repository{executor: executor}
	pool, pooled := executor.(*postgres.Pool)
	if pooled {
		repository.withinTx = func(ctx context.Context, work func(context.Context, postgres.Executor) error) error {
			return postgres.WithinScope(ctx, pool, func(txCtx context.Context) error {
				return work(txCtx, postgres.ExecutorFor(txCtx, executor))
			})
		}
	} else {
		repository.withinTx = func(ctx context.Context, work func(context.Context, postgres.Executor) error) error {
			return work(ctx, executor)
		}
	}

	return repository
}
