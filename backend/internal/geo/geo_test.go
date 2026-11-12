package geo

import "testing"

var validSquare = [][]float64{
	{37.7749, -122.4194},
	{37.7849, -122.4194},
	{37.7849, -122.4094},
	{37.7749, -122.4094},
	{37.7749, -122.4194},
}

func TestValidatePolygon(t *testing.T) {
	tests := []struct {
		name    string
		coords  [][]float64
		wantErr bool
	}{
		{"valid square", validSquare, false},
		{"too few points", [][]float64{{1, 1}, {2, 2}, {1, 1}}, true},
		{"not closed", [][]float64{{1, 1}, {1, 2}, {2, 2}, {2, 1}}, true},
		{"latitude too big", [][]float64{{91, 1}, {1, 2}, {2, 2}, {91, 1}}, true},
		{"longitude too small", [][]float64{{1, -181}, {1, 2}, {2, 2}, {1, -181}}, true},
		{"point with 3 values", [][]float64{{1, 1, 1}, {1, 2}, {2, 2}, {1, 1, 1}}, true},
		{"only two unique points", [][]float64{{1, 1}, {2, 2}, {1, 1}, {1, 1}}, true},
		{"nil coordinates", nil, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePolygon(tc.coords)
			if (err != nil) != tc.wantErr {
				t.Fatalf("got error %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestValidateLatLng(t *testing.T) {
	tests := []struct {
		lat, lng float64
		wantErr  bool
	}{
		{0, 0, false},
		{90, 180, false},
		{-90, -180, false},
		{90.0001, 0, true},
		{0, 180.0001, true},
		{-91, 10, true},
	}

	for _, tc := range tests {
		err := ValidateLatLng(tc.lat, tc.lng)
		if (err != nil) != tc.wantErr {
			t.Errorf("ValidateLatLng(%v, %v) error = %v, wantErr %v", tc.lat, tc.lng, err, tc.wantErr)
		}
	}
}

// PostGIS wants longitude first, so ToWKT must swap the order.
func TestToWKTSwapsLatLng(t *testing.T) {
	got := ToWKT([][]float64{{12.5, 77.25}, {12.5, 77.5}, {12.75, 77.5}, {12.5, 77.25}})
	want := "POLYGON((77.25 12.5, 77.5 12.5, 77.5 12.75, 77.25 12.5))"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestValidCategories(t *testing.T) {
	for _, c := range []string{"delivery_zone", "restricted_zone", "toll_zone", "customer_area"} {
		if !ValidCategories[c] {
			t.Errorf("%s should be a valid category", c)
		}
	}
	if ValidCategories["parking_zone"] {
		t.Error("parking_zone should not be valid")
	}
}
