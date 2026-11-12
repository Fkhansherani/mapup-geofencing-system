import { useCallback, useEffect, useState } from "react";
import { EventBadge } from "../components/Badge";
import { useToast } from "../components/Toasts";
import { api } from "../api";
import { EVENT_TYPES } from "../config";
import { formatTime } from "../utils";

export default function AlertsPage({ geofences, vehicles }) {
  const notify = useToast();

  const [rules, setRules] = useState([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [form, setForm] = useState({ geofence_id: "", vehicle_id: "", event_type: "entry" });

  const loadRules = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.getAlerts();
      setRules(data.alerts);
    } catch (error) {
      notify(error.message, "error");
    } finally {
      setLoading(false);
    }
  }, [notify]);

  useEffect(() => {
    loadRules();
  }, [loadRules]);

  async function handleSubmit(event) {
    event.preventDefault();
    if (!form.geofence_id) {
      notify("Please choose a geofence.", "error");
      return;
    }

    setSaving(true);
    try {
      // An empty vehicle means "all vehicles", so we leave it out of the request.
      const body = { geofence_id: form.geofence_id, event_type: form.event_type };
      if (form.vehicle_id) body.vehicle_id = form.vehicle_id;

      await api.createAlert(body);
      notify("Alert rule created", "success");
      loadRules();
    } catch (error) {
      notify(error.message, "error");
    } finally {
      setSaving(false);
    }
  }

  async function toggleRule(rule) {
    const nextStatus = rule.status === "active" ? "inactive" : "active";
    try {
      await api.setAlertStatus(rule.alert_id, nextStatus);
      loadRules();
    } catch (error) {
      notify(error.message, "error");
    }
  }

  async function removeRule(rule) {
    if (!window.confirm(`Delete this alert rule for "${rule.geofence_name}"?`)) return;
    try {
      await api.deleteAlert(rule.alert_id);
      notify("Alert rule deleted", "success");
      loadRules();
    } catch (error) {
      notify(error.message, "error");
    }
  }

  return (
    <div className="page-grid page-grid-narrow-left">
      <section className="card">
        <h2>Configure alert</h2>
        <form onSubmit={handleSubmit} className="form">
          <label>
            Geofence
            <select value={form.geofence_id} onChange={(e) => setForm({ ...form, geofence_id: e.target.value })}>
              <option value="">Select a geofence...</option>
              {geofences.map((geofence) => (
                <option key={geofence.id} value={geofence.id}>
                  {geofence.name}
                </option>
              ))}
            </select>
          </label>
          <label>
            Vehicle (optional)
            <select value={form.vehicle_id} onChange={(e) => setForm({ ...form, vehicle_id: e.target.value })}>
              <option value="">All vehicles</option>
              {vehicles.map((vehicle) => (
                <option key={vehicle.id} value={vehicle.id}>
                  {vehicle.vehicle_number}
                </option>
              ))}
            </select>
          </label>
          <label>
            Alert when a vehicle
            <select value={form.event_type} onChange={(e) => setForm({ ...form, event_type: e.target.value })}>
              {EVENT_TYPES.map((type) => (
                <option key={type} value={type}>
                  {type === "both" ? "enters or exits" : type === "entry" ? "enters" : "exits"}
                </option>
              ))}
            </select>
          </label>
          <button type="submit" disabled={saving}>
            {saving ? "Saving..." : "Create alert rule"}
          </button>
        </form>
      </section>

      <section className="card">
        <h2>Alert rules</h2>
        {loading ? (
          <p className="hint">Loading...</p>
        ) : rules.length === 0 ? (
          <p className="hint">No alert rules yet. Without a rule, no live alert is sent.</p>
        ) : (
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Geofence</th>
                  <th>Vehicle</th>
                  <th>Event</th>
                  <th>Status</th>
                  <th>Created</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {rules.map((rule) => (
                  <tr key={rule.alert_id}>
                    <td>{rule.geofence_name}</td>
                    <td>{rule.vehicle_number || "All vehicles"}</td>
                    <td>
                      <EventBadge type={rule.event_type} />
                    </td>
                    <td>{rule.status}</td>
                    <td>{formatTime(rule.created_at)}</td>
                    <td className="actions">
                      <button className="btn-secondary btn-small" onClick={() => toggleRule(rule)}>
                        {rule.status === "active" ? "Pause" : "Resume"}
                      </button>
                      <button className="btn-danger btn-small" onClick={() => removeRule(rule)}>
                        Delete
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}
