// Turns text like
//   12.97, 77.59
//   12.98, 77.60
// into [[12.97, 77.59], [12.98, 77.6]]. Lines that are not "number, number" are ignored.
export function parsePoints(text) {
  return text
    .split("\n")
    .map((line) => line.split(/[,\s]+/).filter(Boolean).map(Number))
    .filter((pair) => pair.length === 2 && pair.every(Number.isFinite));
}

// Rounds to 6 decimals (about 10 cm) so map clicks give clean numbers.
export function round6(number) {
  return Math.round(number * 1e6) / 1e6;
}

// <input type="datetime-local"> gives "2025-01-15T10:35" in local time.
// The API wants ISO 8601 in UTC, e.g. "2025-01-15T05:05:00.000Z".
export function localInputToIso(value) {
  return new Date(value).toISOString();
}

// The value a datetime-local input expects for "now" (local time, minutes precision).
export function nowForInput() {
  const now = new Date();
  now.setMinutes(now.getMinutes() - now.getTimezoneOffset());
  return now.toISOString().slice(0, 16);
}

export function formatTime(iso) {
  return new Date(iso).toLocaleString();
}
