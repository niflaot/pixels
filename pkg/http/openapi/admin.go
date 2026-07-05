package openapi

import "net/http"

// adminOperations returns protected connection administration operations.
func adminOperations() []operation {
	return []operation{
		adminRead(http.MethodGet, "/api/admin/connections", "List connections", new(ConnectionListRequest), new(ConnectionListResponse)),
		adminRead(http.MethodGet, "/api/admin/connections/count", "Count connections", new(ConnectionCountRequest), new(ConnectionCountResponse)),
		adminRead(http.MethodGet, "/api/admin/connections/reasons", "List disconnect reasons", new(APIKeyRequest), new(ReasonsResponse)),
		adminDisconnect("/api/admin/connections/disconnect", "Disconnect all connections", new(DisconnectAllRequest)),
		adminDisconnect("/api/admin/connections/{kind}/disconnect", "Disconnect connections by kind", new(DisconnectKindRequest)),
		adminDisconnect("/api/admin/connections/{kind}/{id}/disconnect", "Disconnect one connection", new(DisconnectOneRequest)),
	}
}

// adminRead creates a read-only admin operation.
func adminRead(method string, path string, summary string, request any, body any) operation {
	return operation{
		method:      method,
		path:        path,
		tag:         "Admin Connections",
		summary:     summary,
		description: summary + ".",
		request:     request,
		responses: append(
			[]response{jsonResponse(http.StatusOK, body, summary+".")},
			errorResponses(http.StatusUnauthorized)...,
		),
		secured: true,
	}
}

// adminDisconnect creates a disconnect admin operation.
func adminDisconnect(path string, summary string, request any) operation {
	return operation{
		method:      http.MethodPost,
		path:        path,
		tag:         "Admin Connections",
		summary:     summary,
		description: summary + ".",
		request:     request,
		responses: append(
			[]response{jsonResponse(http.StatusOK, new(DisconnectResponse), "Connections disconnected.")},
			errorResponses(http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound)...,
		),
		secured: true,
	}
}
