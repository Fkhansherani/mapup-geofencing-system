package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type violationResponse struct {
	ID            string    `json:"id"`
	VehicleID     string    `json:"vehicle_id"`
	VehicleNumber string    `json:"vehicle_number"`
	GeofenceID    string    `json:"geofence_id"`
	GeofenceName  string    `json:"geofence_name"`
	EventType     string    `json:"event_type"`
	Latitude      float64   `json:"latitude"`
	Longitude     float64   `json:"longitude"`
	Timestamp     time.Time `json:"timestamp"`
	Alerted       bool      `json:"alerted"`
}

const (
	defaultLimit = 50
	maxLimit     = 500
)

// GET /violations/history?vehicle_id=&geofence_id=&start_date=&end_date=&limit=&offset=
func (s *Server) violationHistory(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	query := r.URL.Query()

	// Build the WHERE clause step by step. Values are always passed as $1, $2...
	// parameters (never pasted into the SQL), so there is no SQL injection risk.
	conditions := []string{"TRUE"}
	args := []any{}
	addCondition := func(sql string, value any) {
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf(sql, len(args)))
	}

	if v := query.Get("vehicle_id"); v != "" {
		addCondition("vl.vehicle_id = $%d", v)
	}
	if v := query.Get("geofence_id"); v != "" {
		addCondition("vl.geofence_id = $%d", v)
	}
	if v := query.Get("start_date"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, start, "start_date must be ISO 8601, for example 2025-01-01T00:00:00Z")
			return
		}
		addCondition("vl.recorded_at >= $%d", t)
	}
	if v := query.Get("end_date"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, start, "end_date must be ISO 8601, for example 2025-01-31T23:59:59Z")
			return
		}
		addCondition("vl.recorded_at <= $%d", t)
	}

	limit := defaultLimit
	if v := query.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, start, "limit must be a positive number")
			return
		}
		limit = min(n, maxLimit)
	}
	offset := 0
	if v := query.Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			writeError(w, http.StatusBadRequest, start, "offset must be zero or a positive number")
			return
		}
		offset = n
	}

	where := strings.Join(conditions, " AND ")

	var total int
	if err := s.db.QueryRow(r.Context(), `SELECT COUNT(*) FROM violations vl WHERE `+where, args...).Scan(&total); err != nil {
		serverError(w, start, err)
		return
	}

	args = append(args, limit, offset)
	rows, err := s.db.Query(r.Context(), fmt.Sprintf(`
		SELECT vl.id, vl.vehicle_id, v.vehicle_number, vl.geofence_id, g.name,
		       vl.event_type, vl.latitude, vl.longitude, vl.recorded_at, vl.alerted
		FROM violations vl
		JOIN vehicles v ON v.id = vl.vehicle_id
		JOIN geofences g ON g.id = vl.geofence_id
		WHERE %s
		ORDER BY vl.recorded_at DESC, vl.created_at DESC
		LIMIT $%d OFFSET $%d`, where, len(args)-1, len(args)), args...)
	if err != nil {
		serverError(w, start, err)
		return
	}
	defer rows.Close()

	violations := make([]violationResponse, 0)
	for rows.Next() {
		var v violationResponse
		if err := rows.Scan(&v.ID, &v.VehicleID, &v.VehicleNumber, &v.GeofenceID, &v.GeofenceName,
			&v.EventType, &v.Latitude, &v.Longitude, &v.Timestamp, &v.Alerted); err != nil {
			serverError(w, start, err)
			return
		}
		violations = append(violations, v)
	}
	if err := rows.Err(); err != nil {
		serverError(w, start, err)
		return
	}

	writeJSON(w, http.StatusOK, start, map[string]any{
		"violations":  violations,
		"total_count": total,
		"limit":       limit,
		"offset":      offset,
	})
}
