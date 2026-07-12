// Package settings contains the USER_SETTINGS outbound packet.
package settings

import "github.com/niflaot/pixels/networking/codec"

const (
	// Header identifies USER_SETTINGS.
	Header uint16 = 513
)

// Definition describes Nitro user settings including selected chat style.
var Definition = codec.Definition{
	codec.Named("volumeSystem", codec.Int32Field), codec.Named("volumeFurniture", codec.Int32Field),
	codec.Named("volumeTrax", codec.Int32Field), codec.Named("oldChat", codec.BooleanField),
	codec.Named("roomInvitesBlocked", codec.BooleanField), codec.Named("cameraFollowBlocked", codec.BooleanField),
	codec.Named("flags", codec.Int32Field), codec.Named("chatStyle", codec.Int32Field),
}

// Option configures USER_SETTINGS fields owned by another capability.
type Option func(*options)

// options stores optional USER_SETTINGS capability values.
type options struct {
	// roomInvitesBlocked stores messenger room-invite privacy.
	roomInvitesBlocked bool
}

// WithRoomInvitesBlocked includes messenger room-invite privacy.
func WithRoomInvitesBlocked(blocked bool) Option {
	return func(configured *options) {
		configured.roomInvitesBlocked = blocked
	}
}

// Encode creates USER_SETTINGS with persisted cross-capability settings.
func Encode(chatStyle int32, optionFunctions ...Option) (codec.Packet, error) {
	configured := options{}
	for _, option := range optionFunctions {
		option(&configured)
	}
	return codec.NewPacket(Header, Definition,
		codec.Int32(100), codec.Int32(100), codec.Int32(100),
		codec.Bool(false), codec.Bool(configured.roomInvitesBlocked), codec.Bool(false), codec.Int32(0), codec.Int32(chatStyle),
	)
}
