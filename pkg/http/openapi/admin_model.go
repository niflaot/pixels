package openapi

// ConnectionCountRequest contains optional count filters.
type ConnectionCountRequest struct {
	APIKeyRequest
	// Kind stores the optional connection kind filter.
	Kind string `query:"kind"`
}

// ConnectionListRequest contains optional list filters.
type ConnectionListRequest struct {
	APIKeyRequest
	// Kind stores the optional connection kind filter.
	Kind string `query:"kind"`
}

// ConnectionCountResponse contains connection count data.
type ConnectionCountResponse struct {
	// Total stores the total active connection count.
	Total int `json:"total" required:"true"`
	// Kind stores the optional filtered connection kind.
	Kind string `json:"kind,omitempty"`
	// Count stores the optional filtered kind count.
	Count *int `json:"count,omitempty"`
}

// ConnectionListResponse contains connection list data.
type ConnectionListResponse struct {
	// Total stores the returned connection count.
	Total int `json:"total" required:"true"`
	// Items stores safe connection rows.
	Items []ConnectionResponse `json:"items" required:"true"`
}

// ConnectionResponse contains safe connection data.
type ConnectionResponse struct {
	// ID stores the connection id.
	ID string `json:"id" required:"true"`
	// Kind stores the connection kind.
	Kind string `json:"kind" required:"true" example:"websocket"`
	// State stores the lifecycle state label.
	State string `json:"state" required:"true" example:"connected"`
	// StartedAt stores the session start time.
	StartedAt string `json:"startedAt" required:"true" format:"date-time"`
	// AuthenticatedAt stores the optional authentication time.
	AuthenticatedAt string `json:"authenticatedAt,omitempty" format:"date-time"`
}

// ReasonsResponse contains supported disconnect reasons.
type ReasonsResponse struct {
	// Items stores supported reason rows.
	Items []ReasonResponse `json:"items" required:"true"`
}

// ReasonResponse contains one supported disconnect reason.
type ReasonResponse struct {
	// Code stores the numeric disconnect code.
	Code uint16 `json:"code" required:"true"`
	// Reason stores the stable disconnect reason label.
	Reason string `json:"reason" required:"true"`
}

// PlayerNotificationRequest contains one localized notification request.
type PlayerNotificationRequest struct {
	APIKeyRequest
	// ID stores the target player id.
	ID int64 `path:"id" required:"true"`
	// Kind stores the notification kind.
	Kind string `json:"kind" enum:"bubble,alert" default:"bubble"`
	// Key stores the required i18n message key.
	Key string `json:"key" required:"true" example:"admin.notification.default"`
	// Locale stores an optional locale override.
	Locale string `json:"locale,omitempty" example:"es"`
	// BubbleKey stores an optional bubble alert type.
	BubbleKey string `json:"bubbleKey,omitempty" example:"admin.notification"`
	// Params stores optional translation parameters.
	Params map[string]string `json:"params,omitempty"`
}

// PlayerNotificationResponse contains delivery status.
type PlayerNotificationResponse struct {
	// PlayerID stores the target player id.
	PlayerID int64 `json:"playerId" required:"true"`
	// Kind stores the delivered notification kind.
	Kind string `json:"kind" required:"true"`
	// Key stores the i18n message key.
	Key string `json:"key" required:"true"`
	// Sent reports whether the packet was sent.
	Sent bool `json:"sent" required:"true"`
}
