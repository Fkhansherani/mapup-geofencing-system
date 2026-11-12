import { useCallback, useEffect, useState } from "react";
import { EventBadge } from "../components/Badge";
import { useToast } from "../components/Toasts";
import { api } from "../api";
import { formatTime, localInputToIso } from "../utils";

const PAGE_SIZE = 20;
const emptyFilters = { vehicle_id: "", geofence_id: "", start: "", end: "" };

export default function HistoryPage({ geofences, vehicles }) {
  const notify = useToast();

  // "form" is what the user is typing; "applied" is what was actually searched.
  // The list only reloads when the user presses "Apply filters".
  const [form, setForm] = useState(emptyFilters);
  const [applied, setApplied] = useState(emptyFilters);
  const [page, setPage] = useState(0);

  const [rows, setRows] = useState([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);

  const loadHistory = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.getHistory({
        vehicle_id: applied.vehicle_id,
        geofence_id: applied.geofence_id,
        start_date: applied.start ? localInputToIso(applied.start) : "",
        end_date: applied.end ? localInputToIso(applied.end) : "",
        limit: PAGE_SIZE,
        offset: page * PAGE_SIZE,
      });
      setRows(data.violations);
      setTotal(data.total_count);
    } catch (error) {
      notify(error.message, "error");
    } finally {
      setLoading(false);
    }
  }, [applied, page, notify]);

  useEffect(() => {
    loadHistory();
  }, [loadHistory]);

  function applyFilters(event) {
    event.preventDefault();
    setPage(0);
    setApplied(form);
  }

  function resetFilters() {
    setForm(emptyFilters);
    setApplied(emptyFilters);
    setPage(0);
  }

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  return (
    <section className="card">
      <h2>Violation history</h2>

      <form onSubmit={applyFilters} className="filters">
        <label>
          Vehicle
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
          Geofence
          <select value={form.geofence_id} onChange={(e) => setForm({ ...form, geofence_id: e.target.value })}>
            <option value="">All geofences</option>
            {geofences.map((geofence) => (
              <option key={geofence.id} value={geofence.id}>
                {geofence.name}
              </option>
            ))}
          </select>
        </label>
        <label>
          From
          <input type="datetime-local" value={form.start} onChange={(e) => setForm({ ...form, start: e.target.value })} />
        </label>
        <label>
          To
          <input type="datetime-local" value={form.end} onChange={(e) => setForm({ ...form, end: e.target.value })} />
        </label>
        <div className="row">
          <button type="submit">Apply filters</button>
          <button type="button" className="btn-secondary" onClick={resetFilters}>
            Reset
          </button>
        </div>
      </form>

      {loading ? (
        <p className="hint">Loading...</p>
      ) : rows.length === 0 ? (
        <p className="hint">No events found for these filters.</p>
      ) : (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Time</th>
                <th>Vehicle</th>
                <th>Geofence</th>
                <th>Event</th>
                <th>Location</th>
                <th>Alert sent</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr key={row.id}>
                  <td>{formatTime(row.timestamp)}</td>
                  <td>{row.vehicle_number}</td>
                  <td>{row.geofence_name}</td>
                  <td>
                    <EventBadge type={row.event_type} />
                  </td>
                  <td>
                    {row.latitude.toFixed(5)}, {row.longitude.toFixed(5)}
                  </td>
                  <td>{row.alerted ? "Yes" : "No"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <div className="row row-between pagination">
        <span className="hint">
          {total} events - page {page + 1} of {totalPages}
        </span>
        <div className="row">
          <button className="btn-secondary btn-small" onClick={() => setPage((p) => p - 1)} disabled={page === 0 || loading}>
            Previous
          </button>
          <button className="btn-secondary btn-small" onClick={() => setPage((p) => p + 1)} disabled={page + 1 >= totalPages || loading}>
            Next
          </button>
        </div>
      </div>
    </section>
  );
}
