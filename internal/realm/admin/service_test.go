package admin

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/niflaot/pixels/internal/plugin/loader"
	admintrace "github.com/niflaot/pixels/internal/realm/admin/trace"
	playerlive "github.com/niflaot/pixels/internal/realm/player/live"
	"github.com/niflaot/pixels/internal/realm/session/binding"
	"github.com/niflaot/pixels/networking/codec"
	netconn "github.com/niflaot/pixels/networking/connection"
	"github.com/niflaot/pixels/pkg/build"
	sdkcommand "github.com/niflaot/pixels/sdk/command"
	"go.uber.org/zap"
)

// commandSender records direct service feedback.
type commandSender struct {
	// name stores the sender username.
	name string
	// replies stores feedback in order.
	replies []string
}

// Name returns the configured sender name.
func (sender *commandSender) Name() string { return sender.name }

// Kind identifies a player sender.
func (*commandSender) Kind() string { return sdkcommand.SenderKindPlayer }

// HasPermission grants direct service tests every node.
func (*commandSender) HasPermission(string) bool { return true }

// Reply records command feedback.
func (sender *commandSender) Reply(_ context.Context, message string) error {
	sender.replies = append(sender.replies, message)
	return nil
}

// lastReply returns the latest command feedback.
func (sender *commandSender) lastReply() string {
	if len(sender.replies) == 0 {
		return ""
	}

	return sender.replies[len(sender.replies)-1]
}

// traceManager records command-facing trace operations.
type traceManager struct {
	// active reports whether a trace exists.
	active bool
	// activates counts activation calls.
	activates int
	// finalizes counts finalization calls.
	finalizes int
}

// Activate records one trace activation.
func (manager *traceManager) Activate(_ context.Context, playerID int64, playerName string) (admintrace.Session, bool, error) {
	manager.active = true
	manager.activates++
	startedAt := time.Unix(100, 0)
	return admintrace.Session{PlayerID: playerID, PlayerName: playerName, StartedAt: startedAt, ExpiresAt: startedAt.Add(admintrace.DefaultDuration)}, true, nil
}

// Active returns configured trace state.
func (manager *traceManager) Active(int64) (admintrace.Session, bool) {
	return admintrace.Session{}, manager.active
}

// Finalize records one trace finalization.
func (manager *traceManager) Finalize(context.Context, int64, string) (admintrace.Result, error) {
	manager.active = false
	manager.finalizes++
	return admintrace.Result{URL: "https://storage.example/trace.txt"}, nil
}

// serviceFixture stores one administrative service graph.
type serviceFixture struct {
	// service is the tested command service.
	service *Service
	// players stores test live players.
	players *playerlive.Registry
	// bindings stores test session bindings.
	bindings *binding.Registry
	// connections stores test connections.
	connections *netconn.Registry
	// trace records trace operations.
	trace *traceManager
}

// newServiceFixture creates an empty administrative service graph.
func newServiceFixture() serviceFixture {
	players := playerlive.NewRegistry()
	bindings := binding.NewRegistry()
	connections := netconn.NewRegistry()
	traces := &traceManager{}
	service := New(build.NewInfo("pixels", "v0.0.3", "1234567890"), &loader.Loader{}, players, bindings, connections, nil, nil, zap.NewNop())
	service.tracer = traces
	return serviceFixture{service: service, players: players, bindings: bindings, connections: connections, trace: traces}
}

// addServicePlayer registers one live player and transport.
func addServicePlayer(t *testing.T, fixture serviceFixture, playerID int64, username string, sendErr error, packets *[]codec.Packet) {
	t.Helper()
	connectionID := netconn.ID(username + "-connection")
	peer, err := playerlive.NewSessionPeer(connectionID, "websocket", time.Now())
	if err != nil {
		t.Fatalf("new peer: %v", err)
	}
	player, err := playerlive.NewPlayer(playerlive.Snapshot{ID: playerID, Username: username}, peer)
	if err != nil {
		t.Fatalf("new player: %v", err)
	}
	if err = fixture.players.Add(player); err != nil {
		t.Fatalf("add player: %v", err)
	}
	if err = fixture.bindings.Add(binding.Binding{PlayerID: playerID, ConnectionID: connectionID, ConnectionKind: "websocket"}); err != nil {
		t.Fatalf("add binding: %v", err)
	}
	outbound := netconn.NewHandlerRegistry()
	outbound.SetFallback(func(netconn.Context, codec.Packet) error { return nil }, netconn.AllowAnyActiveState(), netconn.AllowUnauthenticated())
	session, err := netconn.NewSession(netconn.SessionConfig{
		ID: connectionID, Kind: "websocket", Outbound: outbound,
		Sender: func(_ context.Context, packet codec.Packet) error {
			if sendErr != nil {
				return sendErr
			}
			*packets = append(*packets, packet)
			return nil
		},
		Disposer: func(context.Context, netconn.Reason) error { return nil },
	})
	if err != nil {
		t.Fatalf("new session: %v", err)
	}
	if err = fixture.connections.Register(session); err != nil {
		t.Fatalf("register connection: %v", err)
	}
}

// TestAlertReportsOfflineAndDelivers verifies explicit lookup feedback and delivery.
func TestAlertReportsOfflineAndDelivers(t *testing.T) {
	fixture := newServiceFixture()
	sender := &commandSender{name: "admin"}
	if err := fixture.service.Alert(context.Background(), sender, "missing", "reason"); err != nil || !strings.Contains(sender.lastReply(), "no está conectado") {
		t.Fatalf("offline reply=%q err=%v", sender.lastReply(), err)
	}
	packets := make([]codec.Packet, 0, 1)
	addServicePlayer(t, fixture, 2, "Target", nil, &packets)
	if err := fixture.service.Alert(context.Background(), sender, "target", "Follow the rules"); err != nil {
		t.Fatalf("alert: %v", err)
	}
	if len(packets) != 1 || !strings.Contains(sender.lastReply(), "Target") {
		t.Fatalf("packets=%v reply=%q", packets, sender.lastReply())
	}
}

// TestAlertRejectsTheIssuingPlayer verifies direct alerts cannot target their sender.
func TestAlertRejectsTheIssuingPlayer(t *testing.T) {
	fixture := newServiceFixture()
	packets := make([]codec.Packet, 0, 1)
	addServicePlayer(t, fixture, 1, "admin", nil, &packets)
	sender := &commandSender{name: "admin"}
	if err := fixture.service.Alert(context.Background(), sender, "ADMIN", "self"); err != nil {
		t.Fatalf("self alert: %v", err)
	}
	if len(packets) != 0 || !strings.Contains(sender.lastReply(), "ti mismo") {
		t.Fatalf("packets=%d reply=%q", len(packets), sender.lastReply())
	}
}

// TestHotelAlertContinuesAfterDeliveryFailure verifies partial broadcast behavior.
func TestHotelAlertContinuesAfterDeliveryFailure(t *testing.T) {
	fixture := newServiceFixture()
	packets := make([]codec.Packet, 0, 1)
	addServicePlayer(t, fixture, 2, "broken", errors.New("disconnected"), &packets)
	addServicePlayer(t, fixture, 3, "healthy", nil, &packets)
	sender := &commandSender{name: "admin"}
	if err := fixture.service.HotelAlert(context.Background(), sender, "Maintenance"); err != nil {
		t.Fatalf("hotel alert: %v", err)
	}
	if len(packets) != 1 || !strings.Contains(sender.lastReply(), "1 jugadores; 1 fallaron") {
		t.Fatalf("packets=%d reply=%q", len(packets), sender.lastReply())
	}
}

// TestHotelAlertExcludesTheIssuer verifies a broadcast never echoes to its sender.
func TestHotelAlertExcludesTheIssuer(t *testing.T) {
	fixture := newServiceFixture()
	issuerPackets := make([]codec.Packet, 0, 1)
	targetPackets := make([]codec.Packet, 0, 1)
	addServicePlayer(t, fixture, 1, "admin", nil, &issuerPackets)
	addServicePlayer(t, fixture, 2, "target", nil, &targetPackets)
	sender := &commandSender{name: "admin"}
	if err := fixture.service.HotelAlert(context.Background(), sender, "Maintenance"); err != nil {
		t.Fatalf("hotel alert: %v", err)
	}
	if len(issuerPackets) != 0 || len(targetPackets) != 1 || !strings.Contains(sender.lastReply(), "1 jugadores; 0 fallaron") {
		t.Fatalf("issuer=%d target=%d reply=%q", len(issuerPackets), len(targetPackets), sender.lastReply())
	}
}

// TestAboutAndTraceToggle verifies metadata and trace lifecycle feedback.
func TestAboutAndTraceToggle(t *testing.T) {
	fixture := newServiceFixture()
	packets := make([]codec.Packet, 0)
	addServicePlayer(t, fixture, 7, "admin", nil, &packets)
	sender := &commandSender{name: "admin"}
	if err := fixture.service.About(context.Background(), sender); err != nil || !containsReply(sender.lastReply(), "v0.0.3", "12345678", "ninguno") {
		t.Fatalf("about reply=%q err=%v", sender.lastReply(), err)
	}
	if err := fixture.service.ToggleTrace(context.Background(), sender); err != nil || fixture.trace.activates != 1 {
		t.Fatalf("activate reply=%q err=%v", sender.lastReply(), err)
	}
	if err := fixture.service.ToggleTrace(context.Background(), sender); err != nil || fixture.trace.finalizes != 1 || !strings.Contains(sender.lastReply(), "https://storage.example/trace.txt") {
		t.Fatalf("finalize reply=%q err=%v", sender.lastReply(), err)
	}
}

// TestCommandFailureReturnsLocalizedFeedback verifies internal errors do not leak.
func TestCommandFailureReturnsLocalizedFeedback(t *testing.T) {
	fixture := newServiceFixture()
	sender := &commandSender{name: "admin"}
	reason := strings.Repeat("x", 70_000)
	if err := fixture.service.HotelAlert(context.Background(), sender, reason); err != nil {
		t.Fatalf("failure feedback: %v", err)
	}
	if !strings.Contains(sender.lastReply(), "No se pudo completar") {
		t.Fatalf("unexpected reply: %q", sender.lastReply())
	}
}

// containsReply reports whether one reply contains every fragment.
func containsReply(reply string, fragments ...string) bool {
	for _, fragment := range fragments {
		if !strings.Contains(reply, fragment) {
			return false
		}
	}

	return true
}
