package booking

import (
	"errors"
	"time"
)

var (
	ErrSeatAlreadyBooked = errors.New("seat is already booked")
	ErrSessionNotFound   = errors.New("session not found")
)

// Booking represents a seat reservation (held or confirmed).
type Booking struct {
	ID        string    `json:"id"`
	MovieID   string    `json:"movie_id"`
	SeatID    string    `json:"seat_id"`
	UserID    string    `json:"user_id"`
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

type BookingStore interface {
	Book(b Booking) (Booking, error)
	ListBookings(movieID string) []Booking
	ConfirmSession(sessionID string) (Booking, error)
	ReleaseSession(sessionID string) error
}
