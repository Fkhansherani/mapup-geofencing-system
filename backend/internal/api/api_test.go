package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"geofence-backend/internal/ws"
)

// These tests only cover requests that are rejected BEFORE the database is used,
// so the server can be created without a database connection.
func newTestServer() http.Handler {
	server := NewServer(nil, ws.NewHub(), Options{AllowedOrigins: "*", LocationRateLimitPerMin: 0})
	return server.Routes()
}

func send(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestBadRequestsAreRejected(t *testing.T) {
	handler := newTestServer()

	validSquare := `[[1,1],[1,2],[2,2],[2,1],[1,1]]`

	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{"geofence: invalid json", "POST", "/geofences", `{not json`},
		{"geofence: missing name", "POST", "/geofences", `{"category":"toll_zone","coordinates":` + validSquare + `}`},
		{"geofence: bad category", "POST", "/geofences", `{"name":"A","category":"parking","coordinates":` + validSquare + `}`},
		{"geofence: polygon not closed", "POST", "/geofences", `{"name":"A","category":"toll_zone","coordinates":[[1,1],[1,2],[2,2],[2,1]]}`},
		{"geofence: latitude out of range", "POST", "/geofences", `{"name":"A","category":"toll_zone","coordinates":[[95,1],[1,2],[2,2],[95,1]]}`},
		{"geofences list: unknown category", "GET", "/geofences?category=nope", ``},

		{"vehicle: missing phone", "POST", "/vehicles", `{"vehicle_number":"KA-01","driver_name":"A","vehicle_type":"car"}`},
		{"vehicle: blank number", "POST", "/vehicles", `{"vehicle_number":"  ","driver_name":"A","vehicle_type":"car","phone":"1"}`},

		{"location: missing vehicle_id", "POST", "/vehicles/location", `{"latitude":1,"longitude":1,"timestamp":"2025-01-15T10:35:00Z"}`},
		{"location: missing latitude", "POST", "/vehicles/location", `{"vehicle_id":"veh_1","longitude":1,"timestamp":"2025-01-15T10:35:00Z"}`},
		{"location: latitude too large", "POST", "/vehicles/location", `{"vehicle_id":"veh_1","latitude":100,"longitude":1,"timestamp":"2025-01-15T10:35:00Z"}`},
		{"location: bad timestamp", "POST", "/vehicles/location", `{"vehicle_id":"veh_1","latitude":1,"longitude":1,"timestamp":"yesterday"}`},

		{"alert: missing geofence_id", "POST", "/alerts/configure", `{"event_type":"entry"}`},
		{"alert: bad event_type", "POST", "/alerts/configure", `{"geofence_id":"geo_1","event_type":"leave"}`},
		{"alert patch: bad status", "PATCH", "/alerts/alert_1", `{"status":"maybe"}`},

		{"history: bad start_date", "GET", "/violations/history?start_date=today", ``},
		{"history: bad end_date", "GET", "/violations/history?end_date=2025-13-45", ``},
		{"history: limit zero", "GET", "/violations/history?limit=0", ``},
		{"history: limit not a number", "GET", "/violations/history?limit=abc", ``},
		{"history: negative offset", "GET", "/violations/history?offset=-5", ``},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			response := send(handler, tc.method, tc.path, tc.body)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body: %s)", response.Code, response.Body.String())
			}

			var body map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("response is not JSON: %v", err)
			}
			if message, _ := body["error"].(string); message == "" {
				t.Errorf("expected an error message, got %v", body)
			}
		})
	}
}

// The assessment requires "time_ns" on every response, as a string of digits.
func TestResponsesContainTimeNs(t *testing.T) {
	response := send(newTestServer(), "POST", "/vehicles", `{}`)

	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}

	timeNs, ok := body["time_ns"].(string)
	if !ok {
		t.Fatalf("time_ns should be a string, got %T", body["time_ns"])
	}
	if _, err := strconv.ParseInt(timeNs, 10, 64); err != nil {
		t.Errorf("time_ns %q is not a number", timeNs)
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := newRateLimiter(3)

	for i := 1; i <= 3; i++ {
		if !limiter.allow("1.1.1.1") {
			t.Fatalf("request %d should be allowed", i)
		}
	}
	if limiter.allow("1.1.1.1") {
		t.Error("4th request from the same IP should be blocked")
	}
	if !limiter.allow("2.2.2.2") {
		t.Error("another IP should have its own counter")
	}
}

func TestRateLimiterReturns429(t *testing.T) {
	limiter := newRateLimiter(1)
	handler := limiter.wrap(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	first := httptest.NewRecorder()
	handler(first, httptest.NewRequest("POST", "/", nil))
	if first.Code != http.StatusOK {
		t.Fatalf("first request status = %d, want 200", first.Code)
	}

	second := httptest.NewRecorder()
	handler(second, httptest.NewRequest("POST", "/", nil))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want 429", second.Code)
	}
}

func TestCORSPreflight(t *testing.T) {
	response := send(newTestServer(), "OPTIONS", "/geofences", "")

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", response.Code)
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("missing Access-Control-Allow-Origin header")
	}
}

func TestNewIDHasPrefix(t *testing.T) {
	id := newID("geo")
	if !strings.HasPrefix(id, "geo_") || len(id) != len("geo_")+12 {
		t.Errorf("unexpected id %q", id)
	}
	if newID("geo") == id {
		t.Error("two ids should not be equal")
	}
}

// Test the health endpoint without requiring a database connection.

func TestHealthEndpoint(t *testing.T) {
	response := send(newTestServer(), "GET", "/health", "")

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 (body: %s)", response.Code, response.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	if message, _ := body["error"].(string); message == "" {
		t.Errorf("expected database error message, got %v", body)
	}

	timeNs, ok := body["time_ns"].(string)
	if !ok {
		t.Fatalf("time_ns should be a string, got %T", body["time_ns"])
	}

	if _, err := strconv.ParseInt(timeNs, 10, 64); err != nil {
		t.Errorf("time_ns %q is not numeric", timeNs)
	}
}