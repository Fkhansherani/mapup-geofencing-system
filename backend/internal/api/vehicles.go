package api

import (
	"net/http"
	"strings"
	"time"
)

type createVehicleRequest struct {
	VehicleNumber string `json:"vehicle_number"`
	DriverName    string `json:"driver_name"`
	VehicleType   string `json:"vehicle_type"`
	Phone         string `json:"phone"`
}

type vehicleResponse struct {
	ID            string    `json:"id"`
	VehicleNumber string    `json:"vehicle_number"`
	DriverName    string    `json:"driver_name"`
	VehicleType   string    `json:"vehicle_type"`
	Phone         string    `json:"phone"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

// POST /vehicles
func (s *Server) createVehicle(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var req createVehicleRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, start, "request body must be valid JSON")
		return
	}

	req.VehicleNumber = strings.TrimSpace(req.VehicleNumber)
	req.DriverName = strings.TrimSpace(req.DriverName)
	req.VehicleType = strings.TrimSpace(req.VehicleType)
	req.Phone = strings.TrimSpace(req.Phone)

	if req.VehicleNumber == "" || req.DriverName == "" || req.VehicleType == "" || req.Phone == "" {
		writeError(w, http.StatusBadRequest, start, "vehicle_number, driver_name, vehicle_type and phone are all required")
		return
	}

	id := newID("veh")
	_, err := s.db.Exec(r.Context(), `
		INSERT INTO vehicles (id, vehicle_number, driver_name, vehicle_type, phone)
		VALUES ($1, $2, $3, $4, $5)`,
		id, req.VehicleNumber, req.DriverName, req.VehicleType, req.Phone)
	if err != nil {
		if isPgCode(err, "23505") { // unique violation on vehicle_number
			writeError(w, http.StatusConflict, start, "a vehicle with this number already exists")
			return
		}
		serverError(w, start, err)
		return
	}

	writeJSON(w, http.StatusCreated, start, map[string]any{
		"id":             id,
		"vehicle_number": req.VehicleNumber,
		"status":         "active",
	})
}

// GET /vehicles
func (s *Server) listVehicles(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	rows, err := s.db.Query(r.Context(), `
		SELECT id, vehicle_number, driver_name, vehicle_type, phone, status, created_at
		FROM vehicles
		ORDER BY created_at DESC`)
	if err != nil {
		serverError(w, start, err)
		return
	}
	defer rows.Close()

	vehicles := make([]vehicleResponse, 0)
	for rows.Next() {
		var v vehicleResponse
		if err := rows.Scan(&v.ID, &v.VehicleNumber, &v.DriverName, &v.VehicleType, &v.Phone, &v.Status, &v.CreatedAt); err != nil {
			serverError(w, start, err)
			return
		}
		vehicles = append(vehicles, v)
	}
	if err := rows.Err(); err != nil {
		serverError(w, start, err)
		return
	}

	writeJSON(w, http.StatusOK, start, map[string]any{"vehicles": vehicles})
}
