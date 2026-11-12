// Backend address. Set VITE_API_URL in .env (or in Vercel / Netlify settings).
export const API_URL = (import.meta.env.VITE_API_URL || "http://localhost:8080").replace(/\/$/, "");

// http://... -> ws://...   and   https://... -> wss://...
export const WS_URL = API_URL.replace(/^http/, "ws") + "/ws/alerts";

// Geofence categories: label for the UI and a colour for the map.
export const CATEGORIES = {
  delivery_zone: { label: "Delivery zone", color: "#16a34a" },
  restricted_zone: { label: "Restricted zone", color: "#dc2626" },
  toll_zone: { label: "Toll zone", color: "#ea580c" },
  customer_area: { label: "Customer area", color: "#7c3aed" },
};

export const EVENT_TYPES = ["entry", "exit", "both"];

// Where the map starts (Bengaluru).
export const DEFAULT_CENTER = [12.9716, 77.5946];
