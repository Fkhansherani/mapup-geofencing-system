import { API_URL } from "./config";

// One small helper for every call: sends JSON, reads JSON and turns
// error responses into a normal JavaScript Error with the server's message.
async function request(path, options = {}) {
  let response;
  try {
    response = await fetch(API_URL + path, {
      headers: { "Content-Type": "application/json" },
      ...options,
    });
  } catch {
    throw new Error("Cannot reach the server. Is the backend running?");
  }

  const data = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(data.error || `Request failed (${response.status})`);
  }
  return data;
}

// Builds "?a=1&b=2" and skips empty values.
function queryString(params) {
  const search = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value !== "" && value !== undefined && value !== null) search.set(key, value);
  });
  const text = search.toString();
  return text ? `?${text}` : "";
}

export const api = {
  getGeofences: (category = "") => request(`/geofences${queryString({ category })}`),
  createGeofence: (body) => request("/geofences", { method: "POST", body: JSON.stringify(body) }),

  getVehicles: () => request("/vehicles"),
  createVehicle: (body) => request("/vehicles", { method: "POST", body: JSON.stringify(body) }),
  updateLocation: (body) => request("/vehicles/location", { method: "POST", body: JSON.stringify(body) }),
  getVehicleLocation: (vehicleId) => request(`/vehicles/location/${vehicleId}`),

  getAlerts: () => request("/alerts"),
  createAlert: (body) => request("/alerts/configure", { method: "POST", body: JSON.stringify(body) }),
  setAlertStatus: (id, status) => request(`/alerts/${id}`, { method: "PATCH", body: JSON.stringify({ status }) }),
  deleteAlert: (id) => request(`/alerts/${id}`, { method: "DELETE" }),

  getHistory: (filters) => request(`/violations/history${queryString(filters)}`),
};
