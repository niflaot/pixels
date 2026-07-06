// Package units contains the UNIT outbound packet.
package units

import "github.com/niflaot/pixels/networking/codec"

const (
	// Header is the UNIT packet identifier.
	Header uint16 = 374

	// AvatarType is the Nitro avatar unit type.
	AvatarType int32 = 1
)

// Unit stores one avatar unit record.
type Unit struct {
	// UserID stores the durable player id.
	UserID int64

	// Name stores the visible player name.
	Name string

	// Motto stores the player motto.
	Motto string

	// Figure stores the Nitro figure string.
	Figure string

	// RoomIndex stores the room-local unit id.
	RoomIndex int64

	// X stores the unit tile x coordinate.
	X int32

	// Y stores the unit tile y coordinate.
	Y int32

	// Z stores the unit vertical height.
	Z string

	// Direction stores the body direction.
	Direction int32

	// Gender stores the avatar gender code.
	Gender string

	// GroupID stores the active group id.
	GroupID int32

	// GroupStatus stores the active group status.
	GroupStatus int32

	// GroupName stores the active group name.
	GroupName string

	// SwimFigure stores the optional swim figure.
	SwimFigure string

	// AchievementScore stores the visible achievement score.
	AchievementScore int32

	// Moderator reports whether the user has moderator badge state.
	Moderator bool
}

// Encode creates a UNIT packet.
func Encode(records []Unit) (codec.Packet, error) {
	payload, err := codec.AppendPayload(nil, codec.Definition{codec.Int32Field}, codec.Int32(int32(len(records))))
	if err != nil {
		return codec.Packet{}, err
	}
	for _, record := range records {
		payload, err = appendUnit(payload, record)
		if err != nil {
			return codec.Packet{}, err
		}
	}

	return codec.Packet{Header: Header, Payload: payload}, nil
}

// appendUnit appends one avatar unit.
func appendUnit(dst []byte, record Unit) ([]byte, error) {
	return codec.AppendPayload(dst, unitDefinition(),
		codec.Int32(int32(record.UserID)),
		codec.String(record.Name),
		codec.String(record.Motto),
		codec.String(record.Figure),
		codec.Int32(int32(record.RoomIndex)),
		codec.Int32(record.X),
		codec.Int32(record.Y),
		codec.String(record.Z),
		codec.Int32(record.Direction),
		codec.Int32(AvatarType),
		codec.String(record.Gender),
		codec.Int32(record.GroupID),
		codec.Int32(record.GroupStatus),
		codec.String(record.GroupName),
		codec.String(record.SwimFigure),
		codec.Int32(record.AchievementScore),
		codec.Bool(record.Moderator),
	)
}

// unitDefinition returns the avatar unit field order.
func unitDefinition() codec.Definition {
	return codec.Definition{
		codec.Named("userId", codec.Int32Field),
		codec.Named("name", codec.StringField),
		codec.Named("custom", codec.StringField),
		codec.Named("figure", codec.StringField),
		codec.Named("roomIndex", codec.Int32Field),
		codec.Named("x", codec.Int32Field),
		codec.Named("y", codec.Int32Field),
		codec.Named("z", codec.StringField),
		codec.Named("direction", codec.Int32Field),
		codec.Named("type", codec.Int32Field),
		codec.Named("sex", codec.StringField),
		codec.Named("groupId", codec.Int32Field),
		codec.Named("groupStatus", codec.Int32Field),
		codec.Named("groupName", codec.StringField),
		codec.Named("swimFigure", codec.StringField),
		codec.Named("activityPoints", codec.Int32Field),
		codec.Named("isModerator", codec.BooleanField),
	}
}
