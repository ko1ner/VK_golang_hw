package handler

import (
	"VK_golang_hw/movie"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

type Handler struct {
	client *movie.Client
}

func NewHandler(c *movie.Client) *Handler {
	return &Handler{client: c}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /load-movies", h.loadMovies)
	mux.HandleFunc("GET /last-movie", h.lastMovie)
}

func (h *Handler) loadMovies(w http.ResponseWriter, r *http.Request) {
	from, err := strconv.Atoi(r.URL.Query().Get("from"))
	if err != nil || from < 1 {
		http.Error(w, "invalid 'from'", http.StatusBadRequest)
		return
	}
	to, err := strconv.Atoi(r.URL.Query().Get("to"))
	if err != nil || to < from {
		http.Error(w, "invalid 'to'", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), movie.RangeTimeout)
	defer cancel()

	movies := make([]*movie.Movie, 0, to-from+1)
	var errs []string

	for id := from; id <= to; id++ {
		mov, err := h.client.GetByID(ctx, id)
		if errors.Is(err, movie.ErrNotFound) {
			errs = append(errs, "movie "+strconv.Itoa(id)+" not found")
			continue
		}
		if errors.Is(err, movie.ErrUpstreamTimeout) {
			http.Error(w, "movie stand timeout", http.StatusGatewayTimeout)
			return
		}
		if err != nil {
			http.Error(w, "movie stand error: "+err.Error(), http.StatusBadGateway)
			return
		}
		movies = append(movies, mov)
	}

	// Тело ответа — массив фильмов (контракт задания). Частичные ошибки
	// отдаём отдельным заголовком, чтобы не ломать формат массива.
	if len(errs) > 0 {
		w.Header().Set("X-Partial-Errors", strings.Join(errs, "; "))
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(movies)
}

func (h *Handler) lastMovie(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), movie.RangeTimeout)
	defer cancel()

	mov, err := h.client.FindLast(ctx)
	if errors.Is(err, movie.ErrNotFound) {
		http.Error(w, "no movie found", http.StatusNotFound)
		return
	}
	if errors.Is(err, movie.ErrUpstreamTimeout) {
		http.Error(w, "movie stand timeout", http.StatusGatewayTimeout)
		return
	}
	if err != nil {
		http.Error(w, "movie stand error: "+err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(mov)
}
