package profile

import (
	"testing"

	"github.com/niflaot/pixels/networking/codec"
)

// TestEncodeUsesNativeProfileShape verifies USER_PROFILE wire fields.
func TestEncodeUsesNativeProfileShape(t *testing.T) {
	packet, err := Encode(7, "demo", "look", "motto", "01-01-2026", 3, true, false, true, 10, true)
	values, decodeErr := codec.DecodePacketExact(packet, codec.Definition{
		codec.Int32Field, codec.StringField, codec.StringField, codec.StringField, codec.StringField,
		codec.Int32Field, codec.Int32Field, codec.BooleanField, codec.BooleanField, codec.BooleanField,
		codec.Int32Field, codec.Int32Field, codec.BooleanField,
	})
	if err != nil || decodeErr != nil || values[6].Int32 != 3 || !values[12].Boolean {
		t.Fatalf("unexpected values=%#v err=%v decode=%v", values, err, decodeErr)
	}
}
