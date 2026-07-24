package admin

import (
	"context"
	"strings"
	"testing"
	"time"

	plugincommand "github.com/niflaot/pixels/internal/plugin/command"
	"github.com/niflaot/pixels/internal/plugin/loader"
	"github.com/niflaot/pixels/networking/codec"
	"github.com/niflaot/pixels/pkg/build"
	sdkplayer "github.com/niflaot/pixels/sdk/player"
	"go.uber.org/zap"
)

// commandPlayerAccess records tree feedback and permission decisions.
type commandPlayerAccess struct {
	// allowed controls command permission.
	allowed bool
	// message stores the latest feedback.
	message string
}

// TestRegisteredCommandsExecuteEveryCorePath verifies the complete command tree.
func TestRegisteredCommandsExecuteEveryCorePath(t *testing.T) {
	fixture := newServiceFixture()
	packets := make([]codec.Packet, 0)
	addServicePlayer(t, fixture, 7, "admin", nil, &packets)
	addServicePlayer(t, fixture, 8, "target", nil, &packets)
	access := &commandPlayerAccess{allowed: true}
	tree := plugincommand.NewTree(":", time.Second, nil, zap.NewNop())
	tree.SetPlayers(access)
	if err := RegisterCommands(tree, fixture.service); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	player := sdkplayer.Player{ID: 7, Username: "admin", Online: true}
	for _, command := range []string{":alert target Please behave", ":halert Maintenance soon", ":trace", ":trace"} {
		handled, err := tree.Execute(context.Background(), player, command)
		if err != nil || !handled {
			t.Fatalf("command=%q handled=%v err=%v", command, handled, err)
		}
	}
	if len(packets) != 3 || fixture.trace.activates != 1 || fixture.trace.finalizes != 1 {
		t.Fatalf("packets=%d activates=%d finalizes=%d", len(packets), fixture.trace.activates, fixture.trace.finalizes)
	}
}

// Message records command feedback.
func (access *commandPlayerAccess) Message(_ int64, message string) error {
	access.message = message
	return nil
}

// HasPermission returns the configured decision.
func (access *commandPlayerAccess) HasPermission(int64, string) (bool, error) {
	return access.allowed, nil
}

// TestRegisterCommandsExecutesPermissionGatedAbout verifies core tree integration.
func TestRegisterCommandsExecutesPermissionGatedAbout(t *testing.T) {
	players := &commandPlayerAccess{}
	tree := plugincommand.NewTree(":", time.Second, nil, zap.NewNop())
	tree.SetPlayers(players)
	service := &Service{build: build.NewInfo("pixels", "v0.0.3", "123456789"), plugins: &loader.Loader{}, log: zap.NewNop()}
	if err := RegisterCommands(tree, service); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	player := sdkplayer.Player{ID: 7, Username: "admin", Online: true}
	handled, err := tree.Execute(context.Background(), player, ":about")
	if err != nil || !handled || !strings.Contains(players.message, "permiso") {
		t.Fatalf("denied handled=%v message=%q err=%v", handled, players.message, err)
	}
	players.allowed = true
	handled, err = tree.Execute(context.Background(), player, ":about")
	if err != nil || !handled || !strings.Contains(players.message, "v0.0.3") {
		t.Fatalf("allowed handled=%v message=%q err=%v", handled, players.message, err)
	}
}
