import { useState } from "react";
import MapView from "../components/MapView";
import { useToast } from "../components/Toasts";
import { api } from "../api";
import { localInputToIso, nowForInput } from "../utils";

export default function LocationPage({ geofences, vehicles, vehicleMarkers, onChanged }) {
  const notify = useToast();

  const [vehicleId, setVehicleId] = useState("");
  const [latitude, setLatitude] = useState("");
  const [longitude, setLongitude] = useState("");
  const [time, setTime] = useState(nowForInput());
  const [sending, setSending] = useState(false);
  const [result, setResult] = useState(null);

  const lat = Number(latitude);
  const lng = Number(longitude);
  const hasPoint = latitude !== "" && longitude !== "" && Number.isFinite(lat) && Number.isFinite(lng);

  function handleMapClick([clickedLat, clickedLng]) {
    setLatitude(String(clickedLat));
    setLongitude(String(clickedLng));
  }

  async function handleSubmit(event) {
    event.preventDefault();
    if (!vehicleId) {
      notify("Please choose a vehicle first.", "error");
      return;
    }
    if (!hasPoint) {
      notify("Click on the map or type a latitude and longitude.", "error");
      return;
    }

    setSending(true);
    try {
      const data = await api.updateLocation({
        vehicle_id: vehicleId,
        latitude: lat,
        longitude: lng,
        timestamp: localInputToIso(time),
      });
      setResult(data);
      notify("Location updated", "success");
      onChanged();
    } catch (error) {
      notify(error.message, "error");
    } finally {
      setSending(false);
    }
  }

  return (
    <div className="page-grid">
      <section className="card map-card">
        <h2>Pick a location</h2>
        <p className="hint">Click anywhere on the map to set the vehicle position (red dot).</p>
        <MapView geofences={geofences} vehicles={vehicleMarkers} pickedPoint={hasPoint ? [lat, lng] : null} onMapClick={handleMapClick} />
      </section>

      <div className="side-column">
        <section className="card">
          <h2>Update vehicle location</h2>
          <form onSubmit={handleSubmit} className="form">
            <label>
              Vehicle
              <select value={vehicleId} onChange={(e) => setVehicleId(e.target.value)}>
                <option value="">Select a vehicle...</option>
                {vehicles.map((vehicle) => (
                  <option key={vehicle.id} value={vehicle.id}>
                    {vehicle.vehicle_number} - {vehicle.driver_name}
                  </option>
                ))}
              </select>
            </label>
            <div className="row">
              <label className="grow">
                Latitude
                <input type="number" step="any" value={latitude} onChange={(e) => setLatitude(e.target.value)} />
              </label>
              <label className="grow">
                Longitude
                <input type="number" step="any" value={longitude} onChange={(e) => setLongitude(e.target.value)} />
              </label>
            </div>
            <label>
              Time
              <input type="datetime-local" value={time} onChange={(e) => setTime(e.target.value)} />
            </label>
            <button type="button" className="btn-secondary" onClick={() => setTime(nowForInput())}>
              Use current time
            </button>
            <button type="submit" disabled={sending}>
              {sending ? "Sending..." : "Update location"}
            </button>
          </form>
        </section>

        {result && (
          <section className="card">
            <h2>Result</h2>
            {result.stale && <p className="hint">This update was older than the latest one, so the current state was not changed.</p>}
            {result.current_geofences.length === 0 ? (
              <p>The vehicle is not inside any geofence.</p>
            ) : (
              <>
                <p>The vehicle is inside:</p>
                <ul className="list">
                  {result.current_geofences.map((g) => (
                    <li key={g.geofence_id}>{g.geofence_name}</li>
                  ))}
                </ul>
              </>
            )}
          </section>
        )}
      </div>
    </div>
  );
}
