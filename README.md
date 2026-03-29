# Cinema Booking System

A seat reservation backend built in Go that solves the **double-booking problem** under high concurrency — using a Redis-backed session model with atomic writes.

## The Problem

Two users click "Book" on seat A1 at the exact same moment. Only one should win.

```
User A ──► read seat A1 → "free" ──► write booking ──► success
User B ──► read seat A1 → "free" ──► write booking ──► ???
```

Without protection, both succeed and two people show up for the same seat.

## How It's Solved

Seat ownership is claimed with a single atomic Redis `SET NX` (set-if-not-exists) command. Because Redis is single-threaded, only one writer can win — regardless of how many requests arrive simultaneously.

A booking goes through two phases:

```
Hold (TTL = 2 min) ──► Confirm (no TTL, permanent)
                   └──► Release (delete, seat becomes available again)
```

- **Hold** — reserves the seat temporarily. Expires automatically if the user doesn't confirm.
- **Confirm** — makes the booking permanent by removing the TTL.
- **Release** — frees the seat immediately (user cancels or navigates away).

## Stack

- **Go 1.25** — standard `net/http` router with path parameters
- **Redis 7** — atomic seat locking via `SET NX`, TTL-based hold expiry
- **Redis Commander** — web UI to inspect Redis keys during development

## Project Structure

```
.
├── cmd/
│   └── main.go                  # Server entry point, route registration
├── internal/
│   ├── adapters/redis/
│   │   └── redis.go             # Redis client factory
│   └── booking/
│       ├── domain.go            # Booking type, BookingStore interface, errors
│       ├── service.go           # Business logic layer
│       ├── handler.go           # HTTP handlers (request/response)
│       ├── redis_store.go       # Redis implementation (production)
│       ├── memory_store.go      # In-memory implementation (testing)
│       ├── concurrent_store.go  # Mutex-based in-memory store (learning reference)
│       └── service_test.go      # Concurrent booking stress test
├── static/
│   └── index.html               # Vanilla JS frontend
└── docker-compose.yaml          # Redis + Redis Commander
```

## API

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/movies` | List available movies |
| `GET` | `/movies/{movieID}/seats` | List seat statuses for a movie |
| `POST` | `/movies/{movieID}/seats/{seatID}/hold` | Hold a seat (returns session) |
| `PUT` | `/sessions/{sessionID}/confirm` | Confirm a held seat |
| `DELETE` | `/sessions/{sessionID}` | Release a held seat |

### Hold a seat

```bash
POST /movies/inception/seats/A1/hold
{"user_id": "abc123"}

# 201 Created
{
  "session_id": "f47ac10b-...",
  "movie_id":   "inception",
  "seat_id":    "A1",
  "expires_at": "2026-03-29T10:02:00Z"
}
```

### Confirm

```bash
PUT /sessions/f47ac10b-.../confirm
# 200 OK — booking is now permanent
```

### Release

```bash
DELETE /sessions/f47ac10b-...
# 204 No Content — seat is free again
```

## Running Locally

**1. Start Redis**

```bash
docker compose up
```

Redis is available at `localhost:6379`.
Redis Commander UI is available at `http://localhost:8081`.

**2. Start the server**

```bash
go run ./cmd/main.go
```

Server listens on `http://localhost:8080`.

**3. Open the UI**

Navigate to `http://localhost:8080` in your browser. Each browser tab gets a random user ID — open multiple tabs to simulate concurrent users competing for the same seat.

## Redis Key Design

```
seat:{movieID}:{seatID}   →  session JSON   (TTL while held, no TTL when confirmed)
session:{sessionID}       →  session JSON   (reverse lookup by session ID)
```

## Running Tests

The stress test spins up 100,000 goroutines all trying to book the same seat simultaneously and asserts exactly one succeeds.

```bash
go test ./internal/booking/ -v -run TestConcurrentBooking_ExactlyOneWins
```

> Requires a running Redis instance at `localhost:6379`.
