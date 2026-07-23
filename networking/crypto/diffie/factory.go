package diffie

import (
	"errors"
	"fmt"
)

var (
	// ErrDisabled reports a negotiation request while compatibility is disabled.
	ErrDisabled = errors.New("diffie compatibility disabled")
	// ErrInvalidConfig reports incompatible or malformed Diffie configuration.
	ErrInvalidConfig = errors.New("invalid diffie configuration")
	// ErrInvalidSharedKey reports an unusable negotiated shared key.
	ErrInvalidSharedKey = errors.New("invalid diffie shared key")
	// ErrInvalidState reports a Diffie operation in the wrong phase.
	ErrInvalidState = errors.New("invalid diffie state")
)

// Factory creates isolated legacy handshake channels for WebSocket sessions.
type Factory struct {
	// config stores normalized compatibility settings.
	config Config
	// rsa stores validated server key material when enabled.
	rsa *rsaKey
}

// NewFactory validates configuration and prepares a channel factory.
func NewFactory(config Config) (*Factory, error) {
	config = config.Normalize()
	if err := config.Validate(); err != nil {
		return nil, err
	}
	factory := &Factory{config: config}
	if !config.Enabled {
		return factory, nil
	}
	key, err := newRSAKey(config.RSAExponent, config.RSAModulus, config.RSAPrivateExponent)
	if err != nil {
		return nil, fmt.Errorf("prepare diffie RSA: %w", err)
	}
	factory.rsa = key

	return factory, nil
}

// Enabled reports whether legacy negotiation requests are accepted.
func (factory *Factory) Enabled() bool {
	return factory != nil && factory.config.Enabled && factory.rsa != nil
}

// New creates a fresh per-session negotiating channel.
func (factory *Factory) New() (*Channel, error) {
	if !factory.Enabled() {
		return nil, ErrDisabled
	}

	return newChannel(factory.config, factory.rsa), nil
}
