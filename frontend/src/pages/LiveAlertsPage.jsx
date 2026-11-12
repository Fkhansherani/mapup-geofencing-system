import { CategoryBadge, EventBadge } from "../components/Badge";
import { formatTime } from "../utils";

// Shows the alerts received over the WebSocket since the page was opened (newest first).
export default function LiveAlertsPage({ alerts, status, onClear }) {
  return (
    <section className="card">
      <div className="row row-between">
        <h2>Live alerts</h2>
        <button className="btn-secondary btn-small" onClick={onClear} disabled={alerts.length === 0}>
          Clear feed
        </button>
      </div>

      {status !== "connected" && (
        <p className="hint">
          {status === "connecting" ? "Connecting to the alert stream..." : "Disconnected. Trying to reconnect..."}
        </p>
      )}

      {alerts.length === 0 ? (
        <p className="hint">
          Waiting for alerts. Create an alert rule, then update a vehicle location so it enters or exits that geofence.
        </p>
      ) : (
        <ul className="feed">
          {alerts.map((alert) => (
            <li key={alert.event_id} className={alert.geofence.category === "restricted_zone" ? "feed-danger" : ""}>
              <div className="row row-between">
                <div>
                  <strong>{alert.vehicle.vehicle_number}</strong> ({alert.vehicle.driver_name}){" "}
                  {alert.event_type === "entry" ? "entered" : "exited"} <strong>{alert.geofence.geofence_name}</strong>
                </div>
                <div className="row">
                  <EventBadge type={alert.event_type} />
                  <CategoryBadge category={alert.geofence.category} />
                </div>
              </div>
              <div className="hint">
                {formatTime(alert.timestamp)} - at {alert.location.latitude.toFixed(5)}, {alert.location.longitude.toFixed(5)}
              </div>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
