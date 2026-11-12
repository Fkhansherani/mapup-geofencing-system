-- Schema is applied automatically on every start (all statements are idempotent).

CREATE EXTENSION IF NOT EXISTS postgis;

-- Geofences: we keep the original [lat, lng] list as JSON (to send back to the
-- client unchanged) and a PostGIS polygon (to do fast point-in-polygon checks).
CREATE TABLE IF NOT EXISTS geofences (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category    TEXT NOT NULL CHECK (category IN ('delivery_zone', 'restricted_zone', 'toll_zone', 'customer_area')),
    coordinates JSONB NOT NULL,
    boundary    geometry(Polygon, 4326) NOT NULL,
    status      TEXT NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- The GiST index is what makes "which polygons contain this point?" fast.
CREATE INDEX IF NOT EXISTS idx_geofences_boundary ON geofences USING GIST (boundary);
CREATE INDEX IF NOT EXISTS idx_geofences_category ON geofences (category);

CREATE TABLE IF NOT EXISTS vehicles (
    id             TEXT PRIMARY KEY,
    vehicle_number TEXT NOT NULL UNIQUE,
    driver_name    TEXT NOT NULL,
    vehicle_type   TEXT NOT NULL,
    phone          TEXT NOT NULL,
    status         TEXT NOT NULL DEFAULT 'active',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Every location update we ever receive (the movement history).
CREATE TABLE IF NOT EXISTS vehicle_locations (
    id          BIGSERIAL PRIMARY KEY,
    vehicle_id  TEXT NOT NULL REFERENCES vehicles (id) ON DELETE CASCADE,
    latitude    DOUBLE PRECISION NOT NULL,
    longitude   DOUBLE PRECISION NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_vehicle_locations_latest ON vehicle_locations (vehicle_id, recorded_at DESC);

-- Which geofences each vehicle is inside RIGHT NOW.
-- Comparing this with the new location tells us about entry / exit events.
CREATE TABLE IF NOT EXISTS vehicle_geofence_state (
    vehicle_id  TEXT NOT NULL REFERENCES vehicles (id) ON DELETE CASCADE,
    geofence_id TEXT NOT NULL REFERENCES geofences (id) ON DELETE CASCADE,
    entered_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (vehicle_id, geofence_id)
);

-- Alert rules. vehicle_id = NULL means "applies to all vehicles".
CREATE TABLE IF NOT EXISTS alert_rules (
    id          TEXT PRIMARY KEY,
    geofence_id TEXT NOT NULL REFERENCES geofences (id) ON DELETE CASCADE,
    vehicle_id  TEXT REFERENCES vehicles (id) ON DELETE CASCADE,
    event_type  TEXT NOT NULL CHECK (event_type IN ('entry', 'exit', 'both')),
    status      TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_alert_rules_geofence ON alert_rules (geofence_id);

-- Every entry / exit event. "alerted" is true when an alert rule matched and
-- the event was pushed to the live clients.
CREATE TABLE IF NOT EXISTS violations (
    id          TEXT PRIMARY KEY,
    vehicle_id  TEXT NOT NULL REFERENCES vehicles (id) ON DELETE CASCADE,
    geofence_id TEXT NOT NULL REFERENCES geofences (id) ON DELETE CASCADE,
    event_type  TEXT NOT NULL CHECK (event_type IN ('entry', 'exit')),
    latitude    DOUBLE PRECISION NOT NULL,
    longitude   DOUBLE PRECISION NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL,
    alerted     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_violations_time ON violations (recorded_at DESC);
CREATE INDEX IF NOT EXISTS idx_violations_vehicle ON violations (vehicle_id, recorded_at DESC);
CREATE INDEX IF NOT EXISTS idx_violations_geofence ON violations (geofence_id, recorded_at DESC);
