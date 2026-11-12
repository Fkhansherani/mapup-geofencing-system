import { useCallback, useEffect, useMemo, useState } from "react";
import { api } from "./api";
import { useAlertSocket } from "./hooks/useAlertSocket";
import { useToast } from "./components/Toasts";
import GeofencesPage from "./pages/GeofencesPage";
import VehiclesPage from "./pages/VehiclesPage";
import LocationPage from "./pages/LocationPage";
import AlertsPage from "./pages/AlertsPage";
import LiveAlertsPage from "./pages/LiveAlertsPage";
import HistoryPage from "./pages/HistoryPage";

const TABS = [
  { id: "geofences", label: "Geofences" },
  { id: "vehicles", label: "Vehicles" },
  { id: "location", label: "Update location" },
  { id: "alerts", label: "Alert rules" },
  { id: "live", label: "Live alerts" },
  { id: "history", label: "History" },
];

const STATUS_LABEL = { connected: "Live", connecting: "Connecting...", disconnected: "Offline" };

export default function App() {
  const notify = useToast();

  const [tab, setTab] = useState("geofences");

  // Data shared by several pages lives here, in the top component.
  const [geofences, setGeofences] = useState([]);
  const [vehicles, setVehicles] = useState([]);
  const [locations, setLocations] = useState({}); // vehicle id -> response of GET /vehicles/location/{id}
  const [vehiclesLoading, setVehiclesLoading] = useState(true);

  const [liveAlerts, setLiveAlerts] = useState([]);
  const [unread, setUnread] = useState(0);

  const loadGeofences = useCallback(async () => {
    try {
      const data = await api.getGeofences();
      setGeofences(data.geofences);
    } catch (error) {
      notify(error.message, "error");
    }
  }, [notify]);

  // Loads the vehicles and then the current location of every vehicle.
  const loadVehicles = useCallback(async () => {
    try {
      const data = await api.getVehicles();
      setVehicles(data.vehicles);

      const entries = await Promise.all(
        data.vehicles.map((vehicle) =>
          api
            .getVehicleLocation(vehicle.id)
            .then((info) => [vehicle.id, info])
            .catch(() => [vehicle.id, null])
        )
      );
      setLocations(Object.fromEntries(entries));
    } catch (error) {
      notify(error.message, "error");
    } finally {
      setVehiclesLoading(false);
    }
  }, [notify]);

  useEffect(() => {
    loadGeofences();
    loadVehicles();
  }, [loadGeofences, loadVehicles]);

  // Called for every message that arrives over the WebSocket.
  const handleAlert = useCallback(
    (alert) => {
      setLiveAlerts((current) => [alert, ...current].slice(0, 100)); // keep the latest 100
      setUnread((count) => count + 1);

      const verb = alert.event_type === "entry" ? "entered" : "exited";
      const isRestricted = alert.geofence.category === "restricted_zone";
      notify(
        `${alert.vehicle.vehicle_number} ${verb} ${alert.geofence.geofence_name}`,
        isRestricted ? "danger" : "info"
      );

      loadVehicles(); // the vehicle moved, so refresh the markers
    },
    [notify, loadVehicles]
  );

  const socketStatus = useAlertSocket(handleAlert);

  // Opening the "Live alerts" tab marks everything as read.
  useEffect(() => {
    if (tab === "live") setUnread(0);
  }, [tab, liveAlerts]);

  // The map needs a simple list of vehicles that have a known position.
  const vehicleMarkers = useMemo(
    () =>
      vehicles
        .filter((vehicle) => locations[vehicle.id]?.current_location)
        .map((vehicle) => {
          const info = locations[vehicle.id];
          return {
            id: vehicle.id,
            number: vehicle.vehicle_number,
            driver: vehicle.driver_name,
            position: [info.current_location.latitude, info.current_location.longitude],
            geofences: info.current_geofences.map((g) => g.geofence_name),
          };
        }),
    [vehicles, locations]
  );

  return (
    <div className="app">
      <header className="topbar">
        <h1>Geofence Tracker</h1>
        <span className={`status status-${socketStatus}`}>
          <span className="status-dot" /> {STATUS_LABEL[socketStatus]}
        </span>
      </header>

      <nav className="tabs">
        {TABS.map((item) => (
          <button key={item.id} className={tab === item.id ? "tab tab-active" : "tab"} onClick={() => setTab(item.id)}>
            {item.label}
            {item.id === "live" && unread > 0 && <span className="unread">{unread}</span>}
          </button>
        ))}
      </nav>

      <main className="content">
        {tab === "geofences" && <GeofencesPage vehicleMarkers={vehicleMarkers} onChanged={loadGeofences} />}
        {tab === "vehicles" && (
          <VehiclesPage vehicles={vehicles} locations={locations} loading={vehiclesLoading} onChanged={loadVehicles} />
        )}
        {tab === "location" && (
          <LocationPage geofences={geofences} vehicles={vehicles} vehicleMarkers={vehicleMarkers} onChanged={loadVehicles} />
        )}
        {tab === "alerts" && <AlertsPage geofences={geofences} vehicles={vehicles} />}
        {tab === "live" && <LiveAlertsPage alerts={liveAlerts} status={socketStatus} onClear={() => setLiveAlerts([])} />}
        {tab === "history" && <HistoryPage geofences={geofences} vehicles={vehicles} />}
      </main>
    </div>
  );
}
