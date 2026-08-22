export function formatTimestamp(value?: string) {
  if (!value) {
    return "n/a";
  }
  return new Date(value).toLocaleString();
}

export function formatActivityTimestamp(value?: string) {
  if (!value) {
    return "n/a";
  }
  const date = new Date(value);
  const now = new Date();
  const sameDay = date.toDateString() === now.toDateString();
  const time = date.toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
  });

  if (sameDay) {
    return time;
  }

  return `${date.toLocaleDateString([], {
    month: "short",
    day: "numeric",
  })}, ${time}`;
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

export function availabilitySeverity(value?: number) {
  if (value === undefined) {
    return "unknown";
  }
  if (value < 90) {
    return "critical";
  }
  if (value < 99) {
    return "degraded";
  }
  return "healthy";
}

export function formatEventType(value: string) {
  return value
    .split(".")
    .filter(Boolean)
    .map((part) => part.slice(0, 1).toUpperCase() + part.slice(1))
    .join(" ");
}

export function pluralize(count: number, singular: string, plural?: string) {
  return `${count} ${count === 1 ? singular : (plural ?? `${singular}s`)}`;
}
