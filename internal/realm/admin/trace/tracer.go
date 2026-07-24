package trace

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	playerlive "github.com/niflaot/pixels/internal/realm/player/live"
	"github.com/niflaot/pixels/internal/realm/session/binding"
	netconn "github.com/niflaot/pixels/networking/connection"
	"github.com/niflaot/pixels/pkg/i18n"
	"github.com/niflaot/pixels/pkg/redis"
	"github.com/niflaot/pixels/pkg/storage"
	"go.uber.org/zap"
)

// activeKey stores player ids with durable trace sessions.
const activeKey = "pixels:admin:trace:active"

// Tracer owns active packet traces and their asynchronous persistence.
type Tracer struct {
	// store persists active sessions and ordered trace entries.
	store Store
	// uploader stores finalized trace documents.
	uploader Uploader
	// players resolves current player names.
	players *playerlive.Registry
	// bindings maps connection traffic to player ids.
	bindings *binding.Registry
	// connections sends completion feedback to connected players.
	connections *netconn.Registry
	// translations localizes player-facing trace feedback.
	translations i18n.Translator
	// log records trace lifecycle and persistence failures.
	log *zap.Logger
	// mutex protects active session state.
	mutex sync.RWMutex
	// sessions stores active traces by player id.
	sessions map[int64]*Session
	// writes serializes Redis entry persistence.
	writes chan writeRequest
	// stop requests background worker shutdown.
	stop chan struct{}
	// stopOnce closes stop exactly once.
	stopOnce sync.Once
	// workers waits for persistence and reaper loops.
	workers sync.WaitGroup
	// now supplies deterministic timestamps.
	now func() time.Time
	// uuid supplies deterministic object identifiers.
	uuid func() string
	// duration stores the fixed trace window.
	duration time.Duration
	// maxEntries stores the entry safety limit.
	maxEntries int
	// maxBytes stores the encoded byte safety limit.
	maxBytes int64
	// reaperEvery stores the expiration polling interval.
	reaperEvery time.Duration
}

// New creates a production packet tracer.
func New(store *redis.Client, uploader *storage.DebugClient, players *playerlive.Registry, bindings *binding.Registry, connections *netconn.Registry, translations i18n.Translator, log *zap.Logger) *Tracer {
	if log == nil {
		log = zap.NewNop()
	}

	return &Tracer{
		store: store, uploader: uploader, players: players, bindings: bindings,
		connections: connections, translations: translations, log: log,
		sessions: make(map[int64]*Session), writes: make(chan writeRequest, writeQueueSize),
		stop: make(chan struct{}), now: time.Now, uuid: uuid.NewString,
		duration: DefaultDuration, maxEntries: DefaultMaxEntries,
		maxBytes: DefaultMaxBytes, reaperEvery: reaperInterval,
	}
}

// Activate starts one fixed-window trace or returns the existing trace.
func (tracer *Tracer) Activate(ctx context.Context, playerID int64, playerName string) (Session, bool, error) {
	tracer.mutex.Lock()
	defer tracer.mutex.Unlock()
	if current := tracer.sessions[playerID]; current != nil {
		return *current, false, nil
	}

	startedAt := tracer.now().UTC()
	session := Session{PlayerID: playerID, PlayerName: playerName, StartedAt: startedAt, ExpiresAt: startedAt.Add(tracer.duration)}
	if err := tracer.persistSession(ctx, session); err != nil {
		return Session{}, false, err
	}
	if err := tracer.store.SetAdd(ctx, activeKey, playerKey(playerID)); err != nil {
		_ = tracer.store.Delete(ctx, metadataKey(playerID))
		return Session{}, false, err
	}
	tracer.sessions[playerID] = &session
	tracer.log.Info("packet trace activated", zap.Int64("player_id", playerID), zap.String("player_name", playerName), zap.Time("expires_at", session.ExpiresAt))

	return session, true, nil
}

// Active returns one active trace snapshot.
func (tracer *Tracer) Active(playerID int64) (Session, bool) {
	tracer.mutex.RLock()
	defer tracer.mutex.RUnlock()
	session := tracer.sessions[playerID]
	if session == nil {
		return Session{}, false
	}

	return *session, true
}

// persistSession writes current trace metadata with recovery retention.
func (tracer *Tracer) persistSession(ctx context.Context, session Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	return tracer.store.Set(ctx, metadataKey(session.PlayerID), data, redisRetention)
}
