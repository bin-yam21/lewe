import type {
  AuthResponse, Catalog, Item, ItemInput, List, Match, Message, Notification,
  PublicProfile, Rating, User, Want, WantInput,
} from './types'

const BASE = (import.meta.env.VITE_API_URL as string | undefined) ?? '/api/v1'

const ACCESS_KEY = 'lewe.access'
const REFRESH_KEY = 'lewe.refresh'

export class ApiError extends Error {
  status: number
  fields?: Record<string, string>
  constructor(status: number, message: string, fields?: Record<string, string>) {
    super(message)
    this.status = status
    this.fields = fields
  }
}

export const tokens = {
  get access() { return localStorage.getItem(ACCESS_KEY) },
  get refresh() { return localStorage.getItem(REFRESH_KEY) },
  set(a: AuthResponse) {
    localStorage.setItem(ACCESS_KEY, a.access_token)
    localStorage.setItem(REFRESH_KEY, a.refresh_token)
  },
  clear() {
    localStorage.removeItem(ACCESS_KEY)
    localStorage.removeItem(REFRESH_KEY)
  },
}

let onUnauthorized: () => void = () => {}
export function setUnauthorizedHandler(fn: () => void) { onUnauthorized = fn }

// Concurrent 401s share one refresh call: refresh tokens rotate on use.
let refreshing: Promise<boolean> | null = null
function refreshSession(): Promise<boolean> {
  if (!refreshing) {
    refreshing = (async () => {
      const refresh = tokens.refresh
      if (!refresh) return false
      try {
        const res = await fetch(`${BASE}/auth/refresh`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ refresh_token: refresh }),
        })
        if (!res.ok) return false
        tokens.set((await res.json()) as AuthResponse)
        return true
      } catch {
        return false
      }
    })().finally(() => { refreshing = null })
  }
  return refreshing
}

async function request<T>(method: string, path: string, body?: unknown, retry = true): Promise<T> {
  const headers: Record<string, string> = {}
  if (body !== undefined) headers['Content-Type'] = 'application/json'
  if (tokens.access) headers.Authorization = `Bearer ${tokens.access}`

  const res = await fetch(`${BASE}${path}`, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  })

  if (res.status === 401 && retry && tokens.refresh && !path.startsWith('/auth/')) {
    if (await refreshSession()) return request<T>(method, path, body, false)
    tokens.clear()
    onUnauthorized()
  }

  if (!res.ok) {
    let msg = res.statusText
    let fields: Record<string, string> | undefined
    try {
      const data = await res.json()
      msg = data.message || data.error || msg
      fields = data.fields
      if (fields) msg = Object.entries(fields).map(([k, v]) => `${k}: ${v}`).join('; ')
    } catch { /* non-JSON error body */ }
    throw new ApiError(res.status, msg, fields)
  }
  if (res.status === 204) return undefined as T
  const text = await res.text()
  return (text ? JSON.parse(text) : undefined) as T
}

const get = <T>(p: string) => request<T>('GET', p)
const post = <T>(p: string, b?: unknown) => request<T>('POST', p, b ?? {})
const put = <T>(p: string, b: unknown) => request<T>('PUT', p, b)
const del = <T>(p: string) => request<T>('DELETE', p)

function qs(params: Record<string, string | number | undefined>) {
  const s = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) if (v !== undefined && v !== '') s.set(k, String(v))
  const out = s.toString()
  return out ? `?${out}` : ''
}

export const api = {
  // auth
  telegramLogin: (init_data: string) => post<AuthResponse>('/auth/telegram', { init_data }),
  login: (email: string, password: string) => post<AuthResponse>('/auth/login', { email, password }),
  register: (email: string, password: string, full_name: string) =>
    post<AuthResponse>('/auth/register', { email, password, full_name }),
  logout: (refresh_token: string) => post<void>('/auth/logout', { refresh_token }),

  // users
  me: () => get<User>('/users/me'),
  updateMe: (p: Partial<User> & { full_name: string }) => put<User>('/users/me', p),
  publicProfile: (id: string) => get<PublicProfile>(`/users/${id}`),
  userRatings: (id: string) => get<List<Rating>>(`/users/${id}/ratings`),

  // catalog / items
  catalog: () => get<Catalog>('/categories'),
  items: (p: { category?: string; q?: string; owner_id?: string; limit?: number; offset?: number }) =>
    get<List<Item>>(`/items${qs(p)}`),
  myItems: (status?: string) => get<List<Item>>(`/users/me/items${qs({ status, limit: 100 })}`),
  item: (id: string) => get<Item>(`/items/${id}`),
  createItem: (i: ItemInput) => post<Item>('/items', i),
  updateItem: (id: string, i: ItemInput) => put<Item>(`/items/${id}`, i),
  deleteItem: (id: string) => del<void>(`/items/${id}`),

  // wants
  wants: (status?: string) => get<List<Want>>(`/wants${qs({ status, limit: 100 })}`),
  createWant: (w: WantInput) => post<Want>('/wants', w),
  deleteWant: (id: string) => del<void>(`/wants/${id}`),

  // matches
  matches: (status?: string) => get<List<Match>>(`/matches${qs({ status, limit: 100 })}`),
  match: (id: string) => get<Match>(`/matches/${id}`),
  accept: (id: string) => post<Match>(`/matches/${id}/accept`),
  decline: (id: string) => post<Match>(`/matches/${id}/decline`),
  cancel: (id: string) => post<Match>(`/matches/${id}/cancel`),
  complete: (id: string) => post<Match>(`/matches/${id}/complete`),
  setExchange: (id: string, method: string, details: string | null) =>
    put<Match>(`/matches/${id}/exchange`, { method, details }),
  rate: (id: string, score: number, comment: string | null) =>
    post<Rating>(`/matches/${id}/rating`, { score, comment }),
  messages: (id: string) => get<List<Message>>(`/matches/${id}/messages?limit=100`),
  sendMessage: (id: string, body: string) => post<Message>(`/matches/${id}/messages`, { body }),

  // notifications
  notifications: () => get<List<Notification>>('/notifications?limit=50'),
  unreadCount: () => get<{ unread: number }>('/notifications/unread-count'),
  readAll: () => post<void>('/notifications/read-all'),
  readOne: (id: string) => post<void>(`/notifications/${id}/read`),
}
