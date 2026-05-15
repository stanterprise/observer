function formatDurationParts(totalMilliseconds: number): string {
  if (!Number.isFinite(totalMilliseconds) || totalMilliseconds <= 0) {
    return "0 seconds";
  }

  let remainingMilliseconds = Math.floor(totalMilliseconds);
  const units = [
    { label: "y", value: 31536000000 },
    { label: "d", value: 86400000 },
    { label: "hr", value: 3600000 },
    { label: "m", value: 60000 },
    { label: "s", value: 1000 },
    { label: "ms", value: 1 },
  ];

  return (
    units
      .map((unit) => {
        const value = Math.floor(remainingMilliseconds / unit.value);
        remainingMilliseconds %= unit.value;

        return value > 0 ? `${value}${unit.label}` : "";
      })
      .filter(Boolean)
      .join(" ") || "0 seconds"
  );
}

export function humanizeSeconds(seconds: number): string {
  return humanizeDuration(seconds, 1);
}

export function humanizeMilliseconds(milliseconds: number): string {
  return humanizeDuration(milliseconds, 1_000);
}

export function humanizeDuration(
  duration: number,
  divisor: number = 1_000_000,
): string {
  if (!Number.isFinite(duration) || !Number.isFinite(divisor) || divisor <= 0) {
    return "0 seconds";
  }

  return formatDurationParts((duration * 1_000) / divisor);
}
