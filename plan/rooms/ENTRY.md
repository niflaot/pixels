# Plan: Entrada a Rooms (`internal/realm/room` — profundización de puerta)

Este plan profundiza específicamente la **Parte 2** de `plan/REMAINING-ROOMS.md` (entrada a rooms cerrados: timbre, contraseña, invisible), que ahí quedó a nivel de diseño general. Acá se detalla la máquina de estados completa, un sistema de **hangout** (timeout de cola de timbre, ~5 minutos) que no tiene equivalente en Arcturus, un sistema de **máximo de intentos + freeze + alertas** para contraseña (tampoco existe en Arcturus), y un diseño completo de **forwarding** — cómo se comporta la entrada a un room cuando el disparador no es un jugador tocando la puerta normal, sino un teleport forzado (endpoint admin, furniture teleportador, o una futura feature de usuario). Incluye modelo/esquema, nodos de permiso, hot paths/allocations, testing, y milestones.

**Estado real (actualizado)**: la mayor parte de este plan ya se implementó tal cual — `internal/realm/room/entry`, `internal/realm/room/doorbell`, y `pkg/http/room/routes/teleport.go` ya existen en el código, incluyendo el bypass `Trusted`/`GrantTrusted` y el endpoint admin single-player de la Parte 4. Los dos puntos que este documento dejó deliberadamente en stub — "quién tiene derechos" y "quién está baneado" — se implementan completamente en **`plan/rooms/RIGHTS.md`**, que se engancha a los dos puntos de extensión reales que `internal/realm/room/entry/service.go` ya expone (`entry.RightsChecker`, `entry.BanChecker`, `Service.WithRights`/`Service.WithBans`), sin necesitar ningún cambio adicional acá.

Es un plan solamente — no se escribió código Go todavía.

---

## Parte 0 — Punto de partida real (grounding, confirmado leyendo el código actual)

Antes de diseñar nada nuevo, esto es lo que **ya existe** en Pixels hoy, confirmado leyendo el código (no supuesto):

| Ya existe | Dónde | Nota |
| --- | --- | --- |
| `DoorMode` enum (`Open=0/Doorbell=1/Password=2/Invisible=3`) | `internal/realm/room/model/room.go` | Completo, ya usado por `rooms_door_mode_chk` en la migración. |
| Columna `password_hash text null` y `door_mode smallint` | `internal/realm/room/database/migrations/0002_create_room_records.sql` | La columna existe en Postgres **pero `model.Room` y `repository/scan.go` no la leen/exponen todavía** — gap real, se cierra en Parte 5.1. |
| Inbound `RoomEnter` ya incluye `Password string` | `networking/inbound/room/enter` (usado por `handlers/enter/handler.go`) | El protocolo ya soporta mandar la contraseña en el mismo packet de entrada — no hace falta un packet de "reintentar" aparte, igual que en Arcturus. |
| Outbound `ROOM_ENTER_ERROR` (899) | `networking/outbound/room/entryerror` | Hoy solo se usa `ErrorRoomFull = 1`. Es un simple `int32`, extensible sin tocar el shape del packet. |
| Outbound `ROOM_FORWARD` (160) | `networking/outbound/room/forward` | Ya existe y ya se usa por un endpoint admin bulk (`POST /api/admin/rooms/:id/forward`, `pkg/http/room/routes`) que reubica a TODOS los ocupantes activos de un room. |
| Inbound `FORWARD_TO_SOME_ROOM` (1703) | `networking/inbound/navigator/forward` | Header reservado, pero `Payload{}`/`Definition{}` están **vacíos** — no decodifica ningún campo, y no hay ningún handler registrado para este header. Gap real, se cierra en Parte 4.4. |
| `pkg/redis.Client` | `pkg/redis/client.go` | Expone `Find`/`Set`/`Delete`/`Expire`/`Take` (`GetDel`) — **no tiene ninguna operación atómica de incremento**. Gap real, se cierra en Parte 3.3. |
| Alertas de sesión ya wireadas de punta a punta | `networking/outbound/session/{alert,bubblealert}` + `pkg/http/notification/routes` + `pkg/i18n` | `GENERIC_ALERT` (3801) y `BUBBLE_ALERT` ya existen, ya se mandan hoy vía un endpoint HTTP admin de notificación, y ya usan `pkg/i18n.Translator` para localizar el mensaje. Reusable directo desde un command handler de juego (mismo `netconn.Context.Send`). |
| `internal/permission` — **implementado, no solo planeado** | `internal/permission/{node.go,registry.go,service,...}` | `permission.Node`, `permission.RegisterNode`, `permission.Checker.HasPermission(ctx, playerID, node)` ya son código real. `catalog`, `currency`, y `player` ya tienen su propio `permissions.go` (`internal/realm/<realm>/permissions.go`) — **`room` todavía no tiene el suyo**. Este plan es el primero en necesitarlo (Parte 5.2). |
| Tick loop por room activo, 500ms | `internal/realm/room/live/loop.go`, `DefaultTickInterval = 500 * time.Millisecond` (`live/model.go`) | Ya corre una goroutine por room activo dedicada a mover unidades (`Room.Tick()`, `live/movement.go`). Este plan reusa el mismo ticker para el timeout de hangout (Parte 2.3), sin crear ninguna goroutine nueva. |
| `roomlive.Room.CanManageFurniture` degrada "derechos" a "es el dueño" | `internal/realm/room/live/model.go` | Ya existe un precedente exacto para el criterio interino que este plan usa en 1.5/2.2 mientras `room_rights` (R2 de `REMAINING-ROOMS.md`) no exista: "tiene derechos" ≈ "es el dueño". No es una convención nueva. |
| Teleportadores de furniture ya diseñados | `plan/INTERACTIONS.md`, Parte 8 / Milestone I6 | Ya deja anotado que el cruce entre rooms "reusa el mismo camino que ya usa `room.enter`". Este plan (Parte 4.6) precisa exactamente cómo. |
| No existe ningún hashing de contraseñas en el proyecto | grep sin resultados para `bcrypt` en todo el repo | El login de jugador es vía SSO externo (`internal/auth/sso`), no hay contraseña de cuenta que hashear en Pixels. La contraseña de **room** es un concepto nuevo y necesita su propia dependencia de hashing (Parte 1.3). |

---

## Parte 1 — Máquina de estados de entrada

### 1.1 Pipeline completo (reemplaza/extiende el gating de hoy en `enter.Handler.Handle`)

Orden estricto de chequeos, cada uno con su punto de salida propio:

1. Resolver jugador (`roomsession.Player`, ya existe).
2. Cargar room + layout (`loadRoom`, ya existe).
3. **Chequeo de ban** (`entry.BanChecker.IsBanned`) — depende de `room_bans` (no existe todavía). El punto de enganche real ya existe en el código: `internal/realm/room/entry/service.go` declara la interfaz `BanChecker` y el builder `Service.WithBans(checker BanChecker) *Service`; mientras nadie la satisfaga, `service.bans` es `nil` y `checkBan` degrada a "nunca baneado" sin ningún cambio de forma. `plan/rooms/RIGHTS.md` es quien implementa `BanChecker` de verdad y hace el wiring — este documento no necesita ningún cambio adicional cuando eso pase.
4. **Chequeo de lockout** (Parte 3) — si el jugador está congelado para este room específico, rechazo inmediato (`ErrorEntryLocked`), sin tocar layout/world ni comparar contraseña.
5. **Branch por `DoorMode`**:
   - `Open` → entra directo (comportamiento actual, **sin cambios**, ver 1.2).
   - `Password` → compara contra `PasswordHash` vía bcrypt (ver 1.3); falla → registra intento fallido (Parte 3) y rechaza; excede el máximo → freeze + alert (Parte 3); acierta → limpia contador y entra.
   - `Doorbell` → si el jugador ya "tiene derechos" (owner, interino — ver 1.5) entra directo; si no, entra a la **cola de timbre** (Parte 2) en lugar de unirse al room.
   - `Invisible` → exige "tiene derechos" (owner, interino) **o** `permission.Checker.HasPermission(ctx, playerID, room.EnterAny)`; si ninguna se cumple, rechazo explícito (`ErrorAccessDenied`) — a diferencia de Arcturus, que falla en silencio (mismo criterio ya fijado en `CATALOG.md`/`CURRENCIES-INVENTORY.md`: nunca dejar al cliente sin feedback cuando no hay una razón de seguridad real para el silencio, y acá no la hay — el room ya apareció en el navegador).
6. **Bypass de confianza** (`Trusted`, Parte 4.3) — un origen privilegiado (admin, teleportador) puede saltar los pasos 4 y 5 completos, pero **nunca** el paso 3 (ban) salvo que además tenga `room.EnterAny` resuelto vía permiso real.
7. Room-full check (`roomlive.ErrRoomFull`, ya existe) — salvo `permission.RoomEnterFull`.
8. Join + packets de bootstrap (`handler.join`/`sendEntered`/`broadcastJoined`, ya existen, **sin cambios**).

### 1.2 `DoorModeOpen`

Sin cambios. Se corre la suite de tests existente de `enter/command_test.go` sin modificarla como parte de la validación de este plan — es la regresión explícita que este documento no debe romper.

### 1.3 `DoorModePassword`

- `model.Room` gana `PasswordHash *string` (Parte 5.1), mapeado 1:1 desde la columna `password_hash` ya existente. `nil` significa "sin contraseña seteada" — relevante para el flujo de settings de `REMAINING-ROOMS.md` Parte 3 ("cambiar a `DoorModePassword` sin mandar una contraseña y sin tener una ya seteada falla").
- **Hashing**: `golang.org/x/crypto/bcrypt` — dependencia nueva (confirmado, no existe hoy ningún hashing en el proyecto). Costo default de bcrypt (factor 10, decenas de ms) es aceptable acá porque comparar una contraseña de room **no es un hot path de alta frecuencia** — corre una vez por intento de entrada a un room `Password`-gated, y la enorme mayoría de rooms son `Open` y ni siquiera llegan a este branch (ver Parte 6).
- La columna `password_hash` guarda el HASH (bcrypt siempre produce 60 bytes), no el texto plano — el límite de longitud real se valida sobre la contraseña ORIGINAL antes de hashear (en el service de settings de `REMAINING-ROOMS.md` Parte 3, ej. máx. 25 caracteres), no sobre la columna.
- Flujo: `bcrypt.CompareHashAndPassword([]byte(*room.PasswordHash), []byte(command.Password))`.
- **Riesgo real detectado y a corregir**: `internal/command.Dispatcher.Dispatch` (ya existente, `internal/command/dispatcher.go`) loguea `zap.Any("command", envelope.Command)` en **cada** dispatch. Sin corrección, esto volcaría la contraseña de room en texto plano al log estructurado en cada intento de entrada. Este plan agrega un método de log redactado al propio `enter.Command` (ej. implementar `zapcore.ObjectMarshaler` o simplemente no exponer `Password` vía un `String()`/`MarshalLogObject` que la reemplace por `"***"` cuando no está vacía) — se resuelve en el tipo, **sin tocar el dispatcher genérico compartido por todos los comandos**.

### 1.4 `DoorModeDoorbell`

Ver Parte 2 completa (cola + hangout).

### 1.5 `DoorModeInvisible`

- Interinamente (antes de que `plan/rooms/RIGHTS.md` implemente `room_rights`), "tiene derechos" se degrada a "es el dueño" — mismo criterio ya usado hoy por `roomlive.Room.CanManageFurniture`, no una convención nueva de este plan. El punto de enganche real ya existe: `internal/realm/room/entry/service.go` declara `RightsChecker` y `Service.WithRights(checker RightsChecker) *Service`; con `service.rights == nil`, `hasRights` degrada a `false` sin ningún cambio de forma.
- Además del dueño, un staff con `room.EnterAny` (ya implementado en `internal/realm/room/permissions.go`) siempre entra, sin importar el `DoorMode`.
- Rechazo explícito (`ROOM_ENTER_ERROR(ErrorAccessDenied)`) a diferencia del silencio de Arcturus.
- `plan/rooms/RIGHTS.md` implementa `RightsChecker` de verdad y hace el wiring — este documento no necesita ningún cambio adicional cuando eso pase (mismo punto de enganche que 1.6 deja para el ban).

### 1.6 Ban (dependencia adelantada de R3)

Ver punto 3 del pipeline en 1.1 — el hueco ya está reservado, la implementación real llega con R3.

---

## Parte 2 — Cola de timbre (doorbell) y hangout timeout (~5 minutos)

### 2.1 Estado en memoria

Nuevo estado en `roomlive.Room` (mismo archivo `internal/realm/room/live/room.go`, protegido por el mismo `mutex` que ya protege `occupants`):

```go
// doorbellEntry describes one player waiting for owner/rights-holder approval.
type doorbellEntry struct {
    // Occupant stores the fields needed to notify the waiting player once resolved.
    Occupant    Occupant
    // RequestedAt stores when the request was created or last refreshed.
    RequestedAt time.Time
}

// doorbell stores players waiting for approval, lazily allocated — nil until the
// first doorbell request, so rooms that never use DoorModeDoorbell pay nothing for it.
doorbell map[int64]doorbellEntry
```

### 2.2 Ciclo de vida, paso a paso

1. Un jugador manda `RoomEnter{roomId}` contra un room `DoorModeDoorbell` sin "tener derechos" (owner, interino — 1.5).
2. `enter.Handler` **no** llama a `Runtime.Join` — llama a `active.RequestDoorbell(occupant)` (nuevo método de `roomlive.Room`).
3. `RequestDoorbell`:
   - Si ya existe una entrada para ese `playerID`, la actualiza (refresca `RequestedAt`) en vez de duplicarla.
   - Si **ningún** rights-holder está presente en `occupants` en ese instante (degradado a "está el dueño"), rechaza inmediato (`ErrDoorbellNoOwnerPresent`) — `enter.Handler` lo traduce a `ROOM_ENTER_ERROR(ErrorAccessDenied)`, sin entrar a la cola. Mismo comportamiento que Arcturus ("si nadie con derechos está parado en el room, se rechaza al toque").
   - Si hay al menos un rights-holder presente, agrega la entrada a `doorbell` y retorna éxito.
4. Tras un `RequestDoorbell` exitoso, `enter.Handler` **no** manda los packets de bootstrap del room (el jugador sigue, lógicamente, en el hotel view) — manda `doorbell/add` (outbound nuevo, 2.5) al que tocó ("estás esperando"), y hace `broadcast.RoomPacket` de una variante de `doorbell/add` ("alguien quiere entrar") a cada rights-holder presente.
5. Un rights-holder manda `doorbell/respond{playerId, accept}` (inbound + comando nuevo, 2.5).
6. `doorbell.respond.Handler`:
   - Valida que quien responde efectivamente "tiene derechos" sobre ESE room (mismo degradado interino).
   - Llama a `active.ResolveDoorbell(targetPlayerID, accepted)`:
     - `accepted == true` → remueve la entrada; el handler entonces ejecuta el flujo normal de join+bootstrap para ese jugador, **reusando exactamente** `handler.join`/`sendEntered`/`broadcastJoined` de `enter/command.go` — el mismo camino que un `DoorModeOpen`, sin duplicar lógica.
     - `accepted == false` → remueve la entrada; el rechazado recibe `doorbell/hide` + `ROOM_ENTER_ERROR(ErrorAccessDenied)`.
   - Manda `doorbell/hide` a los demás rights-holders que también habían recibido el aviso (para cerrar el popup en sus pantallas, ya que alguien más ya respondió).
7. **Timeout de hangout** (nuevo, sin equivalente en Arcturus) — ver 2.3.
8. **Vaciado por ausencia de rights-holder** — ver 2.4.

### 2.3 Timeout vía el tick loop existente — sin timers por jugador

**Decisión de diseño explícita, justificada por costo/allocations**: no se crea un `time.Timer`/goroutine por cada jugador en cola. Eso escalaría mal si varios tocan timbre a la vez, y complica la cancelación cuando alguien responde antes de tiempo (habría que cancelar un timer por cada resolución). En su lugar, el loop que ya corre por room activo (`live/loop.go`, `DefaultTickInterval = 500ms`) gana **un segundo método independiente**, sin tocar `Room.Tick()` (que hoy solo se ocupa de movimiento, tiene su propio contrato/tests, y no debería mezclar responsabilidades):

```go
// internal/realm/room/live/doorbell.go

// DoorbellExpired describes one doorbell request removed by timeout or by the
// departure of every present rights-holder.
type DoorbellExpired struct {
    // PlayerID identifies the waiting player.
    PlayerID int64
    // Occupant stores the connection fields needed to notify the player.
    Occupant Occupant
    // Reason distinguishes timeout from no-rights-holder-present, for logging only —
    // both map to the same ROOM_ENTER_ERROR code client-side.
    Reason DoorbellExpiredReason
}

// SweepDoorbell removes doorbell requests older than timeout, returning the removed
// entries. Returns nil immediately when doorbell is nil (the overwhelmingly common
// case: a room that never used DoorModeDoorbell pays nothing beyond a pointer check).
func (room *Room) SweepDoorbell(now time.Time, timeout time.Duration) []DoorbellExpired { ... }
```

`live/loop.go` (`runLoop`) llama a `room.SweepDoorbell(time.Now(), hangoutTimeout)` en el mismo `case <-ticker.C:` que ya llama a `room.Tick()` — dos métodos independientes sobre el mismo ticker, ninguna goroutine nueva. Un nuevo `DoorbellPublisher` (mismo patrón que el `MovementPublisher` ya existente, wireado vía una nueva `RegistryOption`, ej. `WithDoorbellPublisher`) recibe las entradas expiradas y manda `doorbell/hide` + `ROOM_ENTER_ERROR(ErrorEntryTimedOut)` a cada jugador expulsado — reusa el mismo `connections *netconn.Registry` que el broadcaster de movimiento ya tiene, sin abrir ninguna ruta de I/O nueva.

Costo: la cola esperada es minúscula (unos pocos jugadores como mucho, casi siempre 0 para la enorme mayoría de rooms, que ni siquiera son `Doorbell`). Un `for range` sobre un mapa de tamaño ~0-5 cada 500ms es insignificante comparado con el trabajo que el tick ya hace para pathfinding de unidades en movimiento activo. No hace falta un heap de expiración ni ninguna estructura más sofisticada — decisión deliberada, no deuda técnica, siguiendo el criterio ya fijado en este proyecto de no construir para hipotéticos que el research no confirma.

### 2.4 Vaciado si no queda ningún rights-holder presente

`Room.Leave(playerID)` (método existente) se extiende: tras remover al jugador de `occupants`, si `doorbell` no está vacío, chequea si algún rights-holder (owner, interino) sigue presente; si no queda ninguno, vacía **toda** la cola reusando la misma función interna que 2.3 (con `Reason: DoorbellReasonNoRightsHolder` en vez de `DoorbellReasonTimeout` — la distinción es solo para logging/telemetría interna, el código de protocolo (`ErrorAccessDenied`, no `ErrorEntryTimedOut`) es el mismo que un rechazo directo, ya que del lado del cliente "se fue el dueño" y "nadie respondió nunca" son indistinguibles y no hace falta que lo sean).

### 2.5 Packets nuevos

| Dirección | Paquete | Header | Contenido |
| --- | --- | --- | --- |
| Inbound | `networking/inbound/room/doorbell/respond` | TBD — a confirmar contra Nitro real | `playerId int32` (a quién se responde), `accept bool` |
| Outbound | `networking/outbound/room/doorbell/add` | TBD | Variante "estás esperando" (al que tocó): sin campos, mismo criterio que `outentered.Encode()`. Variante "alguien quiere entrar" (a cada rights-holder): `username string`. |
| Outbound | `networking/outbound/room/doorbell/hide` | TBD | Sin campos — cierra el popup en quien lo reciba (el que tocó, o un rights-holder que ya no necesita seguir viéndolo). |

`ROOM_ENTER_ERROR` (899, ya existente) gana códigos nuevos, todos como valores adicionales del mismo `int32`, sin tocar el shape del packet:

```go
const (
    ErrorRoomFull      int32 = 1 // ya existe
    ErrorWrongPassword int32 = 2
    ErrorBanned        int32 = 3 // reservado para R3 — no se usa hasta que exista room_bans
    ErrorAccessDenied  int32 = 4 // invisible sin derechos, timbre rechazado o vaciado
    ErrorEntryLocked   int32 = 5 // freeze por máximo de intentos, Parte 3
    ErrorEntryTimedOut int32 = 6 // timeout de hangout en la cola de timbre
)
```

### 2.6 Config

```go
// internal/realm/room/commands/enter/config.go

// Config controls closed-room entry gating behavior.
type Config struct {
    // HangoutTimeout stores how long a doorbell request waits before auto-rejecting.
    HangoutTimeout time.Duration `env:"PIXELS_ROOM_ENTRY_HANGOUT_TIMEOUT" envDefault:"5m"`

    // MaxPasswordAttempts stores wrong-password attempts allowed before a lockout (Parte 3).
    MaxPasswordAttempts int `env:"PIXELS_ROOM_ENTRY_MAX_PASSWORD_ATTEMPTS" envDefault:"5"`

    // AttemptWindow stores the rolling window during which attempts accumulate (Parte 3).
    AttemptWindow time.Duration `env:"PIXELS_ROOM_ENTRY_ATTEMPT_WINDOW" envDefault:"5m"`

    // LockoutDuration stores how long entry stays frozen after exceeding MaxPasswordAttempts (Parte 3).
    LockoutDuration time.Duration `env:"PIXELS_ROOM_ENTRY_LOCKOUT_DURATION" envDefault:"10m"`
}
```
Mismo patrón ya usado por `pkg/i18n.Config`/`pkg/redis.Config` (`env` tags + `envDefault`, cargado con `env.ParseAs[Config]()`).

### 2.7 Tests

- Tocar timbre sin ningún rights-holder presente → rechazo inmediato, sin entrada en `doorbell`.
- Tocar timbre con el dueño presente → el dueño recibe `doorbell/add` (variante staff), el que toca recibe `doorbell/add` (variante espera), entrada queda en `doorbell`.
- Tocar timbre dos veces seguidas el mismo jugador → una sola entrada, `RequestedAt` se refresca (no se duplica).
- Aceptar → el jugador entra normal (mismos packets de bootstrap que `DoorModeOpen`), la entrada desaparece, otros rights-holders reciben `doorbell/hide`.
- Rechazar → el jugador recibe `doorbell/hide` + `ROOM_ENTER_ERROR(ErrorAccessDenied)`.
- Reloj inyectado avanzado más allá de `HangoutTimeout` sin respuesta → `SweepDoorbell` expulsa la entrada; test unitario directo sobre `SweepDoorbell`, sin `time.Sleep` real (mismo patrón que ya usan `live/loop_test.go`/`live/movement_test.go`).
- El único rights-holder presente se va con la cola no vacía → toda la cola se vacía con `ErrorAccessDenied` (`Reason` interno distinto, mismo código de protocolo).
- `doorbell == nil` (room que nunca usó timbre) → `SweepDoorbell` retorna `nil` sin iterar nada (test de humo para la rama de allocation cero).

---

## Parte 3 — Máximo de intentos de contraseña, freeze, y alertas

### 3.1 Por qué Redis y no en memoria

El estado de intentos debe **sobrevivir a que el room se descargue de memoria** entre intentos — si alguien falla 4 veces, se va, y vuelve cuando el room ya se descargó por estar vacío (`Registry.UnloadIdle`, ya existente), el contador debe seguir en 4, no resetearse a 0. Un contador en `roomlive.Room` (memoria) se perdería exactamente en ese escenario — el mismo tipo de bug que `REMAINING-ROOMS.md` Parte 4.2 ya identificó y decidió NO replicar para el mute de Arcturus (`Room.mutedHabbos`, en memoria, se pierde si el room se descarga). Redis además deja el lockout vigente aunque el jugador reconecte con una sesión nueva, y sienta la base para cuando exista más de una instancia de realm corriendo (mismo razonamiento ya usado en `PERMISSIONS.md` Parte 3.5 para el cache de grupos de permisos).

### 3.2 Diseño de claves

```
room:entry:attempts:{roomID}:{playerID}  → contador entero, TTL = AttemptWindow
room:entry:lockout:{roomID}:{playerID}   → clave de solo-presencia, TTL = LockoutDuration
```

Namespace `room:entry:*`, siguiendo la misma convención de un prefijo por dominio ya en uso por `internal/auth/sso` y `internal/permission/cache` (sin overlap posible entre dominios).

### 3.3 Nuevo método atómico en `pkg/redis.Client`

El cliente hoy expone `Find`/`Set`/`Delete`/`Expire`/`Take` — ninguno alcanza para "incrementar un contador atómicamente, fijando el TTL solo la primera vez que se crea la clave". Se agrega:

```go
// pkg/redis/client.go

// Increment atomically increments a counter key by one, applying ttl only the first
// time the key is created — subsequent calls never extend or reset the window. Used
// for fixed-window rate limiting where the window must start on the FIRST failure,
// not slide forward on every subsequent one.
func (client *Client) Increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
    pipeline := client.client.TxPipeline()
    counter := pipeline.Incr(ctx, key)
    pipeline.ExpireNX(ctx, key, ttl)
    if _, err := pipeline.Exec(ctx); err != nil {
        return 0, err
    }

    return counter.Val(), nil
}
```

`INCR` + `EXPIRE NX` en un pipeline (una sola ida y vuelta de red, dos comandos) — `INCR` ya es atómico en Redis por diseño; `EXPIRE NX` (soportado por `go-redis/v9`, confirmado en `go.mod` — `v9.21.0`) aplica el TTL solo si la clave todavía no tiene uno, exactamente "la ventana arranca en el primer fallo", sin necesitar un script Lua ni una transacción `WATCH`/`MULTI` de lectura-modificación-escritura (que sí tendría una condición de carrera real: dos intentos fallidos casi simultáneos del mismo jugador podrían pisarse entre el `Find` y el `Set` si se implementara como lectura+escritura separadas en vez de `INCR`).

Test nuevo en `pkg/redis/client_test.go` (mismo patrón `miniredis.RunT`, sin Redis real): incrementar la misma clave 3 veces seguidas confirma que retorna `1, 2, 3`, y que el TTL fijado en la primera llamada no se resetea en la segunda/tercera.

### 3.4 Flujo de gating con lockout

```go
// checkLockout reports whether a player is currently frozen out of a password-gated room.
func (handler Handler) checkLockout(ctx context.Context, roomID int64, playerID int64) (bool, error) {
    _, found, err := handler.Redis.Find(ctx, lockoutKey(roomID, playerID))

    return found, err
}

// registerFailedAttempt increments the attempt counter and freezes entry once it
// crosses MaxPasswordAttempts, returning whether a freeze was newly applied.
func (handler Handler) registerFailedAttempt(ctx context.Context, roomID int64, playerID int64) (bool, error) {
    count, err := handler.Redis.Increment(ctx, attemptsKey(roomID, playerID), handler.Config.AttemptWindow)
    if err != nil || count < int64(handler.Config.MaxPasswordAttempts) {
        return false, err
    }

    return true, handler.Redis.Set(ctx, lockoutKey(roomID, playerID), []byte{'1'}, handler.Config.LockoutDuration)
}
```

- `checkLockout` corre **antes** de comparar la contraseña con bcrypt — evita gastar el costo de bcrypt en cada intento mientras el jugador ya está congelado, y como beneficio secundario evita filtrar por timing si está congelado o no.
- Una contraseña **correcta** borra el contador de intentos (`Redis.Delete(attemptsKey(...))`) pero **no** borra un lockout ya activo — si ya cruzaste el máximo y quedaste congelado, escribir la contraseña correcta durante el freeze no lo salta. De lo contrario el freeze no protegería nada real (un atacante con un diccionario de contraseñas simplemente probaría la correcta al final de la ventana igual). El freeze solo se levanta por expiración natural de TTL.
- Camino feliz (contraseña correcta, sin freeze, primer intento): **una sola** llamada a Redis (`checkLockout`) — y esa llamada solo corre dentro del branch `DoorModePassword`, nunca para `Open`/`Doorbell`/`Invisible`.

### 3.5 Alertas ("usar alerts y demás")

- **Intento fallido individual** (no cruza el máximo todavía): `ROOM_ENTER_ERROR(ErrorWrongPassword)` — mismo mecanismo minimalista ya usado para `ErrorRoomFull`, sin contador visible de intentos restantes (mismo comportamiento que Arcturus, que tampoco expone eso; agregar un contador visible requeriría un packet nuevo sin ninguna evidencia de que Nitro lo soporte).
- **Cruzar el máximo** (recién se activa el freeze): además del `ROOM_ENTER_ERROR(ErrorEntryLocked)` de ese intento puntual, se manda un `GENERIC_ALERT` (3801, ya existente) con un mensaje **localizado** vía `pkg/i18n` (ej. key `room.entry.locked`, params `{"minutes": "10"}`). Este es el punto concreto donde este plan cumple con "usar alerts y demás": el rechazo rutinario usa el código minimalista de siempre, pero cruzar el umbral es un evento distinto y más importante, con un mensaje explicativo real en el idioma del jugador — no solo un número que el cliente deba interpretar sin contexto.
- **Intentos posteriores mientras ya está congelado**: directo `ROOM_ENTER_ERROR(ErrorEntryLocked)`, **sin** volver a mandar el `GENERIC_ALERT` en cada intento adicional — el alert es un evento puntual ("acabás de cruzar la línea"), no un recordatorio en cada click, para no generar spam de diálogos si el cliente reintenta rápido.

### 3.6 Tests

- 1 a `MaxPasswordAttempts - 1` intentos fallidos → `ErrorWrongPassword` cada vez, sin `GENERIC_ALERT`, sin clave de lockout.
- El intento número `MaxPasswordAttempts` → `ErrorEntryLocked` + `GENERIC_ALERT` localizado + clave de lockout creada con el TTL correcto.
- Intento posterior durante el freeze, **incluso con la contraseña correcta** → `ErrorEntryLocked` inmediato, sin invocar el comparador bcrypt (verificable con un fake que falla el test si se invoca durante un lockout activo — asegura el orden, no solo el resultado).
- Contraseña correcta antes de cruzar el máximo → entra normal, contador de intentos se borra de Redis.
- Reloj/TTL simulado de `miniredis` avanzado más allá de `LockoutDuration` → el freeze expira solo, sin ninguna acción explícita de "unlock".
- La ventana de intentos **no** se resetea en el segundo/tercer fallo (usa `Increment` con `ExpireNX`, no `Set`) — test directo sobre `pkg/redis.Client.Increment`.

---

## Parte 4 — Sistema de Forwarding

### 4.1 Los dos mecanismos, nunca mezclados

**Mecanismo A — Redirect-and-reenter (`ROOM_FORWARD`, ya existe hoy)**: el servidor no mueve al jugador de room del lado del engine — solo manda `ROOM_FORWARD(targetRoomID)` (160) al cliente, y es el **cliente** quien decide re-disparar una entrada normal (hoy: un `RoomEnter` nuevo; a futuro, `FORWARD_TO_SOME_ROOM` corregido, 4.4). Se usa cuando la transición **no** ocurre dentro de una secuencia que el servidor ya está controlando en tiempo real — el disparador viene de "afuera" del room (un admin, el navegador, una invitación futura). El engine no puede asumir que el cliente ya tiene cargados los assets/estado del room destino, así que le pide que renavegue por su cuenta.

**Mecanismo B — Rejoin directo del lado servidor (nuevo, sin round-trip de protocolo)**: el servidor llama directo a la función interna de "salir del room actual + unirse al nuevo" — reusa exactamente `handler.join`/`sendEntered`/`broadcastJoined` de `enter/command.go` y `enter/runtime.go`, ya existentes — **sin** pasar por `ROOM_FORWARD`. Se usa cuando la transición ocurre **dentro** de una secuencia que el servidor ya controla de punta a punta y el jugador ya está "adentro" de una interacción — el caso confirmado es el teleportador de furniture (`INTERACTIONS.md` Milestone I6): el click ya lo procesó el servidor, la secuencia de animación con delays ya la orquesta el servidor, así que el servidor simplemente continúa esa misma secuencia terminando en un join al room destino y mandando los packets de bootstrap directamente.

**Regla de decisión**: ¿el servidor ya tiene control síncrono de principio a fin de esta transición, con el jugador ya dentro de una interacción que el servidor mismo orquesta? Sí → Mecanismo B. No (el disparador viene de afuera del room) → Mecanismo A.

### 4.2 `Trusted`: bypass de confianza, separado del bypass de ban

`enter.Command` gana un campo nuevo:

```go
// Command joins a room.
type Command struct {
    Handler  netconn.Context
    RoomID   int64
    Password string

    // Trusted marks entries originated by a privileged source (admin HTTP teleport,
    // furniture teleporter) that may bypass door-mode gating (password/doorbell/
    // invisible) without holding any per-request permission node — the CALLER is
    // itself the trust boundary (already gated by admin auth, or by only running for
    // an authenticated in-session player clicking furniture), not this field.
    Trusted bool
}
```

`Trusted: true` **nunca** saltea el chequeo de ban (paso 3 del pipeline, 1.1) — un ban significa "esta persona no entra a este room bajo ninguna circunstancia", conceptualmente distinto del control de acceso social (password/doorbell/invisible). La única forma de saltar **también** el ban es que quien pide la entrada tenga `room.EnterAny` resuelto vía `permission.Checker.HasPermission` de verdad — no alcanza con `Trusted: true` a secas. Esto separa dos preguntas distintas y las hace auditables por separado: "¿este trigger viene de un sistema privilegiado?" (booleano interno, sin permiso de por medio, decidido por el propio código que arma el `Command`) vs. "¿el jugador en cuestión tiene un permiso real, revocable, que lo exime incluso de un ban?" (chequeo de permiso auditable).

### 4.3 Arreglar el stub de `FORWARD_TO_SOME_ROOM` (1703)

Hoy `networking/inbound/navigator/forward/packet.go` decodifica `Payload{}` — cero campos, pese a que el header ya está reservado, y no hay ningún handler registrado para él en ningún `HandlerRegistry`. Este plan:
- Agrega el campo real: `Payload{RoomID int32}`, `Definition = codec.Definition{codec.Named("roomId", codec.Int32Field)}`.
- Agrega un handler (`internal/realm/navigator/handlers/forward`, o reusa `room/handlers/enter` con un decoder de entrada distinto — a decidir en implementación cuál genera menos duplicación) que traduce el payload a un `enter.Command{RoomID: payload.RoomID, Trusted: false}` normal — es decir, este packet es simplemente **otra forma de disparar el mismo pipeline de entrada de siempre**, con el mismo gating completo, no una puerta trasera.

### 4.4 Nuevo endpoint HTTP admin: teleport de un solo jugador

`pkg/http/room/routes` ya tiene `POST /api/admin/rooms/:id/forward` (bulk — TODOS los ocupantes actuales de un room). Falta el caso "un jugador puntual, esté donde esté":

```go
app.Post(roomPath+"/players/:playerId/teleport", teleportPlayerHandler(runtime, connections))
```
```go
// TeleportRequest contains a single-player forced relocation request.
type TeleportRequest struct {
    // TargetRoomID identifies the destination room.
    TargetRoomID int64 `json:"targetRoomId"`
    // Bypass, when true, marks the resulting re-entry as Trusted (Parte 4.2) — the
    // admin explicitly asking to ignore the target room's password/doorbell/invisible
    // gating, never implied by default.
    Bypass bool `json:"bypass"`
}
```

Implementación: resuelve la conexión activa del jugador (mismo patrón ya usado por `pkg/http/notification/routes.playerConnection`), manda `ROOM_FORWARD(targetRoomID)` a esa conexión (Mecanismo A — el jugador puede estar en cualquier estado de cliente; lo más seguro es pedirle que renavegue, no asumir nada del lado servidor). El propio flujo de re-entrada del cliente ya dispara `leavePreviousRoom` como efecto colateral (`enter/runtime.go:join`, ya existente) — este endpoint **no** llama a `Runtime.Leave` de forma proactiva, para no duplicar esa lógica. Cuando `Bypass: true`, el `roomId` viaja acompañado de una marca server-side (ej. una entrada de corta vida en un mapa `pendingTrustedEntries` o, más simple, una clave Redis efímera `room:entry:trusted:{roomID}:{playerID}` con TTL de unos segundos) que el handler de re-entrada consulta una sola vez y borra — evita tener que rediseñar el packet `ROOM_FORWARD`/`FORWARD_TO_SOME_ROOM` para transportar un flag de confianza que el cliente no debería poder falsificar mandándolo él mismo.

### 4.5 Teleportadores de furniture (cruce con `INTERACTIONS.md` Milestone I6)

Confirma y precisa lo que `INTERACTIONS.md` ya dejó anotado ("reusa el mismo camino que ya usa `room.enter`"):
- El handler de click de teleportador (a implementar en I6) construye `enter.Command{RoomID: pairedRoom.ID, Trusted: false}` — **sin bypass por defecto**, para que el gating de puerta del room destino siga aplicando, exactamente igual que Arcturus (que literalmente reusa `RoomManager.enterRoom` sin ningún bypass especial — confirmado en el research original de `INTERACTIONS.md`, no es una desviación de Pixels). Si a futuro se decide que los teleportadores sí deban ignorar password/doorbell del destino, es cambiar un booleano acá, no un rediseño.
- El ban **siempre** aplica (nunca bypaseable vía teleportador) — coherente con 4.2.
- Se usa Mecanismo B (rejoin directo, sin `ROOM_FORWARD`), porque el click ya es una secuencia 100% controlada por el servidor con delays (I6) — exactamente el caso que motiva el Mecanismo B en 4.1.

### 4.6 Futuras features de usuario (visitar amigo, invitaciones, "seguir a")

Cualquier feature futura donde un usuario (no un admin, no una interacción de furniture) desencadene que un jugador entre a un room específico usa Mecanismo A, **sin** `Trusted: true` — un jugador nunca se salta el password/doorbell/invisible de un room ajeno solo porque un amigo lo invitó; la invitación en el mejor de los casos ahorra buscar el room en el navegador, no regala acceso. Un futuro sistema de "invitación con acceso real" (el dueño invita a alguien puntual a su room password-gated) necesitaría su propio mecanismo de permiso puntual (ej. una allowlist temporal por jugador) — **no** el bypass genérico de este documento. Queda anotado en "Milestones futuros confirmados", no descartado.

### 4.7 Tabla resumen

| Origen | Mecanismo | Bypass de gating (password/doorbell/invisible) | Bypass de ban |
| --- | --- | --- | --- |
| Cliente pidiendo `RoomEnter` normal | — (no es forwarding) | No | No |
| Navegador / "visitar amigo" / invitación futura | A (`ROOM_FORWARD` + re-entrada) | No | No |
| Admin HTTP `teleport` sin `Bypass` | A | No | No |
| Admin HTTP `teleport` con `Bypass: true` | A + `Trusted: true` | Sí | Solo si además tiene `room.EnterAny` |
| Admin HTTP `forward` bulk (ya existente) | A (varios jugadores a la vez) | No (hoy) — extensible con el mismo `Bypass` si hace falta en la práctica | No |
| Teleportador de furniture (`INTERACTIONS.md` I6) | B (rejoin directo, sin round-trip) | No, por defecto | No |

---

## Parte 5 — Cambios de modelo, esquema, y nodos de permiso

### 5.1 `model.Room` gana `PasswordHash`

```go
// PasswordHash stores the bcrypt hash of the room's entry password, nil when unset.
PasswordHash *string
```

`repository/scan.go` (`scanRoom`) se extiende para escanear `password_hash` (`pgtype.Text` → `*string`, mismo patrón ya usado para `CategoryID`/`DeletedAt` vía `pgtype.Int8`/`pgtype.Timestamptz` + un helper `stringPointer`, análogo a los `int64Pointer`/`timePointer` ya existentes en el mismo archivo).

### 5.2 `internal/realm/room/permissions.go` — primer archivo de nodos del realm `room`

```go
package room

import "github.com/niflaot/pixels/internal/permission"

var (
    // EnterAny allows entering any room regardless of door mode or bans.
    EnterAny = permission.RegisterNode("room.enter.any", "")

    // EnterFull allows entering a room already at capacity.
    EnterFull = permission.RegisterNode("room.enter.full", "")
)
```

Mismos dos nodos ya previstos en `plan/PERMISSIONS.md` Parte 3.1 — este plan es el primero en necesitarlos realmente y crea el archivo. El resto de los nodos de `room` (moderación/derechos `.own`/`.any`) se agregan al mismo archivo cuando `REMAINING-ROOMS.md` R2/R3 se implementen — no antes.

### 5.3 Namespace Redis completo de este plan

```
room:entry:attempts:{roomID}:{playerID}
room:entry:lockout:{roomID}:{playerID}
room:entry:trusted:{roomID}:{playerID}   (Parte 4.4, TTL de segundos, uso interno del bypass admin)
```

Sin overlap con `sso:*` (`internal/auth/sso`) ni con el namespace de `internal/permission/cache` — mismo criterio de un prefijo por dominio ya en uso en el proyecto.

---

## Parte 6 — Hot paths y allocations (resumen transversal)

- **Camino feliz dominante (`DoorModeOpen`)**: cero cambios de allocation respecto a hoy — el pipeline de gating (1.1) resuelve el branch `Open` en la primera rama de un `switch`, sin tocar Redis, sin tocar bcrypt, sin tocar `doorbell` (que puede seguir `nil`).
- **`doorbell` lazy**: `nil` hasta el primer `RequestDoorbell` — la enorme mayoría de rooms (no-`Doorbell`) no pagan ni el tamaño de un mapa vacío de más en `roomlive.Room`.
- **`SweepDoorbell` de costo casi nulo cuando no hace falta**: si `doorbell == nil`, retorna sin iterar nada — un chequeo de puntero, no una asignación.
- **Redis solo en el camino infeliz de `DoorModePassword`**: `checkLockout` es la única llamada Redis que corre en TODO intento de entrada a un room `Password`-gated (ineludible, es la razón de ser del feature) — rooms `Open`/`Doorbell`/`Invisible` nunca la ejecutan. `Increment`/`Set` de lockout solo corren tras una contraseña incorrecta, nunca en el camino de éxito.
- **bcrypt solo tras pasar el lockout check** — evita el costo de CPU (deliberadamente no trivial, del orden de decenas de ms) en cada intento mientras el jugador ya está congelado, además de no filtrar información por timing.
- **Redacción del log de `Command.Password`** (1.3) — evita que la contraseña de room quede en texto plano en el log estructurado de `Dispatcher.Dispatch`, resuelto en el propio tipo `enter.Command`, sin tocar el dispatcher genérico compartido por el resto de los comandos del proyecto.
- **`ROOM_FORWARD` bulk existente** (`forwardOccupants`, `pkg/http/room/routes/handler.go`) ya preasigna el slice de respuesta con `cap` conocido — el nuevo endpoint single-player (4.4) ni siquiera necesita un slice, al mover un solo jugador.
- **Timeout de hangout sin timers por jugador** (2.3) — evita N timers/goroutines concurrentes cuando varios jugadores tocan timbre a la vez; el costo se paga como un sweep O(n) sobre un mapa minúsculo, cada 500ms, reusando el ticker que ya existe.

---

## Parte 7 — Testing (resumen transversal, detalle por parte ya cubierto arriba)

- Todas las suites siguen el patrón ya establecido en `enter/command_test.go`/`enter/*_test.go` (fakes de repository, sin Postgres real) y `pkg/redis/client_test.go` (`miniredis.RunT`, sin Redis real) — nada de esto necesita infraestructura externa para correr en CI.
- Reloj inyectable: cualquier lógica que dependa de `time.Now()` (sweep de timeout, ventana de intentos) recibe un `func() time.Time` configurable — mismo patrón que `internal/auth/sso.Service.now` — nunca `time.Now()` hardcodeado dentro de la lógica de negocio, para poder testear el timeout de 5 minutos y la ventana de intentos sin `time.Sleep` real.
- Regresión explícita: la suite actual de `enter/command_test.go` corre sin modificaciones como parte de la validación de este plan — `DoorModeOpen` no cambia de comportamiento ni de packets.

---

## Parte 8 — Milestones de implementación

1. **E1 — `PasswordHash` + bcrypt + gating de `DoorModePassword`** (1.3, 5.1): agregar el campo/scan, agregar la dependencia `golang.org/x/crypto/bcrypt`, extender `enter.Command`/`Handler` con el chequeo de contraseña, redactar `Password` del log del dispatcher, nuevos códigos de `ROOM_ENTER_ERROR` (`ErrorWrongPassword` como mínimo).
2. **E2 — Máximo de intentos + freeze + alertas** (Parte 3): `pkg/redis.Client.Increment`, claves de intentos/lockout, config nueva, integración de `GENERIC_ALERT` vía `pkg/i18n` — depende de E1 (necesita que el gating de contraseña ya exista para engancharse ahí).
3. **E3 — Cola de timbre sin timeout todavía** (2.1-2.2, 2.5): estado en `roomlive.Room`, comando `doorbell/respond`, packets nuevos — depende de E1 solo en cuanto a compartir el pipeline de gating (1.1), no en lógica.
4. **E4 — Hangout timeout + vaciado por ausencia de rights-holder** (2.3-2.4): `SweepDoorbell`, `DoorbellPublisher`, wiring en `live/loop.go`/`live/registry.go` — depende de E3.
5. **E5 — `internal/realm/room/permissions.go` + nodos de entrada** (5.2): `EnterAny`/`EnterFull`, wiring de `permission.Checker` en `enter.Handler` para el bypass de `DoorModeInvisible` (1.5) y el bypass de ban vía `Trusted` (4.2) — depende de que `internal/permission` ya esté disponible como dependencia del módulo `room` (ya lo está para otros realms del proyecto). **Estado: implementado** (el archivo ya existe con `EnterAny`/`EnterFull`, y además `AnswerAnyDoorbell`, no previsto originalmente acá).
6. **E6 — `Trusted`, arreglo del stub `FORWARD_TO_SOME_ROOM`, endpoint admin single-player teleport** (4.2-4.4): depende de E5 para el bypass de ban. **Estado: implementado** (`GrantTrusted`, `pkg/http/room/routes/teleport.go`).
7. **E7 — Integración con teleportadores de furniture** (4.5): depende de que `INTERACTIONS.md` Milestone I6 exista, y de E6 para `Trusted`/Mecanismo B — se implementa junto con I6, no antes.
8. **E8 — Derechos y ban reales** (1.5, 1.6): reemplaza los stubs `RightsChecker`/`BanChecker` de `internal/realm/room/entry/service.go` por implementaciones reales de `room_rights`/`room_bans` — este milestone se ejecuta enteramente en **`plan/rooms/RIGHTS.md`** (milestones RM1-RM7 ahí), que además agrega moderación (kick/mute/ban) y un sistema completo de auditoría/historial no cubierto por este documento. Hasta que `RIGHTS.md` aterrice, el pipeline ya tiene ambos puntos de enganche reservados (`Service.WithRights`/`Service.WithBans`), sin ningún cambio de forma pendiente acá.

### Milestones futuros confirmados (fuera de este documento, no descartados)

- **Invitación con acceso puntual a un room password-gated** (4.6) — necesita su propio mecanismo de allowlist temporal por jugador, no el bypass genérico de este plan; se define cuando exista el feature de invitaciones en sí.
- **Extender el `forward` bulk admin existente con el mismo `Bypass`** (4.7) — hoy el bulk endpoint no necesita saltar gating porque solo reubica gente hacia AFUERA de un room que se está cerrando, no hacia ADENTRO de uno restringido; se evalúa si en la práctica hace falta.
