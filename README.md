# Geofencing & Real-time Alert System

Go + PostGIS backend, React + Leaflet frontend, WebSocket live alerts.

**Full instructions (setup, Docker, curl examples, architecture): see [SETUP.md](./SETUP.md).**

Quick start:

```bash
docker compose up --build
# frontend: http://localhost:3000     backend: http://localhost:8080
./scripts/demo.sh                     # creates demo data and triggers live alerts
```

- Video walkthrough: https://www.loom.com/share/27a19d25b1644333ac7b842a96157cdc
- Deployed frontend: https://mapup-geofencing-system.vercel.app/?utm_source=chatgpt.com
- Deployed backend:  https://geofence-backend-production.up.railway.app/?utm_source=chatgpt.com
- Docker Hub image:  https://hub.docker.com/r/fkhansherani/geofence-backend
