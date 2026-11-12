// Package api contains the HTTP handlers of the geofencing service.
package api

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"geofence-backend/internal/ws"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Options holds the settings that come from environment variables.
type Options struct {
	AllowedOrigins          string // "*" or a comma separated list of frontend URLs
	LocationRateLimitPerMin int    // requests per minute per IP on POST /vehicles/location
}

// Server holds everything the handlers need.
type Server struct {
	db              *pgxpool.Pool
	hub             *ws.Hub
	opts            Options
	locationLimiter *rateLimiter
}

func NewServer(db *pgxpool.Pool, hub *ws.Hub, opts Options) *Server {
	return &Server{
		db:              db,
		hub:             hub,
		opts:            opts,
		locationLimiter: newRateLimiter(opts.LocationRateLimitPerMin),
	}
}

// querier is anything that can run queries: a connection pool or a transaction.
// Helper functions accept this so they work inside and outside transactions.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Routes registers all endpoints. Go 1.22+ lets us write "METHOD /path" directly.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", s.health)

	mux.HandleFunc("POST /geofences", s.createGeofence)
	mux.HandleFunc("GET /geofences", s.listGeofences)

	mux.HandleFunc("POST /vehicles", s.createVehicle)
	mux.HandleFunc("GET /vehicles", s.listVehicles)
	mux.HandleFunc("POST /vehicles/location", s.locationLimiter.wrap(s.updateLocation))
	mux.HandleFunc("GET /vehicles/location/{vehicle_id}", s.getVehicleLocation)

	mux.HandleFunc("POST /alerts/configure", s.configureAlert)
	mux.HandleFunc("GET /alerts", s.listAlerts)
	// Extra endpoints (not in the assessment list) so the UI can manage rules.
	mux.HandleFunc("PATCH /alerts/{alert_id}", s.updateAlertStatus)
	mux.HandleFunc("DELETE /alerts/{alert_id}", s.deleteAlert)

	mux.HandleFunc("GET /violations/history", s.violationHistory)

	mux.HandleFunc("GET /ws/alerts", s.hub.ServeWS)

	return s.cors(logRequests(mux))
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if err := s.db.Ping(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, start, "database not reachable")
		return
	}
	writeJSON(w, http.StatusOK, start, map[string]any{"status": "ok"})
}

// logRequests prints one line per request. We do not wrap the ResponseWriter
// here because that would break WebSocket upgrades.
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		begin := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.RequestURI(), time.Since(begin))
	})
}

// cors lets the browser frontend (on another domain) call this API.
func (s *Server) cors(next http.Handler) http.Handler {
	allowAll := strings.TrimSpace(s.opts.AllowedOrigins) == "*"
	allowed := make(map[string]bool)
	for _, origin := range strings.Split(s.opts.AllowedOrigins, ",") {
		allowed[strings.TrimSpace(origin)] = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if allowAll {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else if allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Browsers send an OPTIONS "preflight" request before POST/PATCH/DELETE.
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
