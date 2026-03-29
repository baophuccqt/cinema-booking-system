package booking

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const defaultHoldTTL = 2 * time.Minute

// RedisStore implements session-based seat booking backed by Redis
//
// Key design:
//
// seat:{movieID}:{seatID} --> session JSON (TTL = held, no TTL = confirmed)
// session:{sessionID}     --> session key (reverse lookup)
type RedisStore struct {
	rdb *redis.Client
}

func NewRedisStore(rdb *redis.Client) *RedisStore {
	return &RedisStore{rdb: rdb}
}

func sessionKey(id string) string {
	return fmt.Sprintf("session:%s", id)
}

func (s *RedisStore) Book(b Booking) (Booking, error) {
	session, err := s.hold(b)
	if err != nil {
		return Booking{}, err
	}

	log.Printf("Session booked %v", session)
	return session, nil
}

func (s *RedisStore) ListBookings(movieID string) []Booking {
	pattern := fmt.Sprintf("seat:%s:*", movieID)
	var sessions []Booking

	ctx := context.Background()

	iter := s.rdb.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		val, err := s.rdb.Get(ctx, iter.Val()).Result()
		if err != nil {
			continue
		}

		session, err := parseSession(val)
		if err != nil {
			continue
		}

		sessions = append(sessions, session)
	}

	return sessions
}

// hold is private function (name start with lowercase letter)
func (s *RedisStore) hold(b Booking) (Booking, error) {
	id := uuid.New().String()
	now := time.Now()
	ctx := context.Background()
	key := fmt.Sprintf("seat:%s:%s", b.MovieID, b.SeatID)

	b.ID = id
	val, _ := json.Marshal(b)

	res := s.rdb.SetArgs(ctx, key, val, redis.SetArgs{
		Mode: "NX",
		TTL:  defaultHoldTTL,
	})

	ok := res.Val() == "OK"
	if !ok {
		return Booking{}, ErrSeatAlreadyBooked
	}

	s.rdb.Set(ctx, sessionKey(id), val, defaultHoldTTL)

	return Booking{
		ID:        id,
		MovieID:   b.MovieID,
		SeatID:    b.SeatID,
		UserID:    b.UserID,
		Status:    "held",
		ExpiresAt: now.Add(defaultHoldTTL),
	}, nil
}

func (s *RedisStore) ConfirmSession(sessionID string) (Booking, error) {
	ctx := context.Background()
	sKey := sessionKey(sessionID)

	val, err := s.rdb.Get(ctx, sKey).Result()
	if err == redis.Nil {
		return Booking{}, ErrSessionNotFound
	}
	if err != nil {
		return Booking{}, err
	}

	session, err := parseSession(val)
	if err != nil {
		return Booking{}, err
	}

	session.Status = "confirmed"
	session.ExpiresAt = time.Time{}

	updated, _ := json.Marshal(session)
	seatK := fmt.Sprintf("seat:%s:%s", session.MovieID, session.SeatID)

	pipe := s.rdb.Pipeline()
	pipe.Set(ctx, seatK, updated, 0)  // persist (remove TTL)
	pipe.Set(ctx, sKey, updated, 0)
	if _, err := pipe.Exec(ctx); err != nil {
		return Booking{}, err
	}

	return session, nil
}

func (s *RedisStore) ReleaseSession(sessionID string) error {
	ctx := context.Background()
	sKey := sessionKey(sessionID)

	val, err := s.rdb.Get(ctx, sKey).Result()
	if err == redis.Nil {
		return ErrSessionNotFound
	}
	if err != nil {
		return err
	}

	session, err := parseSession(val)
	if err != nil {
		return err
	}

	seatK := fmt.Sprintf("seat:%s:%s", session.MovieID, session.SeatID)
	return s.rdb.Del(ctx, seatK, sKey).Err()
}

func parseSession(val string) (Booking, error) {
	var data Booking
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return Booking{}, err
	}

	return Booking{
		ID:      data.ID,
		MovieID: data.MovieID,
		SeatID:  data.SeatID,
		UserID:  data.UserID,
		Status:  data.Status,
	}, nil
}
