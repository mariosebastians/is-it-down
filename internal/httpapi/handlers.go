package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

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

// handleHealthz godoc
// @Summary      Liveness check
// @Tags         health
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /healthz [get]
func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type monitorInput struct {
	Name            string `json:"name"`
	URL             string `json:"url"`
	IntervalSeconds int    `json:"interval_seconds"`
	TimeoutSeconds  int    `json:"timeout_seconds"`
}

// handleCreateMonitor godoc
// @Summary      Create a monitor
// @Description  Create a new monitor to be polled on its own interval
// @Tags         monitors
// @Accept       json
// @Produce      json
// @Param        monitor  body      monitorInput   true  "Monitor to create"
// @Success      201      {object}  store.Monitor
// @Failure      400      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /monitors [post]
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

// handleListMonitors godoc
// @Summary      List monitors
// @Tags         monitors
// @Produce      json
// @Success      200  {array}   store.Monitor
// @Failure      500  {object}  map[string]string
// @Router       /monitors [get]
func (s *Server) handleListMonitors(w http.ResponseWriter, r *http.Request) {
	monitors, err := s.store.ListMonitors(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list monitors")
		return
	}
	writeJSON(w, http.StatusOK, monitors)
}

// handleGetMonitor godoc
// @Summary      Get a monitor
// @Tags         monitors
// @Produce      json
// @Param        id   path      string  true  "Monitor ID"
// @Success      200  {object}  store.Monitor
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /monitors/{id} [get]
func (s *Server) handleGetMonitor(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	m, err := s.store.GetMonitor(r.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, http.StatusNotFound, "monitor not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get monitor")
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// handleUpdateMonitor godoc
// @Summary      Update a monitor
// @Tags         monitors
// @Accept       json
// @Produce      json
// @Param        id       path      string         true  "Monitor ID"
// @Param        monitor  body      monitorInput   true  "Updated monitor fields"
// @Success      200      {object}  store.Monitor
// @Failure      400      {object}  map[string]string
// @Failure      404      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /monitors/{id} [put]
func (s *Server) handleUpdateMonitor(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, http.StatusNotFound, "monitor not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update monitor")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// handleDeleteMonitor godoc
// @Summary      Delete a monitor
// @Description  Deletes a monitor and cascades its check history
// @Tags         monitors
// @Param        id   path  string  true  "Monitor ID"
// @Success      204
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /monitors/{id} [delete]
func (s *Server) handleDeleteMonitor(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.store.DeleteMonitor(r.Context(), id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, http.StatusNotFound, "monitor not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete monitor")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleListChecks godoc
// @Summary      List recent checks for a monitor
// @Tags         monitors
// @Produce      json
// @Param        id   path      string  true  "Monitor ID"
// @Success      200  {array}   store.Check
// @Failure      500  {object}  map[string]string
// @Router       /monitors/{id}/checks [get]
func (s *Server) handleListChecks(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
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

// handleStatus godoc
// @Summary      Live status of every monitor
// @Description  Returns every monitor together with its most recent check result
// @Tags         status
// @Produce      json
// @Success      200  {array}   statusEntry
// @Failure      500  {object}  map[string]string
// @Router       /status [get]
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
