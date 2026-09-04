import { tokenStore } from '@/lib/storage';

import { API_V1 } from './config';
import type { AuthResponse } from './types';

/**
 * A single error shape for the whole app. The Go API answers with either
 * `{error, message}` or, for validation failures, `{error, fields:{...}}` —
 * both collapse into this.
 */
export class ApiError extends Error {
  status: number;
  fields?: Record<string, string>;

  constructor(status: number, message: string, fields?: Record<string, string>) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.fields = fields;
  }

  /** True when the request failed before reaching the server. */
  get isNetworkError() {
    return this.status === 0;
  }
}

type RequestOptions = {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE';
  body?: unknown;
  /**
   * Multipart payload. Takes precedence over `body`; the Content-Type header is
   * deliberately left unset so the runtime can add its own multipart boundary.
   */
  formData?: FormData;
  /** Skip the Authorization header and the refresh dance (auth endpoints). */
  anonymous?: boolean;
  signal?: AbortSignal;
};

/** Called when refreshing fails — the session is unrecoverable, log the user out. */
let onSessionExpired: (() => void) | null = null;
export function setSessionExpiredHandler(fn: (() => void) | null) {
  onSessionExpired = fn;
}

/**
 * In-flight refresh. Ten screens mounting at once produce ten 401s; without
 * this they would fire ten refreshes, and since the API *rotates* refresh
 * tokens on use, nine of them would fail and log the user out. They all await
 * the same promise instead.
 */
let refreshInFlight: Promise<string | null> | null = null;

async function refreshAccessToken(): Promise<string | null> {
  if (refreshInFlight) return refreshInFlight;

  refreshInFlight = (async () => {
    try {
      const { refreshToken } = await tokenStore.get();
      if (!refreshToken) return null;

      const res = await fetch(`${API_V1}/auth/refresh`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: refreshToken }),
      });

      if (!res.ok) return null;

      const data = (await res.json()) as AuthResponse;
      await tokenStore.save(data.access_token, data.refresh_token);
      return data.access_token;
    } catch {
      return null;
    } finally {
      // Cleared in a microtask so concurrent callers all see the same promise.
      setTimeout(() => {
        refreshInFlight = null;
      }, 0);
    }
  })();

  return refreshInFlight;
}

async function parseError(res: Response): Promise<ApiError> {
  let message = `Request failed (${res.status})`;
  let fields: Record<string, string> | undefined;

  try {
    const body = await res.json();
    if (body?.fields && typeof body.fields === 'object') {
      fields = body.fields;
      // Surface the first field error as the headline message so a caller that
      // ignores `fields` still shows something useful.
      const first = Object.entries(fields ?? {})[0];
      message = first ? `${first[0].replace(/_/g, ' ')} ${first[1]}` : 'Please check the form';
    } else if (typeof body?.message === 'string' && body.message) {
      message = body.message;
    } else if (typeof body?.error === 'string' && body.error) {
      message = body.error;
    }
  } catch {
    // Non-JSON body — keep the status-based message.
  }

  return new ApiError(res.status, message, fields);
}

async function send(path: string, options: RequestOptions, accessToken: string | null) {
  const headers: Record<string, string> = { Accept: 'application/json' };
  if (options.formData === undefined && options.body !== undefined) {
    headers['Content-Type'] = 'application/json';
  }
  if (accessToken) headers.Authorization = `Bearer ${accessToken}`;

  const body = options.formData
    ? options.formData
    : options.body === undefined
      ? undefined
      : JSON.stringify(options.body);

  return fetch(`${API_V1}${path}`, {
    method: options.method ?? 'GET',
    headers,
    body,
    signal: options.signal,
  });
}

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { accessToken } = options.anonymous ? { accessToken: null } : await tokenStore.get();

  let res: Response;
  try {
    res = await send(path, options, accessToken);
  } catch {
    throw new ApiError(
      0,
      'Cannot reach the server. Check that the API is running and that your phone is on the same network.',
    );
  }

  // One retry, and only one: refresh the access token and replay the request.
  if (res.status === 401 && !options.anonymous) {
    const fresh = await refreshAccessToken();
    if (!fresh) {
      await tokenStore.clear();
      onSessionExpired?.();
      throw new ApiError(401, 'Your session has expired. Please sign in again.');
    }

    try {
      res = await send(path, options, fresh);
    } catch {
      throw new ApiError(0, 'Cannot reach the server.');
    }
  }

  if (!res.ok) throw await parseError(res);

  if (res.status === 204) return undefined as T;

  const text = await res.text();
  if (!text) return undefined as T;
  return JSON.parse(text) as T;
}

export const api = {
  get: <T>(path: string, options?: Omit<RequestOptions, 'method' | 'body'>) =>
    request<T>(path, { ...options, method: 'GET' }),
  post: <T>(path: string, body?: unknown, options?: Omit<RequestOptions, 'method' | 'body'>) =>
    request<T>(path, { ...options, method: 'POST', body }),

  put: <T>(path: string, body?: unknown, options?: Omit<RequestOptions, 'method' | 'body'>) =>
    request<T>(path, { ...options, method: 'PUT', body }),
  delete: <T>(path: string, options?: Omit<RequestOptions, 'method' | 'body'>) =>
    request<T>(path, { ...options, method: 'DELETE' }),
};
