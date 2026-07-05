// Package repository contains PostgreSQL access for player records.
package repository

import "github.com/niflaot/pixels/pkg/postgres"

// Repository reads and writes player persistence records.
type Repository struct {
	// executor runs PostgreSQL queries.
	executor postgres.Executor
}

// New creates a player repository.
func New(executor postgres.Executor) *Repository {
	return &Repository{executor: executor}
}
