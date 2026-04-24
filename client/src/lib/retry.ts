/**
 * Retry a fetch request with exponential backoff and jitter.
 * Permanent errors (400, 401, 403, 404, 409, 415) are not retried.
 * Rate limit (429) and server errors (500, 502, 503, 504) are retried.
 */
const PERMANENT_ERRORS = new Set([400, 401, 403, 404, 409, 415]);

export interface RetryOptions {
  maxRetries: number;
  baseDelayMs: number;
  maxDelayMs: number;
}

const DEFAULT_OPTIONS: RetryOptions = {
  maxRetries: 3,
  baseDelayMs: 1000,
  maxDelayMs: 30000,
};

export async function retryFetch(
  request: () => Promise<Response>,
  options: Partial<RetryOptions> = {}
): Promise<Response> {
  const opts = { ...DEFAULT_OPTIONS, ...options };

  let lastError: Error | null = null;

  for (let attempt = 0; attempt <= opts.maxRetries; attempt++) {
    try {
      const response = await request();

      if (PERMANENT_ERRORS.has(response.status)) {
        return response;
      }

      if (response.status === 429 || response.status >= 500) {
        if (attempt < opts.maxRetries) {
          const delay = backoffDelay(attempt, opts.baseDelayMs, opts.maxDelayMs);
          await sleep(delay);
          continue;
        }
        return response;
      }

      return response;
    } catch (err) {
      lastError = err instanceof Error ? err : new Error(String(err));
      if (attempt < opts.maxRetries) {
        const delay = backoffDelay(attempt, opts.baseDelayMs, opts.maxDelayMs);
        await sleep(delay);
        continue;
      }
    }
  }

  throw lastError || new Error("Max retries exceeded");
}

function backoffDelay(attempt: number, baseMs: number, maxMs: number): number {
  const exponential = baseMs * Math.pow(2, attempt);
  const jitter = Math.random() * baseMs;
  return Math.min(exponential + jitter, maxMs);
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}