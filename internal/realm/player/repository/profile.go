package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	playermodel "github.com/niflaot/pixels/internal/realm/player/model"
)

const (
	// createProfileSQL inserts a player profile record.
	createProfileSQL = `
insert into player_profiles (player_id, look, gender, motto, home_room_id, allow_name_change)
values ($1, $2, $3, $4, $5, $6)
returning player_id, look, gender, motto, home_room_id, allow_name_change, created_at, updated_at, version`

	// findProfileByPlayerIDSQL reads one player profile by player id.
	findProfileByPlayerIDSQL = `
select player_id, look, gender, motto, home_room_id, allow_name_change, created_at, updated_at, version
from player_profiles
where player_id = $1`
)

// CreateProfileParams contains profile creation data.
type CreateProfileParams struct {
	// PlayerID is the owning player identifier.
	PlayerID int64

	// Look is the Nitro avatar figure string.
	Look string

	// Gender is the Nitro avatar gender code.
	Gender playermodel.Gender

	// Motto is the public player motto.
	Motto string

	// HomeRoomID is the optional default home room identifier.
	HomeRoomID *int64

	// AllowNameChange reports whether the player can change username.
	AllowNameChange bool
}

// CreateProfile creates a player profile record.
func (repository *Repository) CreateProfile(ctx context.Context, params CreateProfileParams) (playermodel.Profile, error) {
	if !params.Gender.Valid() {
		return playermodel.Profile{}, ErrInvalidGender
	}

	profile, err := scanProfile(repository.executor.QueryRow(ctx, createProfileSQL, params.PlayerID, params.Look, string(params.Gender), params.Motto, params.HomeRoomID, params.AllowNameChange))
	if err != nil {
		return playermodel.Profile{}, fmt.Errorf("create player profile: %w", err)
	}

	return profile, nil
}

// FindProfileByPlayerID finds a profile by player id.
func (repository *Repository) FindProfileByPlayerID(ctx context.Context, playerID int64) (playermodel.Profile, bool, error) {
	profile, err := scanProfile(repository.executor.QueryRow(ctx, findProfileByPlayerIDSQL, playerID))
	if errors.Is(err, pgx.ErrNoRows) {
		return playermodel.Profile{}, false, nil
	}

	if err != nil {
		return playermodel.Profile{}, false, fmt.Errorf("find player profile by player id: %w", err)
	}

	return profile, true, nil
}
