package health

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type Status struct {
	Status    string    `json:"status"`
	Timestamp string    `json:"timestamp"`
	Version   string    `json:"version,omitempty"`
	Checks    []Check   `json:"checks,omitempty"`
}

type Check struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type Checker func() Check

type Handler struct {
	mu       sync.RWMutex
	version  string
	checkers []Checker
}

func NewHandler(version string) *Handler {
	return &Handler{version: version}
}

func (h *Handler) AddChecker(c Checker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checkers = append(h.checkers, c)
}

func (h *Handler) Healthz() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}
}

func (h *Handler) Readyz() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.mu.RLock()
		checkers := make([]Checker, len(h.checkers))
		copy(checkers, h.checkers)
		h.mu.RUnlock()

		status := Status{
			Status:    "ready",
			Timestamp: time.Now().Format(time.RFC3339),
			Version:   h.version,
		}

		allOK := true
		for _, c := range checkers {
			check := c()
			status.Checks = append(status.Checks, check)
			if check.Status != "ok" {
				allOK = false
			}
		}

		if !allOK {
			status.Status = "degraded"
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	}
}
