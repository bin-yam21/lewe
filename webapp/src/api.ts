import { inTelegram, tg } from './telegram';
import type {
  AuthResponse, Catalog, Item, ItemInput, List, Match, Message, Notification, PublicProfile, User, Want,
} from './types';

const BASE = (import.meta.env.VITE_API_URL ?? '') + '/api/v1';

export class ApiError extends Error {
  status: number;
  fields?: Record<string, string>;
  constructor(status: number, message: string, fields?: Record<string, string>) {
    super(message);
    this.status = status;
    this.fields = fields;
  }
}

// Tokens live in memory; sessionStorage only bridges reloads inside one
// Telegram session. Telegram re-signs init data on every launch, so a lost
// token is recovered by signing in again rather than kept long-term.
let accessToken: string | null = null;
let refreshToken: string | null = null;
try {
  accessToken = sessionStorage.getItem('lewe.access');
  refreshToken = sessionStorage.getItem('lewe.refresh');
} catch {
  // Storage unavailable.
}

function saveTokens(a: AuthResponse) {
  accessToken = a.access_token;
  refreshToken = a.refresh_token;
  try {
    sessionStorage.setItem('lewe.access', a.access_token);
    sessionStorage.setItem('lewe.refresh', a.refresh_token);
  } catch {
    // Storage unavailable.
  }
}

export function signOut() {
  accessToken = refreshToken = null;
  try {
    sessionStorage.removeItem('lewe.access');
    sessionStorage.removeItem('lewe.refresh');
  } catch {
    // Storage unavailable.
  }
}

export const hasSession = () => accessToken !== null;

// One refresh at a time: parallel 401s share the same attempt.
let refreshing: Promise<boolean> | null = null;
function renewSession(): Promise<boolean> {
  refreshing ??= (async () => {
    try {
      if (refreshToken) {
        const res = await fetch(`${BASE}/auth/refresh`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ refresh_token: refreshToken }),
        });
        if (res.ok) {
          saveTokens(await res.json());
          return true;
        }
      }
      if (inTelegram && tg) {
        await telegramSignIn();
        return true;
      }
      return false;
    } catch {
      return false;
    } finally {
      refreshing = null;
    }
  })();
  return refreshing;
}

async function request<T>(method: string, path: string, body?: unknown, retry = true): Promise<T> {
  const headers: Record<string, string> = {};
  if (accessToken) headers.Authorization = `Bearer ${accessToken}`;
  let payload: BodyInit | undefined;
  if (body instanceof FormData) {
    payload = body;
  } else if (body !== undefined) {
    headers['Content-Type'] = 'application/json';
    payload = JSON.stringify(body);
  }

  const res = await fetch(BASE + path, { method, headers, body: payload });
  if (res.status === 401 && retry && !path.startsWith('/auth/')) {
    if (await renewSession()) return request<T>(method, path, body, false);
  }
  if (res.status === 204) return undefined as T;

  const data = await res.json().catch(() => null);
  if (!res.ok) {
    const message = data?.message ?? data?.error ?? `Request failed (${res.status})`;
    throw new ApiError(res.status, message, data?.fields);
  }
  return data as T;
}

const get = <T>(path: string) => request<T>('GET', path);
const post = <T>(path: string, body?: unknown) => request<T>('POST', path, body);
const put = <T>(path: string, body?: unknown) => request<T>('PUT', path, body);
const del = <T>(path: string) => request<T>('DELETE', path);

function qs(params: Record<string, string | number | boolean | undefined>) {
  const p = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) if (v !== undefined && v !== '') p.set(k, String(v));
  const s = p.toString();
  return s ? `?${s}` : '';
}

export async function telegramSignIn(): Promise<AuthResponse> {
  const res = await request<AuthResponse>('POST', '/auth/telegram', { init_data: tg?.initData ?? '' }, false);
  saveTokens(res);
  return res;
}

export async function emailSignIn(email: string, password: string): Promise<AuthResponse> {
  const res = await request<AuthResponse>('POST', '/auth/login', { email, password }, false);
  saveTokens(res);
  return res;
}

export const api = {
  catalog: () => get<Catalog>('/categories'),

  me: () => get<User>('/users/me'),
  updateMe: (u: Partial<User> & { full_name: string }) => put<User>('/users/me', u),
  profile: (id: string) => get<PublicProfile>(`/users/${id}`),

  items: (p: { category?: string; q?: string; owner_id?: string; limit?: number; offset?: number }) =>
    get<List<Item>>(`/items${qs(p)}`),
  myItems: (status?: string) => get<List<Item>>(`/users/me/items${qs({ status, limit: 100 })}`),
  item: (id: string) => get<Item>(`/items/${id}`),
  createItem: (i: ItemInput) => post<Item>('/items', i),
  updateItem: (id: string, i: ItemInput) => put<Item>(`/items/${id}`, i),
  withdrawItem: (id: string) => del<void>(`/items/${id}`),

  wants: () => get<List<Want>>('/wants?status=active&limit=100'),
  createWant: (w: { category: string; keywords: string[]; min_condition?: string | null; any_city: boolean }) =>
    post<Want>('/wants', w),
  cancelWant: (id: string) => del<void>(`/wants/${id}`),

  matches: (status?: string) => get<List<Match>>(`/matches${qs({ status, limit: 100 })}`),
  match: (id: string) => get<Match>(`/matches/${id}`),
  accept: (id: string) => post<Match>(`/matches/${id}/accept`),
  decline: (id: string) => post<Match>(`/matches/${id}/decline`),
  cancel: (id: string) => post<Match>(`/matches/${id}/cancel`),
  complete: (id: string) => post<Match>(`/matches/${id}/complete`),
  setExchange: (id: string, method: string, details?: string) =>
    put<Match>(`/matches/${id}/exchange`, { method, details }),
  rate: (id: string, score: number, comment?: string) => post(`/matches/${id}/rating`, { score, comment }),
  messages: (id: string) => get<List<Message>>(`/matches/${id}/messages?limit=100`),
  sendMessage: (id: string, body: string) => post<Message>(`/matches/${id}/messages`, { body }),

  notifications: () => get<List<Notification>>('/notifications?limit=50'),
  unreadCount: () => get<{ unread: number }>('/notifications/unread-count'),
  readAll: () => post<void>('/notifications/read-all'),

  upload: async (file: Blob): Promise<string> => {
    const form = new FormData();
    form.append('file', file, 'photo.jpg');
    const { url } = await post<{ url: string }>('/uploads', form);
    return url;
  },
};

/** Downscale a photo to at most `max` px on its long side, as JPEG, before upload. */
export async function shrinkImage(file: File, max = 1600): Promise<Blob> {
  const bitmap = await createImageBitmap(file).catch(() => null);
  if (!bitmap) return file;
  const scale = Math.min(1, max / Math.max(bitmap.width, bitmap.height));
  const canvas = document.createElement('canvas');
  canvas.width = Math.round(bitmap.width * scale);
  canvas.height = Math.round(bitmap.height * scale);
  canvas.getContext('2d')!.drawImage(bitmap, 0, 0, canvas.width, canvas.height);
  return new Promise((resolve) => canvas.toBlob((b) => resolve(b ?? file), 'image/jpeg', 0.85));
}
