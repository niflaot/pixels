package security

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/niflaot/pixels/internal/auth/sso"
	"github.com/niflaot/pixels/networking/codec"
	netconn "github.com/niflaot/pixels/networking/connection"
	inmachine "github.com/niflaot/pixels/networking/inbound/security/machine"
	inticket "github.com/niflaot/pixels/networking/inbound/security/ticket"
	outauth "github.com/niflaot/pixels/networking/outbound/authentication/ok"
	outmachine "github.com/niflaot/pixels/networking/outbound/security/machine"
	"github.com/niflaot/pixels/pkg/redis"
)

// TestMachineSendsReplacement verifies invalid machine ids are replaced.
func TestMachineSendsReplacement(t *testing.T) {
	session, sent := testSession(t, testSSO(t))

	if err := session.Receive(context.Background(), machinePacket(t, "~bad")); err != nil {
		t.Fatalf("receive machine: %v", err)
	}
	if len(*sent) != 1 || (*sent)[0].Header != outmachine.Header {
		t.Fatalf("expected machine replacement, got %#v", *sent)
	}
}

// TestMachineAcceptsValidMachine verifies accepted machine ids.
func TestMachineAcceptsValidMachine(t *testing.T) {
	session, sent := testSession(t, testSSO(t))

	if err := session.Receive(context.Background(), machinePacket(t, validMachineID())); err != nil {
		t.Fatalf("receive machine: %v", err)
	}
	if len(*sent) != 0 {
		t.Fatalf("expected no response, got %#v", *sent)
	}
}

// TestTicketAuthenticates verifies SSO authentication.
func TestTicketAuthenticates(t *testing.T) {
	service := testSSO(t)
	ticket, err := service.Create(context.Background(), sso.CreateRequest{UserID: "todo-user", TTL: time.Minute})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	session, sent := testSession(t, service)
	if err := session.Receive(context.Background(), ticketPacket(t, ticket.Value)); err != nil {
		t.Fatalf("receive ticket: %v", err)
	}
	if session.State() != netconn.StateConnected {
		t.Fatalf("expected connected state, got %d", session.State())
	}
	if len(*sent) == 0 || (*sent)[0].Header != outauth.Header {
		t.Fatalf("expected authenticated packet, got %#v", *sent)
	}
}

// TestTicketRejectsInvalidTicket verifies failed authentication disposal.
func TestTicketRejectsInvalidTicket(t *testing.T) {
	session, _ := testSession(t, testSSO(t))

	if err := session.Receive(context.Background(), ticketPacket(t, "missing")); err != nil {
		t.Fatalf("receive ticket: %v", err)
	}
	if session.State() != netconn.StateClosed {
		t.Fatalf("expected closed session, got %d", session.State())
	}
}

// TestTicketRequiresSecurity verifies production encryption policy.
func TestTicketRequiresSecurity(t *testing.T) {
	service := testSSO(t)
	ticket, err := service.Create(context.Background(), sso.CreateRequest{UserID: "todo-user", TTL: time.Minute})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	session, _ := testSession(t, service)
	if err := session.SetSecurityPolicy(netconn.SecurityPolicy{Mode: netconn.SecurityRequired}); err != nil {
		t.Fatalf("set policy: %v", err)
	}
	if err := session.Receive(context.Background(), ticketPacket(t, ticket.Value)); err != netconn.ErrSecurityRequired {
		t.Fatalf("expected security required, got %v", err)
	}
	if session.State() != netconn.StateClosed {
		t.Fatalf("expected closed session, got %d", session.State())
	}
}

// TestHandlersRejectMalformedPayloads verifies decode failures.
func TestHandlersRejectMalformedPayloads(t *testing.T) {
	if err := Machine(netconn.Context{}, codec.Packet{Header: inmachine.Header}); err == nil {
		t.Fatal("expected machine decode failure")
	}
	if err := Ticket(testSSO(t))(netconn.Context{}, codec.Packet{Header: inticket.Header}); err == nil {
		t.Fatal("expected ticket decode failure")
	}
}

// testSession creates a security session.
func testSession(t *testing.T, service *sso.Service) (*netconn.Session, *[]codec.Packet) {
	t.Helper()
	inbound := netconn.NewHandlerRegistry()
	Register(inbound, service)
	outbound := netconn.NewHandlerRegistry()
	outbound.SetFallback(func(netconn.Context, codec.Packet) error {
		return nil
	}, netconn.AllowAnyActiveState(), netconn.AllowUnauthenticated())
	sent := make([]codec.Packet, 0)
	session, err := netconn.NewSession(netconn.SessionConfig{
		ID:       "security-test",
		Kind:     "websocket",
		Inbound:  inbound,
		Outbound: outbound,
		Sender: func(ctx context.Context, packet codec.Packet) error {
			sent = append(sent, packet)
			return nil
		},
		Disposer: func(context.Context, netconn.Reason) error {
			return nil
		},
	})
	if err != nil {
		t.Fatalf("new session: %v", err)
	}
	if err := session.Transition(netconn.EventPacketReceived); err != nil {
		t.Fatalf("packet transition: %v", err)
	}

	return session, &sent
}

// machinePacket creates a machine packet.
func machinePacket(t *testing.T, machineID string) codec.Packet {
	t.Helper()
	packet, err := codec.NewPacket(inmachine.Header, inmachine.Definition, codec.String(machineID), codec.String("fingerprint"), codec.String("capabilities"))
	if err != nil {
		t.Fatalf("new machine packet: %v", err)
	}

	return packet
}

// ticketPacket creates a security-ticket packet.
func ticketPacket(t *testing.T, ticket string) codec.Packet {
	t.Helper()
	packet, err := codec.NewPacket(inticket.Header, inticket.Definition, codec.String(ticket), codec.Int32(1))
	if err != nil {
		t.Fatalf("new ticket packet: %v", err)
	}

	return packet
}

// testSSO creates a test SSO service.
func testSSO(t *testing.T) *sso.Service {
	t.Helper()
	server := miniredis.RunT(t)
	client := redis.New(redis.Config{Address: server.Addr()})
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Fatalf("close redis: %v", err)
		}
	})

	return sso.New(sso.Config{DefaultTTL: time.Minute, Key: "test-key", Prefix: "pixels:sso"}, client)
}

// validMachineID returns a syntactically valid test machine id.
func validMachineID() string {
	return "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
}
