#!/usr/bin/env bash
# Creates a restricted zone, a vehicle and an alert rule, then moves the vehicle
# outside -> inside -> outside so you can watch the live alerts arrive in the UI.
#
# Usage: ./scripts/demo.sh [backend-url]      (default http://localhost:8080)

BASE="${1:-http://localhost:8080}"

# Pulls the first "id" value out of a JSON response without needing jq.
get_id() {
  sed -n 's/.*"id":"\([^"]*\)".*/\1/p'
}

now() {
  date -u +%Y-%m-%dT%H:%M:%SZ
}

post() {
  curl -s -X POST "$BASE$1" -H "Content-Type: application/json" -d "$2"
}

echo "1) Creating a restricted zone..."
GEO=$(post /geofences '{
  "name": "MG Road Restricted Zone",
  "description": "No trucks allowed",
  "category": "restricted_zone",
  "coordinates": [
    [12.970, 77.590],
    [12.980, 77.590],
    [12.980, 77.605],
    [12.970, 77.605],
    [12.970, 77.590]
  ]
}')
echo "$GEO"
GEO_ID=$(echo "$GEO" | get_id)

echo
echo "2) Registering a vehicle..."
VEH=$(post /vehicles "{
  \"vehicle_number\": \"KA-01-DEMO-$RANDOM\",
  \"driver_name\": \"Demo Driver\",
  \"vehicle_type\": \"truck\",
  \"phone\": \"+911234567890\"
}")
echo "$VEH"
VEH_ID=$(echo "$VEH" | get_id)

echo
echo "3) Alert rule: tell me when ANY vehicle enters or exits the zone..."
post /alerts/configure "{\"geofence_id\": \"$GEO_ID\", \"event_type\": \"both\"}"

echo
echo "4) Vehicle outside the zone..."
post /vehicles/location "{\"vehicle_id\": \"$VEH_ID\", \"latitude\": 12.950, \"longitude\": 77.570, \"timestamp\": \"$(now)\"}"
sleep 1

echo
echo
echo "5) Vehicle moves INSIDE the zone (entry alert)..."
post /vehicles/location "{\"vehicle_id\": \"$VEH_ID\", \"latitude\": 12.975, \"longitude\": 77.595, \"timestamp\": \"$(now)\"}"
sleep 1

echo
echo
echo "6) Vehicle leaves again (exit alert)..."
post /vehicles/location "{\"vehicle_id\": \"$VEH_ID\", \"latitude\": 12.950, \"longitude\": 77.570, \"timestamp\": \"$(now)\"}"

echo
echo
echo "7) History for this vehicle:"
curl -s "$BASE/violations/history?vehicle_id=$VEH_ID"
echo
