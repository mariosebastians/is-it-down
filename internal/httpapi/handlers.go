package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"isitdown/internal/store"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type monitorInput struct {
	Name            string `json:"name"`
	URL             string `json:"url"`
	IntervalSeconds int    `json:"interval_seconds"`
	TimeoutSeconds  int    `json:"timeout_seconds"`
}

func (s *Server) handleCreateMonitor(w http.ResponseWriter, r *http.Request) {
	var in monitorInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if in.Name == "" || in.URL == "" {
		writeError(w, http.StatusBadRequest, "name and url are required")
		return
	}
	if in.IntervalSeconds <= 0 {
		in.IntervalSeconds = 60
	}
	if in.TimeoutSeconds <= 0 {
		in.TimeoutSeconds = 10
	}

	created, err := s.store.CreateMonitor(r.Context(), store.Monitor{
		Name:            in.Name,
		URL:             in.URL,
		IntervalSeconds: in.IntervalSeconds,
		TimeoutSeconds:  in.TimeoutSeconds,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create monitor")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleListMonitors(w http.ResponseWriter, r *http.Request) {
	monitors, err := s.store.ListMonitors(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list monitors")
		return
	}
	writeJSON(w, http.StatusOK, monitors)
}

func (s *Server) handleGetMonitor(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	m, err := s.store.GetMonitor(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "monitor not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get monitor")
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (s *Server) handleUpdateMonitor(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var in monitorInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if in.Name == "" || in.URL == "" {
		writeError(w, http.StatusBadRequest, "name and url are required")
		return
	}
	if in.IntervalSeconds <= 0 {
		in.IntervalSeconds = 60
	}
	if in.TimeoutSeconds <= 0 {
		in.TimeoutSeconds = 10
	}

	updated, err := s.store.UpdateMonitor(r.Context(), id, store.Monitor{
		Name:            in.Name,
		URL:             in.URL,
		IntervalSeconds: in.IntervalSeconds,
		TimeoutSeconds:  in.TimeoutSeconds,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "monitor not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update monitor")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDeleteMonitor(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.store.DeleteMonitor(r.Context(), id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "monitor not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete monitor")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListChecks(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	checks, err := s.store.ListChecks(r.Context(), id, 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list checks")
		return
	}
	writeJSON(w, http.StatusOK, checks)
}

type statusEntry struct {
	Monitor store.Monitor `json:"monitor"`
	Latest  *store.Check  `json:"latest_check,omitempty"`
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	monitors, err := s.store.ListMonitors(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load status")
		return
	}

	latest, err := s.store.LatestCheckByMonitor(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load status")
		return
	}

	entries := make([]statusEntry, 0, len(monitors))
	for _, m := range monitors {
		entry := statusEntry{Monitor: m}
		if c, ok := latest[m.ID]; ok {
			entry.Latest = &c
		}
		entries = append(entries, entry)
	}

	writeJSON(w, http.StatusOK, entries)
}
