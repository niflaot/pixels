package connection

import (
	"errors"
	"testing"

	"github.com/niflaot/pixels/networking/codec"
)

// TestHandlerRegistryHandle verifies registered handler command emission.
func TestHandlerRegistryHandle(t *testing.T) {
	registry := NewHandlerRegistry()
	handler := func(context Context, packet codec.Packet) ([]Command, error) {
		return []Command{NewCommand("handled", context, packet, nil)}, nil
	}

	if err := registry.Register(7, handler); err != nil {
		t.Fatalf("register handler: %v", err)
	}

	commands, err := registry.Handle(Context{ConnectionID: "one"}, codec.Packet{Header: 7})
	if err != nil {
		t.Fatalf("handle packet: %v", err)
	}

	if len(commands) != 1 {
		t.Fatalf("expected %d commands, got %d", 1, len(commands))
	}
}

// TestHandlerRegistryRejectsDuplicate verifies duplicate handler protection.
func TestHandlerRegistryRejectsDuplicate(t *testing.T) {
	registry := NewHandlerRegistry()
	handler := func(Context, codec.Packet) ([]Command, error) {
		return nil, nil
	}

	if err := registry.Register(7, handler); err != nil {
		t.Fatalf("register handler: %v", err)
	}

	if err := registry.Register(7, handler); !errors.Is(err, ErrHandlerExists) {
		t.Fatalf("expected handler exists, got %v", err)
	}
}

// TestHandlerRegistryFallback verifies fallback packet handling.
func TestHandlerRegistryFallback(t *testing.T) {
	registry := NewHandlerRegistry()
	registry.SetFallback(func(context Context, packet codec.Packet) ([]Command, error) {
		return []Command{NewCommand("fallback", context, packet, nil)}, nil
	})

	commands, err := registry.Handle(Context{}, codec.Packet{Header: 99})
	if err != nil {
		t.Fatalf("handle fallback: %v", err)
	}

	if commands[0].Name != "fallback" {
		t.Fatalf("expected fallback command, got %s", commands[0].Name)
	}
}

// TestHandlerRegistryMissing verifies missing handler errors.
func TestHandlerRegistryMissing(t *testing.T) {
	registry := NewHandlerRegistry()
	_, err := registry.Handle(Context{}, codec.Packet{Header: 99})
	if !errors.Is(err, ErrHandlerNotFound) {
		t.Fatalf("expected handler missing, got %v", err)
	}
}

// TestHandlerRegistryUnregister verifies handler removal.
func TestHandlerRegistryUnregister(t *testing.T) {
	registry := NewHandlerRegistry()
	handler := func(Context, codec.Packet) ([]Command, error) {
		return nil, nil
	}

	if err := registry.Register(7, handler); err != nil {
		t.Fatalf("register handler: %v", err)
	}

	if registry.Len() != 1 {
		t.Fatalf("expected %d handlers, got %d", 1, registry.Len())
	}

	if !registry.Unregister(7) {
		t.Fatal("expected handler removal")
	}

	if registry.Unregister(7) {
		t.Fatal("expected missing handler removal")
	}
}
