export interface User {
  id: string
  email: string
  full_name: string
  phone?: string
  location?: string
  bio?: string
  avatar_url?: string
  email_verified: boolean
  created_at: string
}

export interface AuthResponse {
  access_token: string
  refresh_token: string
  user: User
}

export interface PublicProfile {
  id: string
  full_name: string
  location?: string
  bio?: string
  avatar_url?: string
  rating: { average: number; count: number }
  created_at: string
}

export interface Item {
  id: string
  owner_id: string
  title: string
  description: string
  category: string
  condition: string
  estimated_value?: number
  location?: string
  image_urls: string[]
  status: 'available' | 'reserved' | 'exchanged' | 'withdrawn'
  created_at: string
  updated_at: string
}

export interface ItemInput {
  title: string
  description: string
  category: string
  condition: string
  estimated_value: number | null
  location: string | null
  image_urls: string[]
}

export interface Want {
  id: string
  category: string
  keywords: string[]
  min_condition?: string
  status: 'active' | 'fulfilled' | 'cancelled'
  created_at: string
}

export interface WantInput {
  category: string
  keywords: string[]
  min_condition: string | null
}

export interface ItemSummary {
  id: string
  title: string
  category: string
  condition: string
  estimated_value?: number
  image_urls: string[]
}

export type MatchStatus = 'pending' | 'accepted' | 'completed' | 'declined' | 'cancelled'

export interface Match {
  id: string
  status: MatchStatus
  score: number
  your_item: ItemSummary
  their_item: ItemSummary
  other_user: { id: string; full_name: string }
  you_accepted: boolean
  they_accepted: boolean
  exchange: { method: string; details?: string; proposed_by_you: boolean } | null
  you_completed: boolean
  they_completed: boolean
  you_rated: boolean
  created_at: string
  updated_at: string
  completed_at?: string
}

export interface Message {
  id: string
  sender_id: string
  from_you: boolean
  body: string
  created_at: string
}

export interface Rating {
  id: string
  match_id: string
  rater: { id: string; full_name: string }
  score: number
  comment?: string
  created_at: string
}

export interface Notification {
  id: string
  type: string
  message: string
  match_id?: string
  read: boolean
  created_at: string
}

export interface Catalog {
  categories: string[]
  conditions: string[]
  exchange_methods: string[]
}

export interface List<T> {
  data: T[]
  limit: number
  offset: number
}
