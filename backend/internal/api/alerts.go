package api

import (
	"net/http"
	"time"
)

type configureAlertRequest struct {
	GeofenceID string  `json:"geofence_id"`
	VehicleID  *string `json:"vehicle_id"` // optional: nil means "all vehicles"
	EventType  string  `json:"event_type"`
}

type alertRuleResponse struct {
	AlertID       string    `json:"alert_id"`
	GeofenceID    string    `json:"geofence_id"`
	GeofenceName  string    `json:"geofence_name"`
	VehicleID     *string   `json:"vehicle_id"`
	VehicleNumber *string   `json:"vehicle_number"`
	EventType     string    `json:"event_type"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

var validEventTypes = map[string]bool{"entry": true, "exit": true, "both": true}

// POST /alerts/configure
func (s *Server) configureAlert(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	var req configureAlertRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, start, "request body must be valid JSON")
		return
	}
	if req.GeofenceID == "" {
		writeError(w, http.StatusBadRequest, start, "geofence_id is required")
		return
	}
	if !validEventTypes[req.EventType] {
		writeError(w, http.StatusBadRequest, start, "event_type must be one of: entry, exit, both")
		return
	}
	// An empty string from a form means "all vehicles", same as leaving it out.
	if req.VehicleID != nil && *req.VehicleID == "" {
		req.VehicleID = nil
	}

	var exists bool
	if err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM geofences WHERE id = $1)`, req.GeofenceID).Scan(&exists); err != nil {
		serverError(w, start, err)
		return
	}
	if !exists {
		writeError(w, http.StatusNotFound, start, "geofence not found")
		return
	}
	if req.VehicleID != nil {
		if err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM vehicles WHERE id = $1)`, *req.VehicleID).Scan(&exists); err != nil {
			serverError(w, start, err)
			return
		}
		if !exists {
			writeError(w, http.StatusNotFound, start, "vehicle not found")
			return
		}
	}

	id := newID("alert")
	if _, err := s.db.Exec(ctx, `
		INSERT INTO alert_rules (id, geofence_id, vehicle_id, event_type)
		VALUES ($1, $2, $3, $4)`,
		id, req.GeofenceID, req.VehicleID, req.EventType); err != nil {
		serverError(w, start, err)
		return
	}

	writeJSON(w, http.StatusCreated, start, map[string]any{
		"alert_id":    id,
		"geofence_id": req.GeofenceID,
		"vehicle_id":  req.VehicleID,
		"event_type":  req.EventType,
		"status":      "active",
	})
}

// GET /alerts?geofence_id=...&vehicle_id=...
func (s *Server) listAlerts(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	query := r.URL.Query()

	rows, err := s.db.Query(r.Context(), `
		SELECT a.id, a.geofence_id, g.name, a.vehicle_id, v.vehicle_number, a.event_type, a.status, a.created_at
		FROM alert_rules a
		JOIN geofences g ON g.id = a.geofence_id
		LEFT JOIN vehicles v ON v.id = a.vehicle_id
		WHERE ($1::text = '' OR a.geofence_id = $1::text)
		  AND ($2::text = '' OR a.vehicle_id = $2::text)
		ORDER BY a.created_at DESC`,
		query.Get("geofence_id"), query.Get("vehicle_id"))
	if err != nil {
		serverError(w, start, err)
		return
	}
	defer rows.Close()

	alerts := make([]alertRuleResponse, 0)
	for rows.Next() {
		var a alertRuleResponse
		if err := rows.Scan(&a.AlertID, &a.GeofenceID, &a.GeofenceName, &a.VehicleID, &a.VehicleNumber, &a.EventType, &a.Status, &a.CreatedAt); err != nil {
			serverError(w, start, err)
			return
		}
		alerts = append(alerts, a)
	}
	if err := rows.Err(); err != nil {
		serverError(w, start, err)
		return
	}

	writeJSON(w, http.StatusOK, start, map[string]any{"alerts": alerts})
}

// PATCH /alerts/{alert_id}   body: {"status": "active" | "inactive"}
// (extra endpoint so the UI can switch a rule on and off)
func (s *Server) updateAlertStatus(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var req struct {
		Status string `json:"status"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, start, "request body must be valid JSON")
		return
	}
	if req.Status != "active" && req.Status != "inactive" {
		writeError(w, http.StatusBadRequest, start, "status must be active or inactive")
		return
	}

	alertID := r.PathValue("alert_id")
	tag, err := s.db.Exec(r.Context(), `UPDATE alert_rules SET status = $1 WHERE id = $2`, req.Status, alertID)
	if err != nil {
		serverError(w, start, err)
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, start, "alert rule not found")
		return
	}

	writeJSON(w, http.StatusOK, start, map[string]any{"alert_id": alertID, "status": req.Status})
}

// DELETE /alerts/{alert_id}
// (extra endpoint so the UI can remove a rule)
func (s *Server) deleteAlert(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	alertID := r.PathValue("alert_id")
	tag, err := s.db.Exec(r.Context(), `DELETE FROM alert_rules WHERE id = $1`, alertID)
	if err != nil {
		serverError(w, start, err)
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, start, "alert rule not found")
		return
	}

	writeJSON(w, http.StatusOK, start, map[string]any{"alert_id": alertID, "deleted": true})
}
