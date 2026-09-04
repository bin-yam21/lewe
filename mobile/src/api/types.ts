/**
 * Mirrors the DTOs in the Go API. Keep field names identical to the JSON tags in
 * the `types.go` of each internal domain package — when they drift, the app
 * breaks silently at runtime.
 */

export type User = {
  id: string;
  email: string;
  full_name: string;
  phone?: string;
  location?: string;
  bio?: string;
  avatar_url?: string;
  created_at: string;
};

export type AuthResponse = {
  access_token: string;
  refresh_token: string;
  user: User;
};

export type ItemCondition = 'new' | 'like_new' | 'good' | 'fair' | 'poor';
export type ExchangeMethod = 'in_person' | 'shipping' | 'either';
export type ItemStatus = 'active' | 'matched' | 'exchanged' | 'archived';

export type Want = {
  id: string;
  category: string;
  description?: string;
};

export type Item = {
  id: string;
  user_id: string;
  title: string;
  description: string;
  category: string;
  condition: ItemCondition;
  exchange_method: ExchangeMethod;
  images: string[];
  location?: string;
  status: ItemStatus;
  wants: Want[];
  created_at: string;
  updated_at: string;
};

export type ItemList = {
  items: Item[];
  total: number;
  page: number;
  per_page: number;
};

export type CreateItemPayload = {
  title: string;
  description: string;
  category: string;
  condition: ItemCondition;
  exchange_method: ExchangeMethod;
  images?: string[];
  location?: string;
  wants: { category: string; description?: string }[];
};

export type UserRating = {
  user_id: string;
  average: number;
  count: number;
};

export const CONDITIONS: { value: ItemCondition; label: string }[] = [
  { value: 'new', label: 'New' },
  { value: 'like_new', label: 'Like new' },
  { value: 'good', label: 'Good' },
  { value: 'fair', label: 'Fair' },
  { value: 'poor', label: 'Poor' },
];

export const EXCHANGE_METHODS: { value: ExchangeMethod; label: string }[] = [
  { value: 'either', label: 'Either' },
  { value: 'in_person', label: 'In person' },
  { value: 'shipping', label: 'Shipping' },
];

/**
 * Categories are free text in the API today, which means a match is only found
 * when two users type the same string. Until `GET /categories` exists (roadmap
 * A1) the app ships a fixed list so the values are at least consistent between
 * everyone using the app.
 */
export const CATEGORIES = [
  'electronics',
  'books',
  'clothing',
  'furniture',
  'music',
  'sports',
  'games',
  'tools',
  'home',
  'art',
  'other',
] as const;

export function labelForCategory(value: string): string {
  return value.charAt(0).toUpperCase() + value.slice(1).replace(/_/g, ' ');
}

export function labelForCondition(value: ItemCondition): string {
  return CONDITIONS.find((c) => c.value === value)?.label ?? value;
}
