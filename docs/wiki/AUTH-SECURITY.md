# Security and Encryption

This page covers the secure channel wrapper, the optional legacy Diffie-Hellman and RC4 compatibility layer, its explicit policy, and the configuration values you must protect. Read [[AUTH-CONNECTIONS]] for the transport boundary and [[AUTH-HANDSHAKE]] for the states these rules apply in.

## The secure channel wrapper

Security is a session capability rather than a special connection type. `SecureChannel` owns five operations:

```go
type SecureChannel interface {
	State() SecurityState
	Begin(context.Context) error
	Open([]byte) ([]byte, error)
	Seal([]byte) ([]byte, error)
	Close(Reason) error
}
```

When no ready channel is attached, `Session.Open` and `Session.Seal` return the original bytes. Once a channel reports `SecurityReady`, inbound transport bytes pass through `Open` before Pixel frame decoding, and complete outbound frames pass through `Seal` immediately before transport writing. Realm handlers always receive the same decoded packets.

Security activation needs an ordering barrier. `CompleteSecurity` first sends the plaintext completion packet, then asks the transport's `SecurityActivator` to queue channel activation. The WebSocket adapter uses its single writer queue for both operations, so encryption cannot begin before the completion packet is physically written.

## Legacy Diffie-Hellman compatibility

Pixels implements the historical in-band handshake used by Nitro-family clients:

```text
Nitro                         Pixels
  HANDSHAKE_INIT_DIFFIE  ────▶
                         ◀──── RSA-signed prime and generator
  RSA-encrypted public key ──▶
                         ◀──── RSA-signed server public key
  RC4 client traffic      ═══▶
  RC4 server traffic      ◀═══  when enabled
```

Every connection receives fresh Diffie values generated with `crypto/rand`. The server validates the client public value, derives the unsigned big-endian shared key, and creates independent stateful RC4 streams for each direction. RSA uses the protocol's PKCS#1 v1.5 block formats: type 1 for server signatures and type 2 for client public-key encryption.

The final server public-key packet is deliberately sent in plaintext. The WebSocket writer queues an activation barrier immediately after it, and only later packets use RC4. This ordering is required for interoperability and is covered by session, transport, crypto, and handler tests.

When compatibility is disabled, clients may skip Diffie and send their SSO ticket normally. A client that explicitly requests Diffie while the layer is disabled is disconnected instead of receiving a partial exchange.

## The security policy

Legacy protocol encryption is a compatibility choice, not an environment identity. `PIXELS_ENV=production` no longer implies Diffie and never blocks an otherwise valid SSO ticket. `PIXELS_DIFFIE_REQUIRED=true` is the only setting that requires a ready in-protocol channel, and startup rejects that setting unless `PIXELS_DIFFIE_ENABLED=true`.

| Mode | Authentication rule |
|---|---|
| `SecurityOptional` | A plaintext session may present an SSO ticket |
| `SecurityRequired` | The session must have a `SecurityReady` channel before the SSO ticket is accepted |

Handlers never see any of this. The session unwraps security before dispatch, so packet handlers are byte for byte identical on plain and secured connections. That separation is a hard architectural rule.

TLS termination in front of the server remains required for public deployments. RC4 and 128-bit legacy Diffie exist only for client compatibility; they are obsolete cryptography and are not a substitute for HTTPS/WSS.

## Compatibility configuration

| Variable | Default | Purpose |
|---|---|---|
| `PIXELS_DIFFIE_ENABLED` | `false` | Accept the legacy handshake |
| `PIXELS_DIFFIE_REQUIRED` | `false` | Require it before SSO authentication |
| `PIXELS_DIFFIE_RSA_EXPONENT` | `3` | Hexadecimal public RSA exponent |
| `PIXELS_DIFFIE_RSA_MODULUS` | empty | Hexadecimal modulus shared with Nitro |
| `PIXELS_DIFFIE_RSA_PRIVATE_EXPONENT` | empty | Server-only hexadecimal private exponent |
| `PIXELS_DIFFIE_PRIME_BITS` | `128` | Per-session legacy DH prime size |
| `PIXELS_DIFFIE_PRIVATE_BITS` | `128` | Per-session legacy DH private value size |
| `PIXELS_DIFFIE_SERVER_CLIENT_ENCRYPTION` | `true` | Encrypt outbound traffic as well as inbound |

The modulus and exponent must match Nitro Renderer's `security.diffie.rsa.modulus` and `security.diffie.rsa.exponent`. Nitro never receives the private exponent. Do not use the public example key pairs copied through old emulator repositories; their private values are already public.

Generate a fresh matching server and renderer configuration locally:

```sh
go run ./cmd/diffie-keygen > diffie-keys.txt
chmod 600 diffie-keys.txt
```

Copy the three `PIXELS_DIFFIE_RSA_*` values into the server's secret environment and only the two `security.diffie.rsa.*` public values into `renderer-config.json`. Delete the temporary file after storing the private exponent in the deployment secret manager.

## Configuration you must not leave at defaults

Everything boots with defaults for development convenience, and three of those defaults are published in this repository, which makes them exactly as secret as a sticky note on the monitor:

| Variable | Default | Why you must change it |
|---|---|---|
| `PIXELS_ACCESS_KEY` | `pixels-development-access-key-change-me` | Guards every private HTTP route, **including SSO ticket creation**. Anyone holding it can mint a login ticket for any player. |
| `SSO_KEY` | `pixels-development-sso-key-change-me` | HMAC key deriving the Redis storage keys for tickets. |
| `PIXELS_ENV` | `development` | Exposes development-only behavior such as `/docs`. Set `production`. |
| `PIXELS_DIFFIE_RSA_PRIVATE_EXPONENT` | empty | Required only when compatibility is enabled and must remain server-side. |

Related knobs with sane defaults you may still want to tune: `SSO_DEFAULT_TTL` (five minutes; tickets are consumed within seconds in a healthy flow, so shorter is fine), `SSO_PREFIX` (Redis namespacing when sharing an instance), and the `PIXELS_WS_*` family that governs the WebSocket layer everything above rides on.
