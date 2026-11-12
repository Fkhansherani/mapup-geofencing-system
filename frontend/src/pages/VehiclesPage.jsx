import { useState } from "react";
import { useToast } from "../components/Toasts";
import { api } from "../api";
import { formatTime } from "../utils";

const emptyForm = { vehicle_number: "", driver_name: "", vehicle_type: "truck", phone: "" };
const VEHICLE_TYPES = ["truck", "car", "van", "motorcycle"];

export default function VehiclesPage({ vehicles, locations, loading, onChanged }) {
  const notify = useToast();
  const [form, setForm] = useState(emptyForm);
  const [saving, setSaving] = useState(false);

  async function handleSubmit(event) {
    event.preventDefault();
    setSaving(true);
    try {
      await api.createVehicle(form);
      notify(`Vehicle ${form.vehicle_number} registered`, "success");
      setForm(emptyForm);
      onChanged();
    } catch (error) {
      notify(error.message, "error");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="page-grid page-grid-narrow-left">
      <section className="card">
        <h2>Register vehicle</h2>
        <form onSubmit={handleSubmit} className="form">
          <label>
            Vehicle number
            <input
              required
              placeholder="KA-01-AB-1234"
              value={form.vehicle_number}
              onChange={(e) => setForm({ ...form, vehicle_number: e.target.value })}
            />
          </label>
          <label>
            Driver name
            <input required value={form.driver_name} onChange={(e) => setForm({ ...form, driver_name: e.target.value })} />
          </label>
          <label>
            Vehicle type
            <select value={form.vehicle_type} onChange={(e) => setForm({ ...form, vehicle_type: e.target.value })}>
              {VEHICLE_TYPES.map((type) => (
                <option key={type} value={type}>
                  {type}
                </option>
              ))}
            </select>
          </label>
          <label>
            Phone
            <input required placeholder="+911234567890" value={form.phone} onChange={(e) => setForm({ ...form, phone: e.target.value })} />
          </label>
          <button type="submit" disabled={saving}>
            {saving ? "Saving..." : "Register vehicle"}
          </button>
        </form>
      </section>

      <section className="card">
        <h2>Vehicles</h2>
        {loading ? (
          <p className="hint">Loading...</p>
        ) : vehicles.length === 0 ? (
          <p className="hint">No vehicles registered yet.</p>
        ) : (
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Vehicle</th>
                  <th>Driver</th>
                  <th>Type</th>
                  <th>Phone</th>
                  <th>Last location</th>
                  <th>Inside</th>
                </tr>
              </thead>
              <tbody>
                {vehicles.map((vehicle) => {
                  const info = locations[vehicle.id];
                  const position = info?.current_location;
                  return (
                    <tr key={vehicle.id}>
                      <td>
                        <strong>{vehicle.vehicle_number}</strong>
                      </td>
                      <td>{vehicle.driver_name}</td>
                      <td>{vehicle.vehicle_type}</td>
                      <td>{vehicle.phone}</td>
                      <td>
                        {position ? (
                          <>
                            {position.latitude.toFixed(5)}, {position.longitude.toFixed(5)}
                            <div className="hint">{formatTime(position.timestamp)}</div>
                          </>
                        ) : (
                          <span className="hint">No location yet</span>
                        )}
                      </td>
                      <td>
                        {info && info.current_geofences.length > 0 ? (
                          info.current_geofences.map((g) => (
                            <div key={g.geofence_id}>{g.geofence_name}</div>
                          ))
                        ) : (
                          <span className="hint">-</span>
                        )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}
