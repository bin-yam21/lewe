import { api } from './client';
import type {
  AuthResponse,
  CreateItemPayload,
  Item,
  ItemList,
  Match,
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

  /** Other people's active listings in the same category. */
  similar: (id: string) => api.get<ItemList>(`/items/${id}/similar`),
};

export const offers = {
  /**
   * Offer one of your items for someone else's, without waiting for the
   * matcher to pair you. The result is an ordinary pending match.
   */
  create: (targetItemId: string, offerItemId: string, message?: string) =>
    api.post<Match>(`/items/${targetItemId}/offers`, {
      offer_item_id: offerItemId,
      message: message?.trim() || undefined,
    }),
};

export const matches = {
  mine: () => api.get<{ matches: Match[] }>('/matches'),
  get: (id: string) => api.get<Match>(`/matches/${id}`),

  respond: (id: string, accept: boolean, exchangeMethod?: 'in_person' | 'shipping') =>
    api.post<Match>(`/matches/${id}/respond`, {
      accept,
      exchange_method: exchangeMethod,
    }),

  /** Marks your side of the exchange done; completes when both sides have. */
  complete: (id: string) => api.post<Match>(`/matches/${id}/complete`),

  cancel: (id: string) => api.post<Match>(`/matches/${id}/cancel`),
};

export const ratings = {
  forUser: (userId: string) => api.get<UserRating>(`/users/${userId}/rating`),
};

/**
 * Image upload is multipart, so it bypasses the JSON client but reuses its
 * token handling. Returns the host-relative path to store in `item.images`.
 */
export const uploads = {
  image: async (uri: string, mimeType = 'image/jpeg'): Promise<string> => {
    const name = uri.split('/').pop() || `photo.${mimeType.split('/')[1] ?? 'jpg'}`;

    const form = new FormData();
    // React Native's FormData takes this {uri, name, type} shape rather than a
    // Blob; it streams the file straight from disk.
    form.append('file', { uri, name, type: mimeType } as unknown as Blob);

    const { url } = await api.post<{ url: string }>('/uploads', undefined, {
      formData: form,
    });
    return url;
  },
};
