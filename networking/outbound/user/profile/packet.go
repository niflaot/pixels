// Package profile contains the USER_PROFILE outbound packet.
package profile

import "github.com/niflaot/pixels/networking/codec"

// Header identifies USER_PROFILE.
const Header uint16 = 3898

// Encode creates USER_PROFILE with groups deferred until the groups realm exists.
func Encode(playerID int64, username string, look string, motto string, registration string, friendCount int32, isFriend bool, requestSent bool, online bool, secondsSinceLastVisit int32, openWindow bool) (codec.Packet, error) {
	return codec.NewPacket(Header, codec.Definition{
		codec.Int32Field, codec.StringField, codec.StringField, codec.StringField, codec.StringField,
		codec.Int32Field, codec.Int32Field, codec.BooleanField, codec.BooleanField, codec.BooleanField,
		codec.Int32Field, codec.Int32Field, codec.BooleanField,
	}, codec.Int32(int32(playerID)), codec.String(username), codec.String(look), codec.String(motto), codec.String(registration),
		codec.Int32(0), codec.Int32(friendCount), codec.Bool(isFriend), codec.Bool(requestSent), codec.Bool(online),
		codec.Int32(0), codec.Int32(secondsSinceLastVisit), codec.Bool(openWindow))
}
