// Mirrors the Go API's JSON. Field names must match the json tags exactly.

export type User = {
  id: string;
  email?: string;
  full_name: string;
  phone?: string;
  location?: string;
  bio?: string;
  avatar_url?: string;
  city?: string;
  telegram_username?: string;
  email_verified: boolean;
  created_at: string;
};

export type PublicProfile = {
  id: string;
  full_name: string;
  location?: string;
  bio?: string;
  avatar_url?: string;
  city?: string;
  rating: { average: number; count: number };
  created_at: string;
};

export type AuthResponse = { access_token: string; refresh_token: string; user: User };

export type List<T> = { data: T[]; limit: number; offset: number };

export type Condition = 'new' | 'like_new' | 'good' | 'fair' | 'poor';
export type ItemStatus = 'available' | 'reserved' | 'exchanged' | 'withdrawn';

export type Item = {
  id: string;
  owner_id: string;
  title: string;
  description: string;
  category: string;
  condition: Condition;
  estimated_value?: number;
  location?: string;
  image_urls: string[];
  status: ItemStatus;
  created_at: string;
  updated_at: string;
};

export type ItemInput = {
  title: string;
  description: string;
  category: string;
  condition: Condition;
  estimated_value?: number | null;
  location?: string | null;
  image_urls: string[];
};

export type Want = {
  id: string;
  category: string;
  keywords: string[];
  min_condition?: Condition;
  any_city: boolean;
  status: 'active' | 'fulfilled' | 'cancelled';
  created_at: string;
};

export type MatchStatus = 'pending' | 'accepted' | 'completed' | 'declined' | 'cancelled';
export type ExchangeMethod = 'meetup' | 'shipping' | 'dropoff';

export type ItemSummary = {
  id: string;
  title: string;
  category: string;
  condition: Condition;
  estimated_value?: number;
  image_urls: string[];
};

export type Match = {
  id: string;
  status: MatchStatus;
  score: number;
  your_item: ItemSummary;
  their_item: ItemSummary;
  other_user: { id: string; full_name: string };
  you_accepted: boolean;
  they_accepted: boolean;
  exchange: { method: ExchangeMethod; details?: string; proposed_by_you: boolean } | null;
  you_completed: boolean;
  they_completed: boolean;
  you_rated: boolean;
  created_at: string;
  updated_at: string;
  completed_at?: string;
};

export type Message = { id: string; sender_id: string; from_you: boolean; body: string; created_at: string };

export type Notification = {
  id: string;
  type: string;
  message: string;
  match_id?: string;
  read: boolean;
  created_at: string;
};

export type Catalog = {
  categories: string[];
  conditions: Condition[];
  exchange_methods: ExchangeMethod[];
  cities: string[];
};

export const CONDITION_LABELS: Record<Condition, string> = {
  new: 'New',
  like_new: 'Like new',
  good: 'Good',
  fair: 'Fair',
  poor: 'Worn',
};

export const METHOD_LABELS: Record<ExchangeMethod, string> = {
  meetup: 'Meet up',
  shipping: 'Ship it',
  dropoff: 'Drop off',
};

export function categoryLabel(c: string) {
  return c.charAt(0).toUpperCase() + c.slice(1);
}

export function formatBirr(n?: number) {
  return n == null ? '' : `${n.toLocaleString('en-US')} Br`;
}
