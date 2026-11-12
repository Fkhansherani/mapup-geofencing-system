package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"geofence-backend/internal/geo"
)

type createGeofenceRequest struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Coordinates [][]float64 `json:"coordinates"`
	Category    string      `json:"category"`
}

type geofenceResponse struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Coordinates [][]float64 `json:"coordinates"`
	Category    string      `json:"category"`
	CreatedAt   time.Time   `json:"created_at"`
}

// POST /geofences
func (s *Server) createGeofence(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var req createGeofenceRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, start, "request body must be valid JSON")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, start, "name is required")
		return
	}
	if !geo.ValidCategories[req.Category] {
		writeError(w, http.StatusBadRequest, start, "category must be one of: delivery_zone, restricted_zone, toll_zone, customer_area")
		return
	}
	if err := geo.ValidatePolygon(req.Coordinates); err != nil {
		writeError(w, http.StatusBadRequest, start, err.Error())
		return
	}

	coordsJSON, err := json.Marshal(req.Coordinates)
	if err != nil {
		serverError(w, start, err)
		return
	}

	// The WHERE ST_IsValid(...) part rejects broken shapes such as a
	// self-intersecting "bow-tie" polygon. In that case no row is inserted.
	id := newID("geo")
	tag, err := s.db.Exec(r.Context(), `
		INSERT INTO geofences (id, name, description, category, coordinates, boundary)
		SELECT $1::text, $2::text, $3::text, $4::text, $5::jsonb, ST_GeomFromText($6::text, 4326)
		WHERE ST_IsValid(ST_GeomFromText($6::text, 4326))`,
		id, req.Name, strings.TrimSpace(req.Description), req.Category, string(coordsJSON), geo.ToWKT(req.Coordinates))
	if err != nil {
		serverError(w, start, err)
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusBadRequest, start, "polygon is invalid (for example its edges cross each other)")
		return
	}

	writeJSON(w, http.StatusCreated, start, map[string]any{
		"id":     id,
		"name":   req.Name,
		"status": "active",
	})
}

// GET /geofences?category=...
func (s *Server) listGeofences(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	category := r.URL.Query().Get("category")
	if category != "" && !geo.ValidCategories[category] {
		writeError(w, http.StatusBadRequest, start, "unknown category filter")
		return
	}

	rows, err := s.db.Query(r.Context(), `
		SELECT id, name, description, coordinates, category, created_at
		FROM geofences
		WHERE ($1::text = '' OR category = $1::text)
		ORDER BY created_at DESC`, category)
	if err != nil {
		serverError(w, start, err)
		return
	}
	defer rows.Close()

	geofences := make([]geofenceResponse, 0)
	for rows.Next() {
		var g geofenceResponse
		var coordsJSON []byte
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &coordsJSON, &g.Category, &g.CreatedAt); err != nil {
			serverError(w, start, err)
			return
		}
		if err := json.Unmarshal(coordsJSON, &g.Coordinates); err != nil {
			serverError(w, start, err)
			return
		}
		geofences = append(geofences, g)
	}
	if err := rows.Err(); err != nil {
		serverError(w, start, err)
		return
	}

	writeJSON(w, http.StatusOK, start, map[string]any{"geofences": geofences})
}
