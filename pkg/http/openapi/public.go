package openapi

import "net/http"

// publicOperations returns public route operations.
func publicOperations() []operation {
	return []operation{
		{
			method:      http.MethodGet,
			path:        "/status",
			tag:         "Public",
			summary:     "Read server status",
			description: "Returns public runtime status without requiring an API key.",
			responses:   []response{jsonResponse(http.StatusOK, new(StatusResponse), "Server status.")},
		},
		{
			method:      http.MethodGet,
			path:        "/ws",
			tag:         "WebSocket",
			summary:     "Open websocket session",
			description: "Upgrades an HTTP request to the pixel-protocol websocket entrypoint.",
			request:     new(WebSocketUpgradeRequest),
			responses: append(
				[]response{emptyResponse(http.StatusSwitchingProtocols, "Websocket upgrade accepted.")},
				errorResponses(http.StatusUpgradeRequired)...,
			),
		},
		{
			method:      http.MethodGet,
			path:        "/docs",
			tag:         "Public",
			summary:     "Read Scalar API documentation",
			description: "Serves public Scalar documentation in development only.",
			responses: []response{
				{status: http.StatusOK, body: "", description: "Scalar documentation HTML.", contentType: "text/html"},
				jsonResponse(http.StatusNotFound, new(ErrorResponse), "Documentation is disabled outside development."),
			},
		},
	}
}

// fallbackOperation returns the authenticated fallback route operation.
func fallbackOperation() operation {
	return operation{
		method:      http.MethodGet,
		path:        "/*",
		tag:         "Fallback",
		summary:     "Private route fallback",
		description: "Represents protected endpoints added after public route registration.",
		request:     new(APIKeyRequest),
		responses:   errorResponses(http.StatusUnauthorized, http.StatusNotFound),
		secured:     true,
	}
}
