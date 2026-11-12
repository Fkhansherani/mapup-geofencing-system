package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// writeJSON sends a JSON response and adds "time_ns" (the time this handler
// took, in nanoseconds, as a string) to every response as the assessment requires.
func writeJSON(w http.ResponseWriter, status int, start time.Time, body map[string]any) {
	body["time_ns"] = strconv.FormatInt(time.Since(start).Nanoseconds(), 10)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("write response: %v", err)
	}
}

// writeError sends {"error": "...", "time_ns": "..."}.
func writeError(w http.ResponseWriter, status int, start time.Time, message string) {
	writeJSON(w, status, start, map[string]any{"error": message})
}

// serverError logs the real error but only shows a generic message to the client.
func serverError(w http.ResponseWriter, start time.Time, err error) {
	log.Printf("internal error: %v", err)
	writeError(w, http.StatusInternalServerError, start, "internal server error")
}

// decodeJSON reads the request body into dst. The body is limited to 1 MB.
func decodeJSON(r *http.Request, dst any) error {
	return json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(dst)
}

// newSuffix returns 12 random hex characters, used to build ids like "geo_a1b2c3d4e5f6".
func newSuffix() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand failing means the OS is broken; there is nothing sensible to do.
		panic(err)
	}
	return hex.EncodeToString(b)
}

func newID(prefix string) string {
	return prefix + "_" + newSuffix()
}

// isPgCode checks if err is a PostgreSQL error with the given code.
// 23505 = unique violation, 23503 = foreign key violation.
func isPgCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}
