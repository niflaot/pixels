package enter

import (
	"context"

	"github.com/niflaot/pixels/internal/realm/room/layout"
	roommodel "github.com/niflaot/pixels/internal/realm/room/model"
	netconn "github.com/niflaot/pixels/networking/connection"
	outentrytile "github.com/niflaot/pixels/networking/outbound/room/entrytile"
	outmodel "github.com/niflaot/pixels/networking/outbound/room/model"
	outmodelname "github.com/niflaot/pixels/networking/outbound/room/modelname"
)

const (
	// DefaultWallHeight stores the initial room wall height.
	DefaultWallHeight int32 = 0
)

// SendModel sends room model name and heightmap packets.
func SendModel(ctx context.Context, connection netconn.Context, room roommodel.Room, roomLayout layout.Layout) error {
	namePacket, err := outmodelname.Encode(room.ModelName, int32(room.ID))
	if err != nil {
		return err
	}
	if err := connection.Send(ctx, namePacket); err != nil {
		return err
	}

	entryPacket, err := outentrytile.Encode(int32(roomLayout.DoorX), int32(roomLayout.DoorY), int32(roomLayout.DoorDirection))
	if err != nil {
		return err
	}
	if err := connection.Send(ctx, entryPacket); err != nil {
		return err
	}

	modelPacket, err := outmodel.Encode(true, DefaultWallHeight, roomLayout.Heightmap)
	if err != nil {
		return err
	}

	return connection.Send(ctx, modelPacket)
}
