export function formatTimestamp(value?: string) {
  if (!value) {
    return "n/a";
  }
  return new Date(value).toLocaleString();
}

export function formatDuration(seconds?: number) {
  if (!seconds || seconds <= 0) {
    return "0s";
  }
  if (seconds < 60) {
    return `${seconds}s`;
  }
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  if (minutes < 60) {
    return `${minutes}m ${remainingSeconds}s`;
  }
  const hours = Math.floor(minutes / 60);
  return `${hours}h ${minutes % 60}m`;
}

export function formatAvailability(value?: number) {
  if (value === undefined) {
    return "n/a";
  }
  return `${value.toFixed(3)}%`;
}
