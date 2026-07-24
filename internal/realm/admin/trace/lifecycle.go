package trace

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	realmconn "github.com/niflaot/pixels/internal/realm/connection"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Register installs bidirectional packet observation and lifecycle workers.
func Register(lifecycle fx.Lifecycle, handlers *realmconn.Handlers, tracer *Tracer) error {
	if err := handlers.Observers.Register(tracer); err != nil {
		return err
	}
	lifecycle.Append(fx.Hook{OnStart: tracer.Start, OnStop: tracer.Stop})

	return nil
}

// Start reconciles interrupted traces and starts persistence workers.
func (tracer *Tracer) Start(ctx context.Context) error {
	tracer.workers.Add(2)
	go tracer.writeLoop()
	go tracer.reaperLoop()

	return tracer.reconcile(ctx)
}

// Stop drains queued persistence work and stops background workers.
func (tracer *Tracer) Stop(ctx context.Context) error {
	tracer.stopOnce.Do(func() { close(tracer.stop) })
	done := make(chan struct{})
	go func() {
		tracer.workers.Wait()
		close(done)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

// writeLoop serializes packet entries and metadata into Redis.
func (tracer *Tracer) writeLoop() {
	defer tracer.workers.Done()
	for {
		select {
		case request := <-tracer.writes:
			tracer.persistWrite(request)
		case <-tracer.stop:
			tracer.drainWrites()
			return
		}
	}
}

// persistWrite stores one ordered trace write and resolves optional barriers.
func (tracer *Tracer) persistWrite(request writeRequest) {
	ctx := context.Background()
	var err error
	if len(request.line) > 0 {
		err = tracer.store.ListAppend(ctx, entriesKey(request.session.PlayerID), request.line)
		if err == nil {
			err = tracer.store.Expire(ctx, entriesKey(request.session.PlayerID), redisRetention)
		}
	}
	if err == nil {
		err = tracer.persistSession(ctx, request.session)
	}
	if err != nil {
		tracer.log.Error("packet trace persistence failed", zap.Int64("player_id", request.session.PlayerID), zap.Error(err))
	}
	if request.done != nil {
		request.done <- err
	}
}

// drainWrites persists every queued write before shutdown.
func (tracer *Tracer) drainWrites() {
	for {
		select {
		case request := <-tracer.writes:
			tracer.persistWrite(request)
		default:
			return
		}
	}
}

// flush waits until prior packet writes and current metadata are durable.
func (tracer *Tracer) flush(ctx context.Context, session Session) error {
	done := make(chan error, 1)
	select {
	case tracer.writes <- writeRequest{session: session, done: done}:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// reaperLoop finalizes sessions after their absolute deadline.
func (tracer *Tracer) reaperLoop() {
	defer tracer.workers.Done()
	ticker := time.NewTicker(tracer.reaperEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			tracer.reap(context.Background())
		case <-tracer.stop:
			return
		}
	}
}

// reap finalizes every expired in-memory trace.
func (tracer *Tracer) reap(ctx context.Context) {
	now := tracer.now()
	tracer.mutex.RLock()
	expired := make([]int64, 0)
	for playerID, session := range tracer.sessions {
		if !session.finalizing && !now.Before(session.ExpiresAt) {
			expired = append(expired, playerID)
		}
	}
	tracer.mutex.RUnlock()
	for _, playerID := range expired {
		result, err := tracer.Finalize(ctx, playerID, "expired")
		if err == nil {
			tracer.Notify(ctx, playerID, result)
		}
	}
}

// reconcile finalizes traces left active by a prior process.
func (tracer *Tracer) reconcile(ctx context.Context) error {
	members, err := tracer.store.SetMembers(ctx, activeKey)
	if err != nil {
		return err
	}
	for _, member := range members {
		playerID, parseErr := strconv.ParseInt(member, 10, 64)
		if parseErr != nil {
			_ = tracer.store.SetRemove(ctx, activeKey, member)
			continue
		}
		data, found, findErr := tracer.store.Find(ctx, metadataKey(playerID))
		if findErr != nil {
			return findErr
		}
		if !found {
			_ = tracer.store.SetRemove(ctx, activeKey, member)
			continue
		}
		var session Session
		if unmarshalErr := json.Unmarshal(data, &session); unmarshalErr != nil {
			tracer.log.Error("packet trace metadata invalid", zap.Int64("player_id", playerID), zap.Error(unmarshalErr))
			continue
		}
		result, finalizeErr := tracer.finalizeStored(ctx, session, "server restarted")
		if finalizeErr == nil {
			tracer.Notify(ctx, playerID, result)
			continue
		}
		session.ExpiresAt = tracer.now()
		tracer.mutex.Lock()
		tracer.sessions[playerID] = &session
		tracer.mutex.Unlock()
		tracer.log.Error("packet trace reconciliation failed", zap.Int64("player_id", playerID), zap.Error(finalizeErr))
	}

	return nil
}
