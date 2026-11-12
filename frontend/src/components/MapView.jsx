import { MapContainer, TileLayer, Polygon, Polyline, CircleMarker, Popup, useMapEvents } from "react-leaflet";
import { CATEGORIES, DEFAULT_CENTER } from "../config";
import { round6 } from "../utils";

// Invisible component: it only listens for clicks on the map.
function ClickHandler({ onMapClick }) {
  useMapEvents({
    click(event) {
      if (onMapClick) onMapClick([round6(event.latlng.lat), round6(event.latlng.lng)]);
    },
  });
  return null;
}

// One map used by several pages.
//   geofences   - saved polygons to draw
//   vehicles    - [{ id, number, driver, position: [lat, lng], geofences: [names] }]
//   draftPoints - points of a geofence the user is drawing right now
//   pickedPoint - a single point chosen by the user (location page)
//   onMapClick  - called with [lat, lng] when the map is clicked
export default function MapView({ geofences = [], vehicles = [], draftPoints = [], pickedPoint = null, onMapClick }) {
  return (
    <MapContainer center={DEFAULT_CENTER} zoom={12} className={`map ${onMapClick ? "map-clickable" : ""}`}>
      <TileLayer
        attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
        url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
      />

      {onMapClick && <ClickHandler onMapClick={onMapClick} />}

      {/* Saved geofences */}
      {geofences.map((geofence) => {
        const color = CATEGORIES[geofence.category]?.color || "#2563eb";
        return (
          <Polygon key={geofence.id} positions={geofence.coordinates} pathOptions={{ color, fillOpacity: 0.2 }}>
            <Popup>
              <strong>{geofence.name}</strong>
              <br />
              {CATEGORIES[geofence.category]?.label}
            </Popup>
          </Polygon>
        );
      })}

      {/* Polygon being drawn: a dashed line plus a dot for every point */}
      {draftPoints.length > 1 && (
        <Polyline positions={draftPoints} pathOptions={{ color: "#0f172a", dashArray: "6 6" }} />
      )}
      {draftPoints.map((point, index) => (
        <CircleMarker key={index} center={point} radius={5} pathOptions={{ color: "#0f172a", fillOpacity: 1 }} />
      ))}

      {/* Vehicles */}
      {vehicles.map((vehicle) => (
        <CircleMarker
          key={vehicle.id}
          center={vehicle.position}
          radius={9}
          pathOptions={{ color: "#1d4ed8", fillColor: "#3b82f6", fillOpacity: 0.9 }}
        >
          <Popup>
            <strong>{vehicle.number}</strong>
            <br />
            Driver: {vehicle.driver}
            <br />
            {vehicle.geofences.length > 0 ? `Inside: ${vehicle.geofences.join(", ")}` : "Not inside any geofence"}
          </Popup>
        </CircleMarker>
      ))}

      {/* The point chosen for a location update */}
      {pickedPoint && (
        <CircleMarker center={pickedPoint} radius={8} pathOptions={{ color: "#b91c1c", fillColor: "#ef4444", fillOpacity: 0.9 }} />
      )}
    </MapContainer>
  );
}
