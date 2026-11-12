// Package geo holds the small, pure helpers we use for geofence data:
// validation and converting coordinates into the format PostGIS understands.
// It has no database code, so it is easy to unit test.
package geo

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ValidCategories lists the geofence categories allowed by the assessment.
var ValidCategories = map[string]bool{
	"delivery_zone":   true,
	"restricted_zone": true,
	"toll_zone":       true,
	"customer_area":   true,
}

// ValidateLatLng checks that a latitude / longitude pair is in the legal range.
func ValidateLatLng(lat, lng float64) error {
	if lat < -90 || lat > 90 {
		return errors.New("latitude must be between -90 and 90")
	}
	if lng < -180 || lng > 180 {
		return errors.New("longitude must be between -180 and 180")
	}
	return nil
}

// ValidatePolygon checks the rules from the assessment:
//   - every point is a [latitude, longitude] pair inside the legal range
//   - at least 4 points (3 different points + 1 closing point)
//   - the first and the last point are identical (closed polygon)
func ValidatePolygon(coords [][]float64) error {
	if len(coords) < 4 {
		return errors.New("coordinates need at least 4 points (3 unique + 1 closing point)")
	}

	for i, point := range coords {
		if len(point) != 2 {
			return fmt.Errorf("point %d must be a [latitude, longitude] pair", i+1)
		}
		if err := ValidateLatLng(point[0], point[1]); err != nil {
			return fmt.Errorf("point %d: %w", i+1, err)
		}
	}

	first := coords[0]
	last := coords[len(coords)-1]
	if first[0] != last[0] || first[1] != last[1] {
		return errors.New("polygon must be closed: first and last point must be identical")
	}

	// A polygon such as A, B, A, A has 4 points but is only a line.
	// Count the different points to catch that.
	unique := make(map[[2]float64]bool)
	for _, point := range coords {
		unique[[2]float64{point[0], point[1]}] = true
	}
	if len(unique) < 3 {
		return errors.New("polygon needs at least 3 different points")
	}

	return nil
}

// ToWKT converts our [lat, lng] points into a WKT polygon string.
// IMPORTANT: PostGIS expects (longitude latitude) order, which is the
// opposite of what the API receives, so we swap the values here.
func ToWKT(coords [][]float64) string {
	parts := make([]string, 0, len(coords))
	for _, point := range coords {
		lat := strconv.FormatFloat(point[0], 'f', -1, 64)
		lng := strconv.FormatFloat(point[1], 'f', -1, 64)
		parts = append(parts, lng+" "+lat)
	}
	return "POLYGON((" + strings.Join(parts, ", ") + "))"
}
