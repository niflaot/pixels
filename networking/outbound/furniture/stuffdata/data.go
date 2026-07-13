// Package stuffdata encodes Nitro furniture object data fragments.
package stuffdata

import "github.com/niflaot/pixels/networking/codec"

const (
	// mapFormat identifies Nitro map-style furniture object data.
	mapFormat int32 = 1
)

// Pair stores one object-data key and value.
type Pair struct {
	// Key stores the object-data key.
	Key string
	// Value stores the object-data value.
	Value string
}

// AppendMap appends map-style furniture object data.
func AppendMap(dst []byte, pairs []Pair) ([]byte, error) {
	payload, err := codec.AppendPayload(dst, codec.Definition{codec.Int32Field, codec.Int32Field},
		codec.Int32(mapFormat), codec.Int32(int32(len(pairs))))
	if err != nil {
		return dst, err
	}
	for _, pair := range pairs {
		payload, err = codec.AppendPayload(payload, codec.Definition{codec.StringField, codec.StringField},
			codec.String(pair.Key), codec.String(pair.Value))
		if err != nil {
			return dst, err
		}
	}

	return payload, nil
}
