package admin

import "github.com/niflaot/pixels/internal/permission"

var (
	// AlertPermission allows sending a direct hotel alert.
	AlertPermission = permission.RegisterNode("admin.alert", "")
	// HotelAlertPermission allows broadcasting a hotel-wide alert.
	HotelAlertPermission = permission.RegisterNode("admin.halert", "")
	// AboutPermission allows viewing private build and plugin metadata.
	AboutPermission = permission.RegisterNode("admin.about", "")
	// TracePermission allows capturing production protocol traffic.
	TracePermission = permission.RegisterNode("admin.trace", "")
)
