import { useCallback, useEffect, useMemo, useState } from "react";
import MapView from "../components/MapView";
import { CategoryBadge } from "../components/Badge";
import { useToast } from "../components/Toasts";
import { api } from "../api";
import { CATEGORIES } from "../config";
import { parsePoints } from "../utils";

const emptyForm = { name: "", description: "", category: "delivery_zone" };

export default function GeofencesPage({ vehicleMarkers, onChanged }) {
  const notify = useToast();

  const [categoryFilter, setCategoryFilter] = useState("");
  const [geofences, setGeofences] = useState([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const [form, setForm] = useState(emptyForm);
  // The coordinates are kept as text, one "lat, lng" per line.
  // Clicking the map adds a line; typing in the box changes the drawing.
  const [pointsText, setPointsText] = useState("");
  const draftPoints = useMemo(() => parsePoints(pointsText), [pointsText]);

  const loadGeofences = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.getGeofences(categoryFilter);
      setGeofences(data.geofences);
    } catch (error) {
      notify(error.message, "error");
    } finally {
      setLoading(false);
    }
  }, [categoryFilter, notify]);

  useEffect(() => {
    loadGeofences();
  }, [loadGeofences]);

  function handleMapClick([lat, lng]) {
    setPointsText((text) => (text.trim() ? `${text.trimEnd()}\n${lat}, ${lng}` : `${lat}, ${lng}`));
  }

  function undoLastPoint() {
    setPointsText((text) => text.trimEnd().split("\n").slice(0, -1).join("\n"));
  }

  async function handleSubmit(event) {
    event.preventDefault();

    if (draftPoints.length < 3) {
      notify("Add at least 3 points: click on the map or type them in the box.", "error");
      return;
    }

    // Be friendly: if the user did not repeat the first point, close the polygon for them.
    const coordinates = [...draftPoints];
    const first = coordinates[0];
    const last = coordinates[coordinates.length - 1];
    if (first[0] !== last[0] || first[1] !== last[1]) coordinates.push(first);

    setSaving(true);
    try {
      await api.createGeofence({ ...form, coordinates });
      notify(`Geofence "${form.name}" created`, "success");
      setForm(emptyForm);
      setPointsText("");
      loadGeofences();
      onChanged();
    } catch (error) {
      notify(error.message, "error");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="page-grid">
      <section className="card map-card">
        <h2>Map</h2>
        <p className="hint">Click on the map to add the corners of a new geofence.</p>
        <MapView geofences={geofences} vehicles={vehicleMarkers} draftPoints={draftPoints} onMapClick={handleMapClick} />
      </section>

      <div className="side-column">
        <section className="card">
          <h2>New geofence</h2>
          <form onSubmit={handleSubmit} className="form">
            <label>
              Name
              <input required value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
            </label>
            <label>
              Description
              <input value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
            </label>
            <label>
              Category
              <select value={form.category} onChange={(e) => setForm({ ...form, category: e.target.value })}>
                {Object.entries(CATEGORIES).map(([value, info]) => (
                  <option key={value} value={value}>
                    {info.label}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Coordinates (latitude, longitude - one point per line)
              <textarea
                rows={5}
                value={pointsText}
                onChange={(e) => setPointsText(e.target.value)}
                placeholder={"12.97, 77.59\n12.98, 77.59\n12.98, 77.60"}
              />
            </label>
            <div className="row">
              <button type="button" className="btn-secondary" onClick={undoLastPoint} disabled={!pointsText}>
                Undo last point
              </button>
              <button type="button" className="btn-secondary" onClick={() => setPointsText("")} disabled={!pointsText}>
                Clear
              </button>
              <span className="hint">{draftPoints.length} points</span>
            </div>
            <button type="submit" disabled={saving}>
              {saving ? "Saving..." : "Create geofence"}
            </button>
          </form>
        </section>

        <section className="card">
          <div className="row row-between">
            <h2>Geofences</h2>
            <select value={categoryFilter} onChange={(e) => setCategoryFilter(e.target.value)} aria-label="Filter by category">
              <option value="">All categories</option>
              {Object.entries(CATEGORIES).map(([value, info]) => (
                <option key={value} value={value}>
                  {info.label}
                </option>
              ))}
            </select>
          </div>

          {loading ? (
            <p className="hint">Loading...</p>
          ) : geofences.length === 0 ? (
            <p className="hint">No geofences yet.</p>
          ) : (
            <ul className="list">
              {geofences.map((geofence) => (
                <li key={geofence.id}>
                  <div>
                    <strong>{geofence.name}</strong>
                    {geofence.description && <div className="hint">{geofence.description}</div>}
                  </div>
                  <CategoryBadge category={geofence.category} />
                </li>
              ))}
            </ul>
          )}
        </section>
      </div>
    </div>
  );
}
