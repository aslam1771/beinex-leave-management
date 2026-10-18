package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"beinex-leave-management/internal/database"
)

// HealthResponse represents the structure returned by the /health endpoint.
type HealthResponse struct {
	Status    string            `json:"status"`
	Service   string            `json:"service"`
	Version   string            `json:"version"`
	Timestamp time.Time         `json:"timestamp"`
	Uptime    string            `json:"uptime"`
	Database  map[string]string `json:"database"`
}

// HealthHandler manages the health check endpoint.
type HealthHandler struct {
	db        *database.DB
	startTime time.Time
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(db *database.DB) *HealthHandler {
	return &HealthHandler{
		db:        db,
		startTime: time.Now(),
	}
}

// ServeHTTP handles GET /health requests.
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Allow up to 6 seconds for health check to account for possible Neon cold starts
	ctx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
	defer cancel()

	dbStatus := "connected"
	httpStatus := http.StatusOK

	if h.db == nil {
		dbStatus = "uninitialized"
	} else if err := h.db.Ping(ctx); err != nil {
		dbStatus = "disconnected: " + err.Error()
		httpStatus = http.StatusServiceUnavailable
	}

	overallStatus := "healthy"
	if dbStatus != "connected" {
		overallStatus = "degraded"
	}

	response := HealthResponse{
		Status:    overallStatus,
		Service:   "beinex-leave-management-api",
		Version:   "1.0.0",
		Timestamp: time.Now().UTC(),
		Uptime:    time.Since(h.startTime).Round(time.Second).String(),
		Database: map[string]string{
			"status": dbStatus,
		},
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(response)
}
