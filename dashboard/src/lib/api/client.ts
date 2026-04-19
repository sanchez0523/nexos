// Typed fetch wrapper for the Nexos API.
//
// Responsibilities:
//   - Same-origin requests only (Caddy serves dashboard + API on one host).
//   - `credentials: 'include'` so httpOnly auth cookies are attached.
//   - Single-flight auto-refresh: if any request returns 401, we exchange the
//     refresh cookie for a new pair and retry the original request exactly
//     once. Multiple 401s in parallel share one refresh promise.
//   - On refresh failure, hard-redirect to /login.

export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly body: unknown
  ) {
    super(`nexos api error ${status}`);
    this.name = 'ApiError';
  }
}

interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE';
  body?: unknown;
  // Skip the auto-refresh retry path (used by the refresh call itself).
  skipAuthRefresh?: boolean;
}

let refreshInFlight: Promise<boolean> | null = null;

async function runRefresh(): Promise<boolean> {
  if (refreshInFlight) return refreshInFlight;
  refreshInFlight = (async () => {
    try {
      const res = await fetch('/api/auth/refresh', {
        method: 'POST',
        credentials: 'include'
      });
      return res.ok;
    } catch {
      return false;
    } finally {
      refreshInFlight = null;
    }
  })();
  return refreshInFlight;
}

export async function api<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const headers: Record<string, string> = {};
  let body: BodyInit | undefined;
  if (opts.body !== undefined) {
    headers['Content-Type'] = 'application/json';
    body = JSON.stringify(opts.body);
  }

  const doFetch = () =>
    fetch(path, {
      method: opts.method ?? 'GET',
      credentials: 'include',
      headers,
      body
    });

  let res = await doFetch();

  if (res.status === 401 && !opts.skipAuthRefresh) {
    const refreshed = await runRefresh();
    if (refreshed) {
      res = await doFetch();
    } else if (typeof window !== 'undefined' && !location.pathname.startsWith('/login')) {
      location.href = '/login';
      // Return a never-resolving promise so callers don't proceed.
      return new Promise<T>(() => {});
    }
  }

  if (!res.ok) {
    let errBody: unknown = null;
    try {
      errBody = await res.json();
    } catch {
      /* ignore */
    }
    throw new ApiError(res.status, errBody);
  }

  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}
