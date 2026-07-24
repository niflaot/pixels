package trace

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	outalert "github.com/niflaot/pixels/networking/outbound/session/alert"
	"github.com/niflaot/pixels/pkg/i18n"
	"go.uber.org/zap"
)

// finalizeLockTTL bounds cross-process trace upload ownership.
const finalizeLockTTL = time.Minute

// Finalize uploads and removes one active trace exactly once per process.
func (tracer *Tracer) Finalize(ctx context.Context, playerID int64, reason string) (Result, error) {
	tracer.mutex.Lock()
	session := tracer.sessions[playerID]
	if session == nil {
		tracer.mutex.Unlock()
		return Result{}, ErrInactive
	}
	if session.finalizing {
		tracer.mutex.Unlock()
		return Result{}, ErrFinalizing
	}
	session.finalizing = true
	snapshot := *session
	tracer.mutex.Unlock()

	if err := tracer.flush(ctx, snapshot); err != nil {
		tracer.resetFinalizing(playerID)
		return Result{}, err
	}
	result, err := tracer.finalizeStored(ctx, snapshot, reason)
	if err != nil {
		tracer.resetFinalizing(playerID)
		return Result{}, err
	}
	tracer.mutex.Lock()
	delete(tracer.sessions, playerID)
	tracer.mutex.Unlock()

	return result, nil
}

// finalizeStored uploads one already-persisted session under a distributed lock.
func (tracer *Tracer) finalizeStored(ctx context.Context, session Session, reason string) (Result, error) {
	locked, err := tracer.store.SetIfAbsent(ctx, finalizeKey(session.PlayerID), []byte("1"), finalizeLockTTL)
	if err != nil {
		return Result{}, err
	}
	if !locked {
		return Result{}, ErrFinalizing
	}
	defer func() { _ = tracer.store.Delete(context.Background(), finalizeKey(session.PlayerID)) }()

	entries, err := tracer.store.ListRange(ctx, entriesKey(session.PlayerID), 0, -1)
	if err != nil {
		return Result{}, err
	}
	finishedAt := tracer.now().UTC()
	body := renderFile(session, finishedAt, reason, entries)
	key := "debug/traces/" + tracer.uuid() + ".txt"
	url, err := tracer.uploader.Put(ctx, key, bytes.NewReader(body), int64(len(body)), "text/plain; charset=utf-8")
	if err != nil {
		return Result{}, err
	}
	result := Result{URL: url, Count: len(entries), Truncated: session.Truncated}
	if err = tracer.cleanup(ctx, session.PlayerID); err != nil {
		tracer.log.Error("packet trace cleanup failed", zap.Int64("player_id", session.PlayerID), zap.Error(err))
	}
	tracer.log.Warn(
		"packet trace finalized",
		zap.Int64("player_id", session.PlayerID),
		zap.String("player_name", session.PlayerName),
		zap.String("reason", reason),
		zap.Int("packet_count", result.Count),
		zap.Bool("truncated", result.Truncated),
		zap.String("link", result.URL),
	)

	return result, nil
}

// cleanup removes every durable key for one completed trace.
func (tracer *Tracer) cleanup(ctx context.Context, playerID int64) error {
	activeErr := tracer.store.SetRemove(ctx, activeKey, playerKey(playerID))
	metadataErr := tracer.store.Delete(ctx, metadataKey(playerID))
	entriesErr := tracer.store.Delete(ctx, entriesKey(playerID))

	return errors.Join(activeErr, metadataErr, entriesErr)
}

// resetFinalizing makes a failed finalization retryable.
func (tracer *Tracer) resetFinalizing(playerID int64) {
	tracer.mutex.Lock()
	if session := tracer.sessions[playerID]; session != nil {
		session.finalizing = false
	}
	tracer.mutex.Unlock()
}

// Notify sends finalized trace feedback when the player remains connected.
func (tracer *Tracer) Notify(ctx context.Context, playerID int64, result Result) {
	message := tracer.message("admin.command.trace.saved", "Trace guardado: {url}", i18n.Params{"url": result.URL})
	packet, err := outalert.Encode(message)
	if err != nil {
		return
	}
	current, found := tracer.bindings.FindByPlayer(playerID)
	if !found {
		return
	}
	connection, found := tracer.connections.Get(current.ConnectionKind, current.ConnectionID)
	if !found {
		return
	}

	_ = connection.Send(ctx, packet)
}

// message resolves one trace translation with a fallback.
func (tracer *Tracer) message(key string, fallback string, params ...i18n.Params) string {
	if tracer.translations == nil {
		return replaceParams(fallback, params...)
	}
	value := tracer.translations.Default(i18n.Key(key), params...)
	if value == key {
		return replaceParams(fallback, params...)
	}

	return value
}

// replaceParams applies translation parameters to fallback text.
func replaceParams(value string, params ...i18n.Params) string {
	for _, replacements := range params {
		for key, replacement := range replacements {
			value = strings.ReplaceAll(value, "{"+key+"}", replacement)
		}
	}

	return value
}

// playerKey returns the stable Redis player member.
func playerKey(playerID int64) string { return strconv.FormatInt(playerID, 10) }

// metadataKey returns one trace metadata key.
func metadataKey(playerID int64) string {
	return fmt.Sprintf("pixels:admin:trace:%d:session", playerID)
}

// entriesKey returns one trace entry-list key.
func entriesKey(playerID int64) string { return fmt.Sprintf("pixels:admin:trace:%d:entries", playerID) }

// finalizeKey returns one trace finalization lock key.
func finalizeKey(playerID int64) string {
	return fmt.Sprintf("pixels:admin:trace:%d:finalize", playerID)
}
