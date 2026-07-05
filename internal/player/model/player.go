// Package model contains persistent player records.
package model

import (
	"time"

	sharedmodel "github.com/niflaot/pixels/pkg/model"
)

// Player contains durable player identity fields.
type Player struct {
	// Base contains shared durable record fields.
	sharedmodel.Base

	// Username is the unique visible player name.
	Username string

	// LastLoginAt is the last successful login time.
	LastLoginAt *time.Time

	// LastLogoutAt is the last recorded logout time.
	LastLogoutAt *time.Time

	// LastSeenAt is the last time the player was seen by profile systems.
	LastSeenAt *time.Time
}
