package trace

import (
	"bytes"
	"fmt"
	"time"
)

// traceTimestamp is the millisecond-precision UTC trace layout.
const traceTimestamp = "2006-01-02T15:04:05.000Z07:00"

// renderFile builds one human-readable TOON packet trace document.
func renderFile(session Session, finishedAt time.Time, reason string, entries [][]byte) []byte {
	var output bytes.Buffer
	_, _ = fmt.Fprintf(&output, "Pixels packet trace\n")
	_, _ = fmt.Fprintf(&output, "player: %s (%d)\n", session.PlayerName, session.PlayerID)
	_, _ = fmt.Fprintf(&output, "started_at: %s\n", session.StartedAt.UTC().Format(traceTimestamp))
	_, _ = fmt.Fprintf(&output, "finished_at: %s\n", finishedAt.UTC().Format(traceTimestamp))
	_, _ = fmt.Fprintf(&output, "duration: %s\n", finishedAt.Sub(session.StartedAt).Round(time.Millisecond))
	_, _ = fmt.Fprintf(&output, "reason: %s\n", reason)
	_, _ = fmt.Fprintf(&output, "packet_count: %d\n", len(entries))
	_, _ = fmt.Fprintf(&output, "truncated: %t\n\n", session.Truncated)
	for _, entry := range entries {
		output.Write(entry)
		if len(entry) == 0 || entry[len(entry)-1] != '\n' {
			output.WriteByte('\n')
		}
	}
	if session.Truncated {
		output.WriteString("-- truncado: se alcanzó el límite de captura --\n")
	}

	return output.Bytes()
}
