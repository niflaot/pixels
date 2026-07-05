# Handshake And Secure Connection Plan

This plan describes how the pixel-protocol Diffie-Hellman handshake should fit into the transport-agnostic `networking/connection` model. It is intentionally not an implementation plan for TCP, WebSocket, or any concrete transport. The goal is to define the base connection contract so a transport can later plug in packet IO, while protocol security, handler routing, and realm-owned commands remain separated.

## Current Protocol Shape

The packet catalog describes the Diffie flow as a crypto-phase handshake:

1. The client sends `HANDSHAKE_INIT_DIFFIE` inbound header `3110`.
   This packet has no payload. It asks the server for signed Diffie-Hellman parameters.
2. The server replies with `HANDSHAKE_INIT_DIFFIE` outbound header `1347`.
   The payload contains `encryptedPrime string` and `encryptedGenerator string`. These are RSA-signed Diffie-Hellman values.
3. The client sends `HANDSHAKE_COMPLETE_DIFFIE` inbound header `773`.
   The payload contains `encryptedPublicKey string`, which is the client-side Diffie-Hellman public key protected by RSA.
4. The server replies with `HANDSHAKE_COMPLETE_DIFFIE` outbound header `3885`.
   The payload contains `encryptedPublicKey string` and optionally `serverClientEncryption bool`.
5. After both sides compute the shared secret, RC4 encryption is installed for subsequent traffic according to the protocol docs.

The important architecture detail is that packets only define the wire shape. They should not own the handshake algorithm, session state, encryption switches, authentication rules, or realm commands.

## Current Package Responsibilities

`networking/codec` owns frame and payload encoding. It should remain unaware of connection state, authentication, encryption, and packet meaning.

`networking/inbound` owns typed client-to-server packet decoders. It should expose packet payloads such as `complete.Payload{EncryptedPublicKey: value}` and nothing more.

`networking/outbound` owns typed server-to-client packet encoders. It should expose packet builders such as `complete.Encode(serverPublicKey, complete.WithServerClientEncryption(true))` and nothing more.

`networking/connection` owns transport-agnostic session state, packet flow, security policy, handler registration, handler routing, and disconnection. It should not know whether bytes arrived from TCP, WebSocket, or another transport.

`networking/crypto` should own cryptographic algorithms and protocol security implementations. Diffie-Hellman, RSA adapters, RC4 setup, signing, shared-secret derivation, and encryption streams belong there, not in packet packages, handlers, transport adapters, or realm logic.

`internal/` should eventually own realm decisions, realm handlers, and realm commands. Each realm should register its handlers with the connection handler registry. The registry routes decoded plain packets to the correct handler; the handler owns how its realm command is created or executed.

## Desired Connection States

`Session` tracks ID, kind, start time, authentication time, receive/send, disconnect, handler registries, security policy, attached secure channel, and lifecycle state.

Proposed states:

| State | Meaning |
| --- | --- |
| `Created` | Session exists but no protocol packet has been processed. |
| `Handshaking` | Crypto negotiation or client metadata exchange is in progress. |
| `Securing` | Diffie parameters or public keys are being processed and ciphers are being prepared. |
| `Authenticating` | SSO ticket or account proof is being validated. |
| `Authenticated` | Authentication succeeded but session bootstrapping may still be incomplete. |
| `Connected` | Normal session traffic is allowed. |
| `Closing` | Disposal started and no new packet work should begin. |
| `Closed` | Disposal finished. |
| `Error` | A protocol, transport, security, or policy failure occurred. |

The first implementation can keep this state simple and strict. It does not need a large generic FSM package. A small state type, transition method, and transition table are enough.

## Proposed Transition Rules

The state machine should reject invalid packet flow early:

| From | Event | To |
| --- | --- | --- |
| `Created` | first inbound packet accepted | `Handshaking` |
| `Handshaking` | Diffie requested | `Securing` |
| `Securing` | Diffie completed | `Handshaking` or `Authenticating` |
| `Handshaking` | auth begins without required production encryption | `Error` |
| `Handshaking` | auth begins with development plain mode or completed encryption | `Authenticating` |
| `Authenticating` | auth succeeds | `Authenticated` |
| `Authenticated` | session bootstrap completes | `Connected` |
| any active state | local or remote close | `Closing` |
| `Closing` | disposer returns | `Closed` |
| any active state | protocol/security failure | `Error` |
| `Error` | disconnect starts | `Closing` |

The exact events should be named after connection-level meaning, not packet names. Examples:

- `PacketReceived`
- `DiffieRequested`
- `DiffieParametersSent`
- `DiffiePublicKeyReceived`
- `SecureChannelInstalled`
- `AuthenticationStarted`
- `AuthenticationAccepted`
- `AuthenticationRejected`
- `SessionReady`
- `DisconnectRequested`
- `TransportFailed`

Packet handlers should not know whether security is enabled. They receive a plain packet after the session has already applied security. Handlers can create or execute realm-owned commands, but connection state mutation should still happen through the session contract.

## Secure Connection Abstraction

The connection package should eventually expose a small security abstraction that can wrap packet bytes without binding to a concrete algorithm. The interface belongs near the session because the session owns state and policy. The concrete Diffie implementation belongs in `networking/crypto`.

Possible contract:

This is implemented as `SecureChannel` in `networking/connection`.

This exact API is not final. The intent matters more than these names:

- The session can ask security to transform inbound bytes before frame decoding, once encryption is installed.
- The session can ask security to transform outbound bytes after frame encoding, once encryption is installed.
- The session can let security react to handshake packets and produce connection-level results.
- The security module can expose whether traffic is plain, negotiating, ready, or failed.
- The transport never needs to know which algorithm is active.
- Handlers never need to know whether a packet came from plain bytes or decrypted bytes.

## Diffie Implementation Boundary

The Diffie-specific implementation must not live directly in `Session`. The concrete implementation should live in `networking/crypto`, most likely `networking/crypto/diffie` if the package grows beyond a single file.

The Diffie implementation should own:

- Server private Diffie-Hellman value.
- Server public Diffie-Hellman value.
- Prime and generator source.
- RSA signing/encryption adapter.
- Shared secret derivation.
- RC4 cipher construction.
- Security flags such as whether server-to-client encryption is enabled.

The `networking/connection` package should only own:

- Current connection state.
- Current security state.
- Security policy, including whether secure traffic is required before authentication.
- Whether inbound and outbound bytes must pass through `Open` or `Seal`.
- Which connection-level results were produced when security progresses or fails.

The production policy is mandatory encryption. If the server is running in production and no encryption module is attached, the session must reject the handshake or authentication flow with `DisconnectProtocolError`.

The development policy is optional encryption. A development session can remain in `SecurityPlain` and still authenticate, but it should still support the Diffie flow when the client requests it.

## Packet Handler Flow

Handshake packet handlers should stay thin and must receive plain, already-unwrapped packets. They must never know whether the session used encrypted bytes, plain bytes, TCP, WebSocket, or anything else.

The routing and command boundary is:

1. Transport receives bytes.
2. Session applies `SecureChannel.Open` when security is ready.
3. Session decodes frames into plain `codec.Packet` values.
4. Session dispatches the plain packet to the handler registry.
5. Handler registry routes the packet by header to the registered realm handler.
6. Realm handler decodes the packet-specific payload.
7. Realm handler creates or executes its realm-owned command.

The handler sees only the final unencrypted packet or typed payload. The command belongs to that realm and carries decoded meaning forward. There should not be a generic networking command executor deciding realm behavior.

For inbound `HANDSHAKE_INIT_DIFFIE`:

1. Decode the packet with `networking/inbound/handshake/diffie/init`.
2. Create or execute a realm-owned command like `handshake.diffie.requested`.
3. Do not branch on whether the connection is secure or plain.
4. Do not generate prime, generator, RSA signatures, or outbound packets in the packet decoder.

For inbound `HANDSHAKE_COMPLETE_DIFFIE`:

1. Decode the packet with `networking/inbound/handshake/diffie/complete`.
2. Create or execute a realm-owned command containing the encrypted client public key.
3. Do not branch on whether the connection is secure or plain.
4. Do not derive shared secrets in the packet decoder.

For outbound `HANDSHAKE_INIT_DIFFIE` and `HANDSHAKE_COMPLETE_DIFFIE`:

1. The realm handler command requests prepared Diffie values from `networking/crypto` through a realm dependency.
2. The outbound packet package encodes the typed values.
3. `Session.Send` delegates packet writing to the session's transport callback.
4. The byte writer applies `SecureChannel.Seal` when security is ready.

This keeps packet packages as wire definitions, the handler registry as a router, realm handlers as command owners, and security as a session capability.

## Where Encryption Should Apply

Encryption should apply at the session byte boundary around frame encoding and decoding. The protocol docs describe subsequent traffic as encrypted after Diffie completes, so the session should make handlers see only plain packets:

- Inbound transport bytes arrive.
- Session checks the connection security policy and security state.
- If security is ready, `SecureChannel.Open` decrypts bytes.
- If security is required but unavailable, the session disconnects with `DisconnectProtocolError`.
- `codec.DecodeFrames` parses decrypted bytes into packets.
- Inbound packet handlers receive plain packets.
- Realm handlers own commands from decoded packet meaning.

Outbound:

- Outbound packet is encoded with `codec.AppendFrame`.
- Session checks the connection security policy and security state.
- If security is ready, `SecureChannel.Seal` encrypts bytes.
- If security is required but unavailable, the session disconnects with `DisconnectProtocolError`.
- Transport writes encrypted bytes.

The current `Session.Send` operates on `codec.Packet`, not raw bytes. A future session byte adapter should own this byte loop without becoming a TCP or WebSocket transport. Concrete transports still only read and write bytes.

## Authentication Rules

Authentication and security should be separate but coordinated.

Proposed rules:

- `SECURITY_TICKET` should only be accepted in `Handshaking` or `Authenticating`.
- In production, `SECURITY_TICKET` must be rejected unless the security state is `SecurityReady`.
- In development, `SECURITY_TICKET` can be accepted when security is `SecurityPlain` or `SecurityReady`.
- A failed Diffie exchange should move the connection to `Error` and disconnect with `DisconnectProtocolError` or `DisconnectAuthenticationFailed`, depending on the failure.
- A missing encryption module when encryption is required must disconnect with `DisconnectProtocolError`.
- Handler code must not branch on encrypted versus unencrypted traffic. Session state and security policy own that decision entirely.

The connection model should expose connection identity and lifecycle state to handlers when needed, but it should not expose security internals. Final authentication acceptance belongs to the realm handler and its realm-owned command flow, not to transport code.

## Implemented Connection Base

The base `networking/connection` contract now includes:

```go
type State uint8

type Transition struct {
	From State
	Event Event
	To State
}

type Event string

type SecurityMode uint8

type SecurityPolicy struct {
	Mode SecurityMode
}
```

`SecurityMode` should have at least:

- `SecurityOptional`, used by development by default.
- `SecurityRequired`, used by production by default.

`Session` exposes:

- `State() State`
- `SecurityState() SecurityState`
- `Transition(event Event) error`
- `AttachSecurity(channel SecureChannel) error`
- `SecurityPolicy() SecurityPolicy`
- `SetSecurityPolicy(policy SecurityPolicy) error`, only during construction or before traffic starts.
- `ValidateAuthenticationSecurity(ctx) error`, used by authentication commands before accepting a ticket.
- `Open(bytes)` and `Seal(bytes)`, used by byte adapters around codec framing.

The implementation should avoid a broad interface unless multiple session implementations truly need it. A concrete `Session` with a small exported contract is still the Go-way default.

## Command Naming

Commands owned by realm handlers should use stable names. They are the realm handoff from decoded packet meaning into realm, security, or authentication behavior. A first pass could use:

- `handshake.diffie.requested`
- `handshake.diffie.client_public_key_received`
- `handshake.diffie.parameters_ready`
- `handshake.diffie.completed`
- `handshake.diffie.failed`
- `authentication.ticket.received`
- `authentication.accepted`
- `authentication.rejected`
- `session.ready`
- `session.disconnect.requested`

Command payloads should be typed structs where possible, not generic maps. Realm command constructors should keep payload usage disciplined.

Commands should never contain encrypted bytes. By the time a command exists, the session has already applied security and the handler has decoded meaningful packet data.

## Incremental Implementation Steps

1. Add realm-owned handshake handlers that register with `HandlerRegistry`.
2. Add realm command types for Diffie init, Diffie completion, authentication, and session readiness.
3. Use `networking/crypto/diffie.Provider` from realm commands to prepare Diffie values.
4. Implement Diffie as a concrete secure channel in `networking/crypto`.
5. Add byte-level session adapter tests with fake transport bytes only.
6. Integrate real TCP/WebSocket transports later, after the base connection contract is stable.

## Non-Goals For The Next Step

- Do not implement TCP.
- Do not implement WebSocket.
- Do not expose SDK hooks for security.
- Do not put authentication inside packet definitions or transport adapters.
- Do not put Diffie math directly inside `Session`.
- Do not make a generic FSM framework.
- Do not hide packet flow behind a large interface before we have multiple real implementations.

## References

- Pixel Protocol packet catalog: https://niflaot.github.io/pixel-protocol/docs/protocol/packet-catalog/
- Client Diffie init: https://niflaot.github.io/pixel-protocol/docs/protocol/handshake-security/c2s/handshake-init-diffie/
- Server Diffie init: https://niflaot.github.io/pixel-protocol/docs/protocol/handshake-security/s2c/handshake-init-diffie/
- Client Diffie complete: https://niflaot.github.io/pixel-protocol/docs/protocol/handshake-security/c2s/handshake-complete-diffie/
- Server Diffie complete: https://niflaot.github.io/pixel-protocol/docs/protocol/handshake-security/s2c/handshake-complete-diffie/
