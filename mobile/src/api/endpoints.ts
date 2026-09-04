import { api } from './client';
import type {
  AuthResponse,
  CreateItemPayload,
  Item,
  ItemList,
  User,
  UserRating,
} from './types';

export const auth = {
  register: (email: string, password: string, fullName: string) =>
    api.post<AuthResponse>(
      '/auth/register',
      { email, password, full_name: fullName },
      { anonymous: true },
    ),

  login: (email: string, password: string) =>
    api.post<AuthResponse>('/auth/login', { email, password }, { anonymous: true }),

  me: () => api.get<User>('/users/me'),

  updateProfile: (payload: {
    full_name: string;
    phone?: string | null;
    location?: string | null;
    bio?: string | null;
    avatar_url?: string | null;
  }) => api.put<User>('/users/me', payload),
};

export type ItemQuery = {
  category?: string;
  status?: string;
  q?: string;
  page?: number;
  per_page?: number;
};

function toQueryString(query: ItemQuery): string {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(query)) {
    if (value !== undefined && value !== null && value !== '') {
      params.append(key, String(value));
    }
  }
  const s = params.toString();
  return s ? `?${s}` : '';
}

export const items = {
  list: (query: ItemQuery = {}) => api.get<ItemList>(`/items${toQueryString(query)}`),

  /** Public listing — no auth header needed, but sending one is harmless. */
  get: (id: string) => api.get<Item>(`/items/${id}`),

  mine: (query: ItemQuery = {}) => api.get<ItemList>(`/users/me/items${toQueryString(query)}`),

  create: (payload: CreateItemPayload) => api.post<Item>('/items', payload),

  update: (id: string, payload: Partial<CreateItemPayload>) =>
    api.put<Item>(`/items/${id}`, payload),

  archive: (id: string) => api.delete<{ message: string }>(`/items/${id}`),
};

export const ratings = {
  forUser: (userId: string) => api.get<UserRating>(`/users/${userId}/rating`),
};
