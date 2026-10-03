import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react'
import { api, setUnauthorizedHandler, tokens } from './api'
import { inTelegram, tg } from './telegram'
import type { AuthResponse, User } from './types'

interface AuthState {
  user: User | null
  loading: boolean
  error: string | null
  signIn(a: AuthResponse): void
  signOut(): Promise<void>
  setUser(u: User): void
}

const Ctx = createContext<AuthState>(null!)
export const useAuth = () => useContext(Ctx)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const signIn = useCallback((a: AuthResponse) => {
    tokens.set(a)
    setUser(a.user)
  }, [])

  const signOut = useCallback(async () => {
    const refresh = tokens.refresh
    tokens.clear()
    setUser(null)
    if (refresh) await api.logout(refresh).catch(() => {})
  }, [])

  useEffect(() => {
    setUnauthorizedHandler(() => setUser(null))
    ;(async () => {
      try {
        // Existing session first, then Telegram's signed launch data.
        if (tokens.access || tokens.refresh) {
          try {
            setUser(await api.me())
            return
          } catch { tokens.clear() }
        }
        if (inTelegram && tg) signIn(await api.telegramLogin(tg.initData))
      } catch (e) {
        setError(e instanceof Error ? e.message : 'Sign-in failed')
      } finally {
        setLoading(false)
      }
    })()
  }, [signIn])

  return <Ctx.Provider value={{ user, loading, error, signIn, signOut, setUser }}>{children}</Ctx.Provider>
}
