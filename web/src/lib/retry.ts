// Mirrors the server's retry schedule (queue.WebhookWorker.NextRetry): the
// delay after attempt N is base * 2^(N-1) seconds, capped at 24 hours, minus a
// random 0-20%. Used to show how long a delivery keeps retrying before it is
// marked failed.

export const DEFAULT_MAX_RETRIES = 13;
export const DEFAULT_RETRY_BACKOFF_SECONDS = 60;
export const MAX_MAX_RETRIES = 20;
export const MAX_RETRY_BACKOFF_SECONDS = 3600;

const MAX_DELAY_SECONDS = 24 * 60 * 60;

/** Seconds from the first attempt to the last retry, without jitter (the upper bound). */
export function retryWindowSeconds(maxRetries: number, baseSeconds: number): number {
  if (!(maxRetries > 0) || !(baseSeconds > 0)) return 0;
  let total = 0;
  for (let attempt = 1; attempt <= maxRetries; attempt++) {
    total += Math.min(baseSeconds * 2 ** (attempt - 1), MAX_DELAY_SECONDS);
  }
  return total;
}

/** "about 3.4 days", "about 7 minutes", or "no retries". */
export function describeRetryWindow(maxRetries: number, baseSeconds: number): string {
  const s = retryWindowSeconds(maxRetries, baseSeconds);
  if (s === 0) return 'no retries';
  const units: [number, string][] = [
    [86400, 'day'],
    [3600, 'hour'],
    [60, 'minute'],
  ];
  for (const [size, name] of units) {
    if (s >= size) {
      const n = Math.round((s / size) * 10) / 10;
      return `about ${n} ${name}${n === 1 ? '' : 's'}`;
    }
  }
  return `about ${s} second${s === 1 ? '' : 's'}`;
}
