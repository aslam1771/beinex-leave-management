package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler_ServeHTTP(t *testing.T) {
	handler := NewHealthHandler(nil) // nil DB tests uninitialized/scaffold state

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		t.Errorf("expected Content-Type application/json; charset=utf-8, got %s", contentType)
	}

	var resp HealthResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response JSON: %v", err)
	}

	if resp.Service != "beinex-leave-management-api" {
		t.Errorf("unexpected service name: %s", resp.Service)
	}
	if resp.Database["status"] != "uninitialized" {
		t.Errorf("expected uninitialized db status for nil db, got %s", resp.Database["status"])
	}
}

func TestHealthHandler_MethodNotAllowed(t *testing.T) {
	handler := NewHealthHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405 Method Not Allowed, got %d", rr.Code)
	}
}
