import { CATEGORIES } from "../config";

// Small coloured label showing a geofence category.
export function CategoryBadge({ category }) {
  const info = CATEGORIES[category];
  return (
    <span className="badge" style={{ background: info?.color || "#64748b" }}>
      {info?.label || category}
    </span>
  );
}

export function EventBadge({ type }) {
  return <span className={`badge badge-${type}`}>{type}</span>;
}
