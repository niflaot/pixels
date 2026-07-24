package trace

import (
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	playerlive "github.com/niflaot/pixels/internal/realm/player/live"
	"github.com/niflaot/pixels/internal/realm/session/binding"
	netconn "github.com/niflaot/pixels/networking/connection"
	"github.com/niflaot/pixels/pkg/redis"
	"go.uber.org/zap"
)

// traceUploader records finalized object uploads.
type traceUploader struct {
	// mutex protects captured uploads.
	mutex sync.Mutex
	// calls counts upload attempts.
	calls int
	// key stores the latest object key.
	key string
	// body stores the latest object body.
	body string
	// err stores an injected upload failure.
	err error
}

// Put records one trace object upload.
func (uploader *traceUploader) Put(_ context.Context, key string, body io.Reader, _ int64, _ string) (string, error) {
	uploader.mutex.Lock()
	defer uploader.mutex.Unlock()
	uploader.calls++
	uploader.key = key
	data, _ := io.ReadAll(body)
	uploader.body = string(data)
	if uploader.err != nil {
		return "", uploader.err
	}

	return "https://storage.example/" + key, nil
}

// snapshot returns stable upload state.
func (uploader *traceUploader) snapshot() (int, string, string) {
	uploader.mutex.Lock()
	defer uploader.mutex.Unlock()

	return uploader.calls, uploader.key, uploader.body
}

// traceFixture stores one running tracer test graph.
type traceFixture struct {
	// tracer is the tested packet tracer.
	tracer *Tracer
	// redis is the reusable Redis client.
	redis *redis.Client
	// uploader records finalized objects.
	uploader *traceUploader
	// bindings maps test traffic to players.
	bindings *binding.Registry
}

// newTraceFixture creates and starts one isolated trace graph.
func newTraceFixture(t *testing.T) traceFixture {
	t.Helper()
	server := miniredis.RunT(t)
	client := redis.New(redis.Config{Address: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	return newTraceFixtureWithStore(t, client, &traceUploader{})
}

// newTraceFixtureWithStore starts one trace graph over supplied persistence.
func newTraceFixtureWithStore(t *testing.T, store *redis.Client, uploader *traceUploader) traceFixture {
	t.Helper()
	bindings := binding.NewRegistry()
	tracer := &Tracer{
		store: store, uploader: uploader, players: playerlive.NewRegistry(),
		bindings: bindings, connections: netconn.NewRegistry(), log: zap.NewNop(),
		sessions: make(map[int64]*Session), writes: make(chan writeRequest, 32),
		stop: make(chan struct{}), now: func() time.Time { return time.Unix(100, 0).UTC() },
		uuid: func() string { return "trace-id" }, duration: DefaultDuration,
		maxEntries: DefaultMaxEntries, maxBytes: DefaultMaxBytes, reaperEvery: time.Hour,
	}
	if err := tracer.Start(context.Background()); err != nil {
		t.Fatalf("start tracer: %v", err)
	}
	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := tracer.Stop(stopCtx); err != nil {
			t.Fatalf("stop tracer: %v", err)
		}
	})

	return traceFixture{tracer: tracer, redis: store, uploader: uploader, bindings: bindings}
}

// bindTracePlayer maps one test player to a connection.
func bindTracePlayer(t *testing.T, bindings *binding.Registry, playerID int64, connectionID netconn.ID) {
	t.Helper()
	err := bindings.Add(binding.Binding{PlayerID: playerID, ConnectionID: connectionID, ConnectionKind: "websocket"})
	if err != nil {
		t.Fatalf("bind player: %v", err)
	}
}

// containsAll reports whether text contains every requested fragment.
func containsAll(text string, fragments ...string) bool {
	for _, fragment := range fragments {
		if !strings.Contains(text, fragment) {
			return false
		}
	}

	return true
}
