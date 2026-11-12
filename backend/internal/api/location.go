package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"geofence-backend/internal/geo"

	"github.com/jackc/pgx/v5"
)

type locationRequest struct {
	VehicleID string `json:"vehicle_id"`
	// Pointers so we can tell "missing" apart from a real 0 value.
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	Timestamp string   `json:"timestamp"`
}

// geofenceRef is the short form of a geofence we pass around inside this file.
type geofenceRef struct {
	ID       string
	Name     string
	Category string
}

// alertEvent is exactly the message format the assessment asks for on the WebSocket.
type alertEvent struct {
	EventID   string        `json:"event_id"`
	EventType string        `json:"event_type"`
	Timestamp time.Time     `json:"timestamp"`
	Vehicle   eventVehicle  `json:"vehicle"`
	Geofence  eventGeofence `json:"geofence"`
	Location  eventLocation `json:"location"`
}

type eventVehicle struct {
	VehicleID     string `json:"vehicle_id"`
	VehicleNumber string `json:"vehicle_number"`
	DriverName    string `json:"driver_name"`
}

type eventGeofence struct {
	GeofenceID   string `json:"geofence_id"`
	GeofenceName string `json:"geofence_name"`
	Category     string `json:"category"`
}

type eventLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// POST /vehicles/location
//
// Steps:
//  1. validate the input
//  2. lock the vehicle row so two updates for the same vehicle run one after another
//  3. save the location
//  4. ask PostGIS which geofences contain the point
//  5. compare with the geofences the vehicle was inside before -> entry / exit events
//  6. save the events, check alert rules, and after commit push alerts over WebSocket
func (s *Server) updateLocation(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	var req locationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, start, "request body must be valid JSON")
		return
	}
	if req.VehicleID == "" {
		writeError(w, http.StatusBadRequest, start, "vehicle_id is required")
		return
	}
	if req.Latitude == nil || req.Longitude == nil {
		writeError(w, http.StatusBadRequest, start, "latitude and longitude are required")
		return
	}
	lat, lng := *req.Latitude, *req.Longitude
	if err := geo.ValidateLatLng(lat, lng); err != nil {
		writeError(w, http.StatusBadRequest, start, err.Error())
		return
	}
	recordedAt, err := time.Parse(time.RFC3339, req.Timestamp)
	if err != nil {
		writeError(w, http.StatusBadRequest, start, "timestamp must be ISO 8601, for example 2025-01-15T10:35:00Z")
		return
	}
	recordedAt = recordedAt.UTC()

	tx, err := s.db.Begin(ctx)
	if err != nil {
		serverError(w, start, err)
		return
	}
	defer tx.Rollback(ctx) // does nothing after a successful Commit

	// FOR UPDATE locks this vehicle's row until the transaction ends.
	var vehicle eventVehicle
	vehicle.VehicleID = req.VehicleID
	err = tx.QueryRow(ctx,
		`SELECT vehicle_number, driver_name FROM vehicles WHERE id = $1 FOR UPDATE`,
		req.VehicleID).Scan(&vehicle.VehicleNumber, &vehicle.DriverName)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, start, "vehicle not found")
		return
	}
	if err != nil {
		serverError(w, start, err)
		return
	}

	// If this update is older than one we already have (it arrived late),
	// we still store it in the history but we do not change the current state.
	var latest *time.Time
	if err := tx.QueryRow(ctx,
		`SELECT MAX(recorded_at) FROM vehicle_locations WHERE vehicle_id = $1`,
		req.VehicleID).Scan(&latest); err != nil {
		serverError(w, start, err)
		return
	}
	isStale := latest != nil && recordedAt.Before(*latest)

	if _, err := tx.Exec(ctx,
		`INSERT INTO vehicle_locations (vehicle_id, latitude, longitude, recorded_at) VALUES ($1, $2, $3, $4)`,
		req.VehicleID, lat, lng, recordedAt); err != nil {
		serverError(w, start, err)
		return
	}

	previous, err := loadCurrentGeofences(ctx, tx, req.VehicleID)
	if err != nil {
		serverError(w, start, err)
		return
	}

	current := previous
	var alerts []alertEvent

	if !isStale {
		current, err = findContainingGeofences(ctx, tx, lat, lng)
		if err != nil {
			serverError(w, start, err)
			return
		}

		// Entered = inside now but not before. Exited = inside before but not now.
		for _, g := range current {
			if !hasGeofence(previous, g.ID) {
				alert, err := recordEvent(ctx, tx, vehicle, g, "entry", lat, lng, recordedAt)
				if err != nil {
					serverError(w, start, err)
					return
				}
				if alert != nil {
					alerts = append(alerts, *alert)
				}
			}
		}
		for _, g := range previous {
			if !hasGeofence(current, g.ID) {
				alert, err := recordEvent(ctx, tx, vehicle, g, "exit", lat, lng, recordedAt)
				if err != nil {
					serverError(w, start, err)
					return
				}
				if alert != nil {
					alerts = append(alerts, *alert)
				}
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		serverError(w, start, err)
		return
	}

	// Push alerts in a separate goroutine so the HTTP response is never delayed.
	// This happens after Commit so we never announce something that was rolled back.
	if len(alerts) > 0 {
		go func() {
			for _, alert := range alerts {
				s.hub.Broadcast(alert)
			}
		}()
	}

	inside := make([]map[string]any, 0, len(current))
	for _, g := range current {
		inside = append(inside, map[string]any{
			"geofence_id":   g.ID,
			"geofence_name": g.Name,
			"status":        "inside",
		})
	}

	writeJSON(w, http.StatusOK, start, map[string]any{
		"vehicle_id":        req.VehicleID,
		"location_updated":  true,
		"current_geofences": inside,
		"stale":             isStale,
	})
}

// GET /vehicles/location/{vehicle_id}
func (s *Server) getVehicleLocation(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()
	vehicleID := r.PathValue("vehicle_id")

	var vehicleNumber string
	err := s.db.QueryRow(ctx, `SELECT vehicle_number FROM vehicles WHERE id = $1`, vehicleID).Scan(&vehicleNumber)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, start, "vehicle not found")
		return
	}
	if err != nil {
		serverError(w, start, err)
		return
	}

	// The newest location by the time it was recorded (not by arrival time).
	var currentLocation map[string]any
	var lat, lng float64
	var recordedAt time.Time
	err = s.db.QueryRow(ctx, `
		SELECT latitude, longitude, recorded_at
		FROM vehicle_locations
		WHERE vehicle_id = $1
		ORDER BY recorded_at DESC, id DESC
		LIMIT 1`, vehicleID).Scan(&lat, &lng, &recordedAt)
	switch {
	case err == nil:
		currentLocation = map[string]any{
			"latitude":  lat,
			"longitude": lng,
			"timestamp": recordedAt.UTC(),
		}
	case errors.Is(err, pgx.ErrNoRows):
		currentLocation = nil // vehicle exists but has never sent a location
	default:
		serverError(w, start, err)
		return
	}

	geofences, err := loadCurrentGeofences(ctx, s.db, vehicleID)
	if err != nil {
		serverError(w, start, err)
		return
	}
	inside := make([]map[string]any, 0, len(geofences))
	for _, g := range geofences {
		inside = append(inside, map[string]any{
			"geofence_id":   g.ID,
			"geofence_name": g.Name,
			"category":      g.Category,
		})
	}

	writeJSON(w, http.StatusOK, start, map[string]any{
		"vehicle_id":        vehicleID,
		"vehicle_number":    vehicleNumber,
		"current_location":  currentLocation,
		"current_geofences": inside,
	})
}

// findContainingGeofences is the point-in-polygon check, done by PostGIS.
// ST_Covers is true when the point is inside the polygon OR exactly on its edge.
// The GiST index on "boundary" means PostGIS only tests polygons whose bounding
// box contains the point, so this stays fast with thousands of geofences.
// Note: ST_MakePoint takes (longitude, latitude).
func findContainingGeofences(ctx context.Context, q querier, lat, lng float64) ([]geofenceRef, error) {
	rows, err := q.Query(ctx, `
		SELECT id, name, category
		FROM geofences
		WHERE status = 'active'
		  AND ST_Covers(boundary, ST_SetSRID(ST_MakePoint($1, $2), 4326))
		ORDER BY name`, lng, lat)
	if err != nil {
		return nil, err
	}
	return scanGeofenceRefs(rows)
}

// loadCurrentGeofences returns the geofences the vehicle is inside according to
// the vehicle_geofence_state table (the result of its previous update).
func loadCurrentGeofences(ctx context.Context, q querier, vehicleID string) ([]geofenceRef, error) {
	rows, err := q.Query(ctx, `
		SELECT g.id, g.name, g.category
		FROM vehicle_geofence_state s
		JOIN geofences g ON g.id = s.geofence_id
		WHERE s.vehicle_id = $1
		ORDER BY g.name`, vehicleID)
	if err != nil {
		return nil, err
	}
	return scanGeofenceRefs(rows)
}

func scanGeofenceRefs(rows pgx.Rows) ([]geofenceRef, error) {
	defer rows.Close()

	refs := make([]geofenceRef, 0)
	for rows.Next() {
		var g geofenceRef
		if err := rows.Scan(&g.ID, &g.Name, &g.Category); err != nil {
			return nil, err
		}
		refs = append(refs, g)
	}
	return refs, rows.Err()
}

func hasGeofence(list []geofenceRef, id string) bool {
	for _, g := range list {
		if g.ID == id {
			return true
		}
	}
	return false
}

// recordEvent saves one entry/exit event, updates the vehicle's state and checks
// the alert rules. It returns the alert message when a rule matched, or nil.
func recordEvent(ctx context.Context, tx pgx.Tx, vehicle eventVehicle, g geofenceRef, eventType string, lat, lng float64, recordedAt time.Time) (*alertEvent, error) {
	shouldAlert, err := alertRuleMatches(ctx, tx, g.ID, vehicle.VehicleID, eventType)
	if err != nil {
		return nil, err
	}

	// The violation id and the event id share the same random part, so they are easy to match.
	suffix := newSuffix()

	if _, err := tx.Exec(ctx, `
		INSERT INTO violations (id, vehicle_id, geofence_id, event_type, latitude, longitude, recorded_at, alerted)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		"viol_"+suffix, vehicle.VehicleID, g.ID, eventType, lat, lng, recordedAt, shouldAlert); err != nil {
		return nil, err
	}

	if eventType == "entry" {
		_, err = tx.Exec(ctx, `
			INSERT INTO vehicle_geofence_state (vehicle_id, geofence_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING`, vehicle.VehicleID, g.ID)
	} else {
		_, err = tx.Exec(ctx, `
			DELETE FROM vehicle_geofence_state WHERE vehicle_id = $1 AND geofence_id = $2`,
			vehicle.VehicleID, g.ID)
	}
	if err != nil {
		return nil, err
	}

	if !shouldAlert {
		return nil, nil
	}

	return &alertEvent{
		EventID:   "evt_" + suffix,
		EventType: eventType,
		Timestamp: recordedAt,
		Vehicle:   vehicle,
		Geofence:  eventGeofence{GeofenceID: g.ID, GeofenceName: g.Name, Category: g.Category},
		Location:  eventLocation{Latitude: lat, Longitude: lng},
	}, nil
}

// alertRuleMatches is true when an active rule exists for this geofence that
// applies to this vehicle (or to all vehicles) and to this event type.
func alertRuleMatches(ctx context.Context, q querier, geofenceID, vehicleID, eventType string) (bool, error) {
	var matches bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM alert_rules
			WHERE status = 'active'
			  AND geofence_id = $1
			  AND (vehicle_id IS NULL OR vehicle_id = $2)
			  AND event_type IN ('both', $3::text)
		)`, geofenceID, vehicleID, eventType).Scan(&matches)
	return matches, err
}
