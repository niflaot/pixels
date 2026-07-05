package service

import (
	"context"

	playermodel "github.com/niflaot/pixels/internal/player/model"
)

// Creator creates player identity and profile records.
type Creator interface {
	// Create creates a player with a profile.
	Create(ctx context.Context, params CreateParams) (Record, error)
}

// Finder reads player identity and profile records.
type Finder interface {
	// FindByID finds a player by id.
	FindByID(ctx context.Context, id int64) (Record, bool, error)

	// FindByUsername finds a player by username.
	FindByUsername(ctx context.Context, username string) (Record, bool, error)
}

// Manager creates and reads player records.
type Manager interface {
	Creator
	Finder
}

// Record contains a player identity and profile pair.
type Record struct {
	// Player is the durable player identity.
	Player playermodel.Player

	// Profile is the durable player profile.
	Profile playermodel.Profile
}

// managerAssertion verifies Service implements Manager.
var managerAssertion Manager = (*Service)(nil)
