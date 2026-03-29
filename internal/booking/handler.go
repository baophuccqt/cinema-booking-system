package booking

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/sikozonpc/cinema/internal/utils"
)

// --- request / response types ---

type holdRequest struct {
	UserID string `json:"user_id"`
}

type holdResponse struct {
	SessionID string `json:"session_id"`
	MovieID   string `json:"movie_id"`
	SeatID    string `json:"seat_id"`
	ExpiresAt string `json:"expires_at"`
}

type seatResponse struct {
	SeatID string `json:"seat_id"`
	UserID string `json:"user_id"`
	Booked bool   `json:"booked"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// --- handler ---

type handler struct {
	svc Service
}

func NewHandler(svc Service) *handler {
	return &handler{svc: svc}
}

func (h *handler) ListSeats(w http.ResponseWriter, r *http.Request) {
	movieID := r.PathValue("movieID")
	bookings := h.svc.ListBookings(movieID)

	seats := make([]seatResponse, 0, len(bookings))
	for _, b := range bookings {
		seats = append(seats, seatResponse{SeatID: b.SeatID, UserID: b.UserID, Booked: true})
	}

	utils.WriteJSON(w, http.StatusOK, seats)
}

func (h *handler) HoldSeat(w http.ResponseWriter, r *http.Request) {
	movieID := r.PathValue("movieID")
	seatID := r.PathValue("seatID")

	var req holdRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteJSON(w, http.StatusBadRequest, errorResponse{err.Error()})
		return
	}

	session, err := h.svc.Book(Booking{UserID: req.UserID, SeatID: seatID, MovieID: movieID})
	if err != nil {
		utils.WriteJSON(w, http.StatusConflict, errorResponse{err.Error()})
		return
	}

	utils.WriteJSON(w, http.StatusCreated, holdResponse{
		SessionID: session.ID,
		MovieID:   session.MovieID,
		SeatID:    session.SeatID,
		ExpiresAt: session.ExpiresAt.Format(time.RFC3339),
	})
}

func (h *handler) ConfirmSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("sessionID")

	booking, err := h.svc.ConfirmSession(sessionID)
	if err != nil {
		utils.WriteJSON(w, http.StatusNotFound, errorResponse{err.Error()})
		return
	}

	utils.WriteJSON(w, http.StatusOK, booking)
}

func (h *handler) ReleaseSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("sessionID")

	if err := h.svc.ReleaseSession(sessionID); err != nil {
		utils.WriteJSON(w, http.StatusNotFound, errorResponse{err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
