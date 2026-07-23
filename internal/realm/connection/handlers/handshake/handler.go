// Package handshake contains connection handshake packet handlers.
package handshake

import (
	"context"
	"errors"

	"github.com/niflaot/pixels/networking/codec"
	netconn "github.com/niflaot/pixels/networking/connection"
	"github.com/niflaot/pixels/networking/crypto/diffie"
	indiffiecomplete "github.com/niflaot/pixels/networking/inbound/handshake/diffie/complete"
	indiffieinit "github.com/niflaot/pixels/networking/inbound/handshake/diffie/init"
	inpolicy "github.com/niflaot/pixels/networking/inbound/handshake/policy"
	inrelease "github.com/niflaot/pixels/networking/inbound/handshake/release"
	invariables "github.com/niflaot/pixels/networking/inbound/handshake/variables"
	outdiffiecomplete "github.com/niflaot/pixels/networking/outbound/handshake/diffie/complete"
	outdiffieinit "github.com/niflaot/pixels/networking/outbound/handshake/diffie/init"
)

var (
	// ErrDiffieUnavailable reports missing Diffie support.
	ErrDiffieUnavailable = errors.New("diffie unavailable")
)

// Handler coordinates optional per-session legacy Diffie compatibility.
type Handler struct {
	// Factory creates isolated negotiating channels.
	Factory *diffie.Factory
}

// Register adds handshake handlers to a registry.
func Register(registry *netconn.HandlerRegistry, factory ...*diffie.Factory) {
	handler := Handler{}
	if len(factory) > 0 {
		handler.Factory = factory[0]
	}
	early := []netconn.HandlerOption{netconn.AllowStates(netconn.StateCreated, netconn.StateHandshaking), netconn.AllowUnauthenticated()}
	_ = registry.Register(inrelease.Header, Release, early...)
	_ = registry.Register(invariables.Header, Variables, early...)
	_ = registry.Register(inpolicy.Header, Policy, early...)
	_ = registry.Register(indiffieinit.Header, handler.DiffieInit, early...)
	_ = registry.Register(indiffiecomplete.Header, handler.DiffieComplete, netconn.AllowStates(netconn.StateSecuring), netconn.AllowUnauthenticated())
}

// Release handles client release metadata.
func Release(context netconn.Context, packet codec.Packet) error {
	_, err := inrelease.Decode(packet)
	// TODO: Store this on connection.
	return err
}

// Variables handles client variable metadata.
func Variables(context netconn.Context, packet codec.Packet) error {
	_, err := invariables.Decode(packet)
	// TODO: Store this on connection.
	return err
}

// Policy handles client policy probes.
func Policy(context netconn.Context, packet codec.Packet) error {
	_, err := inpolicy.Decode(packet)
	// TODO: Ideate an implementaiton of policy handling.
	return err
}

// DiffieInit starts a configured legacy Diffie exchange.
func (handler Handler) DiffieInit(connection netconn.Context, packet codec.Packet) error {
	if _, err := indiffieinit.Decode(packet); err != nil {
		return err
	}
	if handler.Factory == nil || !handler.Factory.Enabled() {
		return disconnectDiffie(connection, ErrDiffieUnavailable)
	}
	if err := connection.Transition(netconn.EventDiffieRequested); err != nil {
		return err
	}
	channel, err := handler.Factory.New()
	if err != nil {
		return disconnectDiffie(connection, err)
	}
	if err = connection.BeginSecurity(context.Background(), channel); err != nil {
		return disconnectDiffie(connection, err)
	}
	parameters, err := channel.Parameters()
	if err != nil {
		return disconnectDiffie(connection, err)
	}
	response, err := outdiffieinit.Encode(parameters.EncryptedPrime, parameters.EncryptedGenerator)
	if err != nil {
		return disconnectDiffie(connection, err)
	}

	return connection.Send(context.Background(), response)
}

// DiffieComplete prepares RC4 and queues activation after the plaintext response.
func (handler Handler) DiffieComplete(connection netconn.Context, packet codec.Packet) error {
	payload, err := indiffiecomplete.Decode(packet)
	if err != nil {
		return err
	}
	channel, ok := connection.Security().(*diffie.Channel)
	if !ok {
		return disconnectDiffie(connection, ErrDiffieUnavailable)
	}
	publicKey, err := diffie.NewPublicKey(payload.EncryptedPublicKey)
	if err != nil {
		return disconnectDiffie(connection, err)
	}
	result, err := channel.Complete(context.Background(), publicKey)
	if err != nil {
		return disconnectDiffie(connection, err)
	}
	options := make([]outdiffiecomplete.Option, 0, 1)
	options = append(options, outdiffiecomplete.WithServerClientEncryption(result.ServerClientEncryption))
	response, err := outdiffiecomplete.Encode(result.PublicKey.Encrypted, options...)
	if err != nil {
		return disconnectDiffie(connection, err)
	}
	if err = connection.CompleteSecurity(context.Background(), response, channel); err != nil {
		return disconnectDiffie(connection, err)
	}

	return connection.Transition(netconn.EventDiffieCompleted)
}

// disconnectDiffie closes a session after a failed or unavailable negotiation.
func disconnectDiffie(connection netconn.Context, cause error) error {
	_ = connection.Transition(netconn.EventProtocolFailed)

	return connection.Disconnect(context.Background(), netconn.Reason{Code: netconn.DisconnectProtocolError, Message: cause.Error()})
}
