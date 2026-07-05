package service

import "github.com/niflaot/pixels/internal/realm/player/repository"

// Service validates and coordinates player persistence behavior.
type Service struct {
	// store reads and writes player persistence records.
	store repository.Store
}

// New creates a player service.
func New(store repository.Store) *Service {
	return &Service{store: store}
}
