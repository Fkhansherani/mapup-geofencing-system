# Geofencing & Real-time Alert System - Setup Guide

A full-stack geofencing and vehicle tracking system.

- **Backend:** Go (`net/http`, `pgx`, `gorilla/websocket`) + PostgreSQL with PostGIS
- **Frontend:** React (Vite) + Leaflet map
- **Real-time alerts:** WebSocket at `/ws/alerts`
- **Infrastructure:** Docker and Docker Compose

> **Video walkthrough:** `<paste your Loom / unlisted YouTube link here>`

---

## 1. Prerequisites

| Tool | Version | Needed for |
| --- | --- | --- |
| Docker + Docker Compose | recent | running everything with one command |
| Go | 1.24+ | running the backend without Docker |
| Node.js | 20+ | running the frontend without Docker |

## 2. Run everything with Docker Compose (easiest)

```bash
docker compose up --build
```

| Service | URL |
| --- | --- |
| Frontend | http://localhost:3000 |
| Backend API | http://localhost:8080 |
| Health check | http://localhost:8080/health |
| PostgreSQL | localhost:5433 (user `geo`, password `geo`, db `geofence`) |

The database tables are created automatically when the backend starts.
To start from an empty database: `docker compose down -v`.

To fill the system with demo data and trigger live alerts (keep the frontend open on the **Live alerts** tab):

```bash
./scripts/demo.sh
```

## 3. Run locally without Docker (for development)

**Database** (the PostGIS extension is required, so use the PostGIS image):

```bash
docker compose up db
```

**Backend:**

```bash
cd backend
go mod tidy          # first time only: downloads libraries and creates go.sum
export DATABASE_URL="postgres://geo:geo@localhost:5433/geofence?sslmode=disable"
go run .
```

Environment variables (all optional):

| Variable | Default | Meaning |
| --- | --- | --- |
| `PORT` | `8080` | HTTP port |
| `DATABASE_URL` | `postgres://geo:geo@localhost:5432/geofence?sslmode=disable` | PostgreSQL connection string |
| `ALLOWED_ORIGINS` | `*` | CORS: `*` or a comma separated list of frontend URLs |
| `LOCATION_RATE_LIMIT_PER_MIN` | `120` | Rate limit for `POST /vehicles/location` per IP (`0` = off) |

**Frontend:**

```bash
cd frontend
cp .env.example .env     # VITE_API_URL=http://localhost:8080
npm install
npm run dev              # http://localhost:5173
```

**Unit tests:**

```bash
cd backend
go test ./...
```

## 4. API testing guide (curl)

Every response contains `"time_ns"`, the time the handler took in nanoseconds. Errors look like `{"error": "...", "time_ns": "..."}`.

```bash
BASE=http://localhost:8080
```

### 4.1 Create a geofence - `POST /geofences`

Coordinates are `[latitude, longitude]` pairs; the first and last point must be the same.

```bash
curl -s -X POST $BASE/geofences -H "Content-Type: application/json" -d '{
  "name": "Downtown Delivery Zone",
  "description": "Main delivery area for downtown customers",
  "category": "delivery_zone",
  "coordinates": [
    [12.970, 77.590],
    [12.980, 77.590],
    [12.980, 77.605],
    [12.970, 77.605],
    [12.970, 77.590]
  ]
}'
```

Categories: `delivery_zone`, `restricted_zone`, `toll_zone`, `customer_area`.

### 4.2 List geofences - `GET /geofences`

```bash
curl -s $BASE/geofences
curl -s "$BASE/geofences?category=restricted_zone"
```

### 4.3 Register a vehicle - `POST /vehicles`

```bash
curl -s -X POST $BASE/vehicles -H "Content-Type: application/json" -d '{
  "vehicle_number": "KA-01-AB-1234",
  "driver_name": "John Doe",
  "vehicle_type": "truck",
  "phone": "+1234567890"
}'
```

A duplicate `vehicle_number` returns `409`.

### 4.4 List vehicles - `GET /vehicles`

```bash
curl -s $BASE/vehicles
```

### 4.5 Configure an alert rule - `POST /alerts/configure`

`vehicle_id` is optional (leave it out to watch all vehicles). `event_type` is `entry`, `exit` or `both`.

```bash
curl -s -X POST $BASE/alerts/configure -H "Content-Type: application/json" -d '{
  "geofence_id": "geo_xxxxxxxxxxxx",
  "vehicle_id": "veh_xxxxxxxxxxxx",
  "event_type": "entry"
}'
```

> **Important:** live alerts are only sent when a matching alert rule exists. Entry/exit events are always saved in the history, with or without a rule.

### 4.6 List alert rules - `GET /alerts`

```bash
curl -s $BASE/alerts
curl -s "$BASE/alerts?geofence_id=geo_xxxxxxxxxxxx"
curl -s "$BASE/alerts?vehicle_id=veh_xxxxxxxxxxxx"
```

Extra endpoints (not part of the assessment list) used by the UI to manage rules:

```bash
curl -s -X PATCH  $BASE/alerts/alert_xxxxxxxxxxxx -H "Content-Type: application/json" -d '{"status": "inactive"}'
curl -s -X DELETE $BASE/alerts/alert_xxxxxxxxxxxx
```

### 4.7 Update a vehicle location - `POST /vehicles/location`

```bash
curl -s -X POST $BASE/vehicles/location -H "Content-Type: application/json" -d '{
  "vehicle_id": "veh_xxxxxxxxxxxx",
  "latitude": 12.975,
  "longitude": 77.595,
  "timestamp": "2025-01-15T10:35:00Z"
}'
```

The response lists every geofence the vehicle is currently inside. If the vehicle just entered or left a geofence, the event is saved and (when a rule matches) pushed over the WebSocket.

### 4.8 Get the current location of a vehicle - `GET /vehicles/location/{vehicle_id}`

```bash
curl -s $BASE/vehicles/location/veh_xxxxxxxxxxxx
```

### 4.9 Violation history - `GET /violations/history`

Query parameters: `vehicle_id`, `geofence_id`, `start_date`, `end_date` (ISO 8601), `limit` (default 50, max 500) and `offset` (for pagination).

```bash
curl -s "$BASE/violations/history?vehicle_id=veh_xxxxxxxxxxxx&limit=100"
curl -s "$BASE/violations/history?start_date=2025-01-01T00:00:00Z&end_date=2025-12-31T23:59:59Z&limit=20&offset=20"
```

### 4.10 Watch live alerts from the terminal

With [websocat](https://github.com/vi/websocat) or `npx wscat`:

```bash
websocat ws://localhost:8080/ws/alerts
# or
npx wscat -c ws://localhost:8080/ws/alerts
```

Each alert looks like:

```json
{
  "event_id": "evt_a1b2c3d4e5f6",
  "event_type": "entry",
  "timestamp": "2025-01-15T10:35:00Z",
  "vehicle": { "vehicle_id": "veh_...", "vehicle_number": "KA-01-AB-1234", "driver_name": "John Doe" },
  "geofence": { "geofence_id": "geo_...", "geofence_name": "Downtown Delivery Zone", "category": "delivery_zone" },
  "location": { "latitude": 12.975, "longitude": 77.595 }
}
```

## 5. Frontend usage guide

| Tab | What you can do |
| --- | --- |
| **Geofences** | Click on the map to add polygon corners (or type `lat, lng` lines), choose a name and category, press *Create*. Filter the list by category. |
| **Vehicles** | Register a vehicle. The table shows each vehicle's last location and which geofences it is inside. |
| **Update location** | Choose a vehicle, click on the map (or type coordinates), pick a time and send. The result shows the geofences the vehicle is inside. |
| **Alert rules** | Pick a geofence, optionally a vehicle, and entry / exit / both. Rules can be paused, resumed and deleted. |
| **Live alerts** | Real-time feed of alerts. Toast messages also appear on every tab, and restricted-zone alerts are shown in red. The badge in the header shows the connection state and reconnects automatically. |
| **History** | Past entry/exit events with filters (vehicle, geofence, date range) and pagination. |

**Quick demo flow:** create a `restricted_zone` -> register a vehicle -> create an alert rule for that zone -> on *Update location* click outside the zone and send, then click inside the zone and send -> watch the toast and the *Live alerts* tab -> open *History*.

## 6. Architecture overview

```
 React (Vite + Leaflet)                      Go backend                          PostgreSQL + PostGIS
 ----------------------                      ----------                          --------------------
 REST calls  ───────────────────────────►    net/http handlers  ───────────────►  geofences (GiST index)
 WebSocket   ◄───────────────────────────    WebSocket hub      ◄─ after commit   vehicles, vehicle_locations
                                                                                  vehicle_geofence_state
                                                                                  alert_rules, violations
```

**What happens on `POST /vehicles/location`:**

1. Validate the input and open a transaction. The vehicle row is locked (`FOR UPDATE`) so two updates for the same vehicle cannot race.
2. Save the location (movement history).
3. PostGIS finds the geofences containing the point: `ST_Covers(boundary, ST_MakePoint(lng, lat))`. The GiST index on `boundary` keeps this fast.
4. Compare with `vehicle_geofence_state` (where the vehicle was after the previous update): *inside now but not before* = **entry**, *inside before but not now* = **exit**.
5. For each event: check `alert_rules`, save a row in `violations` (`alerted = true/false`) and update the state table.
6. Commit, then broadcast the matching alerts to all WebSocket clients **in a goroutine**, so the HTTP response is never delayed.

**Design decisions**

- **PostGIS for point-in-polygon** instead of Go code: the database index avoids testing every polygon, so it scales to many geofences.
- **State table** for entry/exit: finding the previous state is one indexed lookup, not a scan of the whole location history.
- **Events are stored always, alerts are sent only when a rule matches.** The history is complete even for vehicles without rules.
- **Late (out-of-order) updates** are saved in the history but do not change the current state.
- **Invalid polygons** (for example edges that cross) are rejected with `ST_IsValid`.
- **Rate limiting** (120 requests/minute/IP by default) on `POST /vehicles/location`.
- Coordinates are `[latitude, longitude]` in the API; PostGIS needs `(longitude latitude)`, which is converted in one place: `geo.ToWKT`.

**Project structure**

```
backend/
  main.go                  start-up, config, graceful shutdown
  internal/db/             connection pool + schema.sql
  internal/geo/            validation + WKT conversion (unit tested)
  internal/ws/             WebSocket hub
  internal/api/            handlers: geofences, vehicles, location, alerts, violations
frontend/src/
  App.jsx                  tabs, shared data, WebSocket connection
  api.js                   all backend calls
  hooks/useAlertSocket.js  WebSocket with auto-reconnect
  components/              MapView, Toasts, Badge
  pages/                   one file per tab
```

## 7. Docker Hub

```bash
docker login
docker build -t <your-dockerhub-username>/geofence-backend:latest ./backend
docker push <your-dockerhub-username>/geofence-backend:latest
```

## 8. Deployment

**Backend** (any host that runs Docker, e.g. Render, Railway, Fly.io):

1. Create a PostgreSQL database **with PostGIS support** (for example Neon, or Postgres on your host). The backend runs `CREATE EXTENSION IF NOT EXISTS postgis` on start.
2. Deploy the Docker image (or the `backend` folder with its Dockerfile).
3. Set environment variables: `DATABASE_URL` (use `sslmode=require` for cloud databases), `ALLOWED_ORIGINS=https://<your-frontend-domain>`.
4. Check `https://<backend-domain>/health`.

**Frontend** (Vercel / Netlify / Cloudflare Pages):

- Root directory: `frontend`, build command: `npm run build`, output directory: `dist`.
- Environment variable: `VITE_API_URL=https://<backend-domain>` (set it **before** building; Vite bakes it into the bundle).
- `vercel.json` and `public/_redirects` are included for single-page-app routing.
- The WebSocket URL is derived automatically (`https://` becomes `wss://`).

## 9. Known limitations / next steps

- No authentication (API keys or JWT would be the next step) and the WebSocket accepts any origin.
- The vehicles page loads the location of each vehicle with one request per vehicle; a bulk endpoint would be better for thousands of vehicles.
- The hub works inside one server instance. With several instances, alerts should be distributed through Redis pub/sub or PostgreSQL `LISTEN/NOTIFY`.
- `vehicle_locations` grows forever; it should be partitioned by time or cleaned up.
- Only the geo helpers have unit tests; handlers need integration tests against a real PostGIS database.
- No geofence edit / delete endpoints (they were not part of the assessment).
