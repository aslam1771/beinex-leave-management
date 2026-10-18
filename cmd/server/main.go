package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"beinex-leave-management/internal/config"
	"beinex-leave-management/internal/database"
	"beinex-leave-management/internal/handlers"
	"beinex-leave-management/internal/repository"
)

func main() {
	log.Println("[server] starting Beinex Employee Leave Management System backend...")

	// 1. Load configuration from environment or .env
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[server] fatal error loading configuration: %v", err)
	}

	// 2. Initialize database connection pool (pgx)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var db *database.DB
	if cfg.HasDatabaseConfig() {
		log.Printf("[database] connecting to PostgreSQL (target: %s)...", cfg.SanitizedDatabaseURL())
		var dbErr error
		db, dbErr = database.NewPostgresPool(ctx, cfg)
		if dbErr != nil {
			log.Printf("[database] WARNING: could not establish database connection: %v", dbErr)
			log.Println("[database] server will continue in degraded mode (health check will report database status)")
		} else {
			defer db.Close()
		}
	} else {
		log.Println("[database] NOTICE: no database credentials specified in environment (DATABASE_URL is unset)")
		log.Println("[database] running in scaffold mode; set DATABASE_URL or DB_* environment variables to enable database connectivity")
	}

	// 3. Setup HTTP router and endpoints
	mux := http.NewServeMux()

	healthHandler := handlers.NewHealthHandler(db)
	mux.Handle("/health", healthHandler)

	var leaveRepo *repository.LeaveRepository
	var employeeRepo *repository.EmployeeRepository
	if db != nil {
		leaveRepo = repository.NewLeaveRepository(db)
		employeeRepo = repository.NewEmployeeRepository(db)
	}
	leaveHandler := handlers.NewLeaveHandler(leaveRepo, employeeRepo)

	// Leave Application & Query Endpoints
	mux.HandleFunc("POST /api/leaves", leaveHandler.ApplyLeave)
	mux.HandleFunc("GET /api/leaves", leaveHandler.ListLeaves)
	mux.HandleFunc("GET /api/leaves/{id}", leaveHandler.GetLeaveByID)

	// Lifecycle Action Endpoints
	mux.HandleFunc("POST /api/leaves/{id}/approve", leaveHandler.ApproveLeave)
	mux.HandleFunc("PUT /api/leaves/{id}/approve", leaveHandler.ApproveLeave)
	mux.HandleFunc("POST /api/leaves/{id}/reject", leaveHandler.RejectLeave)
	mux.HandleFunc("PUT /api/leaves/{id}/reject", leaveHandler.RejectLeave)
	mux.HandleFunc("POST /api/leaves/{id}/cancel", leaveHandler.CancelLeave)
	mux.HandleFunc("PUT /api/leaves/{id}/cancel", leaveHandler.CancelLeave)

	// Employee & Manager Relationship Endpoints
	mux.HandleFunc("GET /api/employees", leaveHandler.ListEmployees)
	mux.HandleFunc("GET /api/employees/{id}/leaves", leaveHandler.GetEmployeeLeaves)
	mux.HandleFunc("GET /api/managers/{id}/pending-leaves", leaveHandler.GetManagerPendingLeaves)

	// Friendly welcome/info handler on root
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, `{"message":"Beinex Leave Management API is running. Check /health for system status."}`)
	})

	serverAddr := fmt.Sprintf("0.0.0.0:%s", cfg.Port)
	srv := &http.Server{
		Addr:         serverAddr,
		Handler:      enableCORS(mux, cfg.AllowedOrigins),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 4. Start HTTP server in a separate goroutine
	serverErrCh := make(chan error, 1)
	go func() {
		log.Printf("[server] HTTP server listening on http://localhost%s", serverAddr)
		log.Printf("[server] Health check available at http://localhost%s/health", serverAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
		}
	}()

	// 5. Handle graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrCh:
		log.Fatalf("[server] HTTP server failure: %v", err)
	case sig := <-quit:
		log.Printf("[server] received shutdown signal (%v), terminating gracefully...", sig)
	}

	// Graceful shutdown with 5-second deadline
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("[server] server shutdown error: %v", err)
	}

	log.Println("[server] backend shutdown complete")
}

// enableCORS wraps an http.Handler with standard CORS headers to permit frontend communication.
func enableCORS(next http.Handler, allowed []string) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Determine allowed origin based on configuration
        origin := r.Header.Get("Origin")
        allowedOrigin := "*"
        if len(allowed) > 0 {
            for _, o := range allowed {
                if o == "*" {
                    allowedOrigin = "*"
                    break
                }
                if o == origin {
                    allowedOrigin = origin
                    break
                }
            }
        }
        w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusOK)
            return
        }
        next.ServeHTTP(w, r)
    })
}

}

