package booking

import "sync"

type MemoryStore struct {
	seats    map[string]Booking // seatID → booking
	sessions map[string]string  // sessionID → seatID
	sync.Mutex
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		seats:    map[string]Booking{},
		sessions: map[string]string{},
	}
}

func (s *MemoryStore) Book(b Booking) (Booking, error) {
	s.Lock()
	defer s.Unlock()

	if _, exists := s.seats[b.SeatID]; exists {
		return Booking{}, ErrSeatAlreadyBooked
	}

	b.Status = "held"
	s.seats[b.SeatID] = b
	s.sessions[b.ID] = b.SeatID
	return b, nil
}

func (s *MemoryStore) ListBookings(movieID string) []Booking {
	s.Lock()
	defer s.Unlock()

	var bookings []Booking
	for _, b := range s.seats {
		if b.MovieID == movieID {
			bookings = append(bookings, b)
		}
	}
	return bookings
}

func (s *MemoryStore) ConfirmSession(sessionID string) (Booking, error) {
	s.Lock()
	defer s.Unlock()

	seatID, ok := s.sessions[sessionID]
	if !ok {
		return Booking{}, ErrSessionNotFound
	}

	b := s.seats[seatID]
	b.Status = "confirmed"
	s.seats[seatID] = b
	return b, nil
}

func (s *MemoryStore) ReleaseSession(sessionID string) error {
	s.Lock()
	defer s.Unlock()

	seatID, ok := s.sessions[sessionID]
	if !ok {
		return ErrSessionNotFound
	}

	delete(s.seats, seatID)
	delete(s.sessions, sessionID)
	return nil
}
