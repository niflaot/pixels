package openapi

import "net/http"

// ssoOperations returns protected SSO route operations.
func ssoOperations() []operation {
	return []operation{
		{
			method:      http.MethodPost,
			path:        "/api/sso/tickets",
			tag:         "SSO",
			summary:     "Create SSO ticket",
			description: "Creates a Redis-backed one-time SSO ticket for the configured TTL.",
			request:     new(CreateSSOTicketRequest),
			responses: append(
				[]response{jsonResponse(http.StatusCreated, new(CreateSSOTicketResponse), "SSO ticket created.")},
				errorResponses(http.StatusBadRequest, http.StatusUnauthorized, http.StatusInternalServerError)...,
			),
			secured: true,
		},
	}
}
