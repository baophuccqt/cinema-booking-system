package booking

type Service struct {
	store BookingStore
}

func NewService(store BookingStore) *Service {
	return &Service{store}
}

func (s *Service) Book(b Booking) (Booking, error) {
	return s.store.Book(b)
}

func (s *Service) ListBookings(movieID string) []Booking {
	return s.store.ListBookings(movieID)
}

func (s *Service) ConfirmSession(sessionID string) (Booking, error) {
	return s.store.ConfirmSession(sessionID)
}

func (s *Service) ReleaseSession(sessionID string) error {
	return s.store.ReleaseSession(sessionID)
}
