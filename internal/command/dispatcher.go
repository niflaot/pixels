package command

import (
	"context"
	"time"
)

// Dispatcher validates and dispatches one typed command kind.
type Dispatcher[T Command] struct {
	// handler receives valid command envelopes.
	handler Handler[T]
}

// NewDispatcher creates a command dispatcher.
func NewDispatcher[T Command](handler Handler[T], middleware ...Middleware[T]) (*Dispatcher[T], error) {
	if handler == nil {
		return nil, ErrInvalidHandler
	}

	return &Dispatcher[T]{handler: Chain(handler, middleware...)}, nil
}

// Dispatch sends an envelope to the configured handler.
func (dispatcher *Dispatcher[T]) Dispatch(ctx context.Context, envelope Envelope[T]) error {
	envelope = envelope.WithCreatedAt(time.Now())
	if !envelope.Valid() {
		return ErrInvalidName
	}

	return dispatcher.handler.Handle(ctx, envelope)
}
