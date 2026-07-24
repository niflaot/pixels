# Core Commands

Pixels ships four first-party room-chat commands for hotel staff. They use `PIXELS_COMMAND_PREFIX`, which defaults to `:`, and are consumed before the text can appear as normal room chat. Every command has its own dotted permission node and follows the normal player, group, inheritance, wildcard, and deny resolution rules.

| Command | Permission | Purpose |
|---|---|---|
| `:alert <player> <reason>` | `admin.alert` | Send a popup to one exact connected username |
| `:halert <reason>` | `admin.halert` | Send one popup to every currently connected player |
| `:about` | `admin.about` | Show the running Pixels version, commit, and loaded plugins |
| `:trace` | `admin.trace` | Toggle a bounded bidirectional packet trace for the issuing player |

The seeded `admin` group already inherits these nodes through its `*` grant. Production operators can grant or deny each node explicitly through the permission administration API.

## Direct and hotel alerts

`:alert` resolves usernames case-insensitively against the live player registry. It rejects the issuing account as its target and sends only the reason argument in the popup. An offline or disconnecting target produces explicit localized feedback instead of silently dropping the command. Successful sends are logged with issuer, target, and reason.

`:halert` reuses the same generic-alert packet, excludes the issuer, and continues through the remaining online snapshot if an individual connection fails. Its reply reports successful and failed deliveries, and the operation is logged with the issuer and reason.

## Build and plugin discovery

`:about` reads the build metadata compiled into the running binary and the immutable report produced by the native plugin loader. It exposes the semantic version, short commit, loaded plugin count, and loaded names. It does not query PostgreSQL or an external service.

## Production packet traces

`:trace` is a single-player toggle. The first invocation starts capture for the issuing account; the second finalizes it. Capture follows the player ID, so refreshing Nitro and receiving a new connection does not reset the trace deadline.

The trace observes both directions:

- `in`: packets successfully decoded from the client before realm dispatch;
- `out`: packets successfully written by the server transport.

Each line uses the same compact TOON escaping rules as protocol logs and contains sequence, timestamp, direction, compact connection ID, header, payload byte count, and Base64 payload. Capture has a fixed 30-minute deadline and never extends on activity or reconnection. It stops accepting entries after 20,000 packets or 20 MiB of encoded trace data, whichever happens first, and marks the final document as truncated.

Active metadata and ordered entries live in Redis. A controlled or unexpected process restart leaves that state intact; the next Pixels process reconciles and finalizes the interrupted trace with reason `server restarted`. Normal client disconnects do not finalize it.

Final documents are uploaded below `debug/traces/` in the S3-compatible bucket selected by `STORAGE_DEBUG_BUCKET`. The staff member receives the durable URL when still connected, and Pixels emits a warning log containing player, reason, count, truncation state, and URL. Upload failures retain the active state so expiry or another toggle can retry without losing the captured entries.

Packet payloads can contain private chat, room state, tickets, or other operational data visible to that session. Keep `admin.trace` tightly restricted, protect Redis and the S3 API, and apply an appropriate retention policy to `debug/traces/`. When `STORAGE_PUBLIC_READ=true`, anyone holding a trace URL can read it.

## Required infrastructure

`:trace` reuses the existing `REDIS_*` and shared `STORAGE_*` credentials documented in [[ENVIRONMENT-VARIABLES]], but routes objects through `STORAGE_DEBUG_BUCKET` and its optional `STORAGE_DEBUG_PUBLIC_BASE_URL`. Production startup validates both the camera and debug buckets independently.
