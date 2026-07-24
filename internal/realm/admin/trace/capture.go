package trace

import (
	"encoding/base64"

	"github.com/niflaot/pixels/internal/realm/session/binding"
	"github.com/niflaot/pixels/networking/codec"
	netconn "github.com/niflaot/pixels/networking/connection"
	"github.com/niflaot/pixels/pkg/logger"
	"go.uber.org/zap"
)

// Observe captures successful traffic for an authenticated traced player.
func (tracer *Tracer) Observe(connection netconn.Context, packet codec.Packet) {
	current, found := tracer.bindings.FindByConnection(bindingKey(connection))
	if !found {
		return
	}

	tracer.capture(current.PlayerID, connection, packet)
}

// capture enqueues one bounded packet trace entry.
func (tracer *Tracer) capture(playerID int64, connection netconn.Context, packet codec.Packet) {
	tracer.mutex.Lock()
	defer tracer.mutex.Unlock()
	session := tracer.sessions[playerID]
	if session == nil || session.finalizing || session.Truncated {
		return
	}
	now := tracer.now()
	if !now.Before(session.ExpiresAt) {
		return
	}

	line := []byte(logger.FormatToonLine(map[string]any{
		"seq":     session.Count + 1,
		"ts":      now.UTC().Format(traceTimestamp),
		"dir":     directionName(connection.Direction),
		"cid":     shortConnectionID(connection.ConnectionID),
		"header":  packet.Header,
		"bytes":   len(packet.Payload),
		"payload": base64.StdEncoding.EncodeToString(packet.Payload),
	}))
	if session.Count >= tracer.maxEntries || session.Bytes+int64(len(line)) > tracer.maxBytes {
		session.Truncated = true
		tracer.enqueueMetadata(*session)
		return
	}

	next := *session
	next.Count++
	next.Bytes += int64(len(line))
	request := writeRequest{session: next, line: line}
	select {
	case tracer.writes <- request:
		session.Count = next.Count
		session.Bytes = next.Bytes
	default:
		session.Truncated = true
		tracer.log.Warn("packet trace persistence queue full", zap.Int64("player_id", playerID))
	}
}

// enqueueMetadata schedules a best-effort metadata update.
func (tracer *Tracer) enqueueMetadata(session Session) {
	select {
	case tracer.writes <- writeRequest{session: session}:
	default:
	}
}

// bindingKey converts traffic context to a session binding key.
func bindingKey(connection netconn.Context) binding.ConnectionKey {
	return binding.ConnectionKey{Kind: connection.ConnectionKind, ID: connection.ConnectionID}
}

// directionName returns the compact trace direction label.
func directionName(direction netconn.Direction) string {
	if direction == netconn.OutboundDirection {
		return "out"
	}

	return "in"
}

// shortConnectionID returns a compact connection identifier.
func shortConnectionID(id netconn.ID) string {
	value := string(id)
	if len(value) <= 8 {
		return value
	}

	return value[:8]
}
