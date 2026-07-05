package connection

import "errors"

var (
	// ErrConnectionExists reports an already registered connection id for a kind.
	ErrConnectionExists = errors.New("connection exists")

	// ErrConnectionNotFound reports a missing connection id for a kind.
	ErrConnectionNotFound = errors.New("connection not found")

	// ErrDisposed reports an operation attempted after disposal.
	ErrDisposed = errors.New("connection disposed")

	// ErrHandlerExists reports an already registered handler for a packet header.
	ErrHandlerExists = errors.New("handler exists")

	// ErrHandlerNotFound reports a missing handler for a packet header.
	ErrHandlerNotFound = errors.New("handler not found")

	// ErrInvalidConnection reports an invalid connection value.
	ErrInvalidConnection = errors.New("invalid connection")

	// ErrInvalidConnectionConfig reports an invalid session configuration.
	ErrInvalidConnectionConfig = errors.New("invalid connection config")

	// ErrInvalidHandler reports an invalid packet handler.
	ErrInvalidHandler = errors.New("invalid handler")
)
