package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthz(t *testing.T) {
	h := NewHandler("v0.1.0")
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	h.Healthz()(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "ok" {
		t.Fatalf("expected ok, got %s", w.Body.String())
	}
}

func TestReadyzAllOK(t *testing.T) {
	h := NewHandler("v0.1.0")
	h.AddChecker(func() Check { return Check{Name: "db", Status: "ok"} })
	req := httptest.NewRequest("GET", "/readyz", nil)
	w := httptest.NewRecorder()
	h.Readyz()(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var s Status
	json.NewDecoder(w.Body).Decode(&s)
	if s.Status != "ready" {
		t.Fatalf("expected ready, got %s", s.Status)
	}
}

func TestReadyzDegraded(t *testing.T) {
	h := NewHandler("v0.1.0")
	h.AddChecker(func() Check { return Check{Name: "cache", Status: "down", Message: "timeout"} })
	req := httptest.NewRequest("GET", "/readyz", nil)
	w := httptest.NewRecorder()
	h.Readyz()(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}
