import { useQueryClient } from '@tanstack/react-query';
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';

import { setSessionExpiredHandler } from '@/api/client';
import { auth } from '@/api/endpoints';
import type { User } from '@/api/types';
import { tokenStore } from '@/lib/storage';

type AuthState = {
  user: User | null;
  /** True until the stored session has been checked — gates the splash screen. */
  restoring: boolean;
  signIn: (email: string, password: string) => Promise<void>;
  signUp: (email: string, password: string, fullName: string) => Promise<void>;
  signOut: () => Promise<void>;
  setUser: (user: User) => void;
};

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [restoring, setRestoring] = useState(true);
  const queryClient = useQueryClient();

  const signOut = useCallback(async () => {
    await tokenStore.clear();
    setUser(null);
    // Otherwise the next account to sign in briefly sees the previous one's data.
    queryClient.clear();
  }, [queryClient]);

  // A refresh failure deep inside any request lands here.
  useEffect(() => {
    setSessionExpiredHandler(() => {
      setUser(null);
      queryClient.clear();
    });
    return () => setSessionExpiredHandler(null);
  }, [queryClient]);

  // Restore the session on cold launch: if a token survives, prove it still
  // works by fetching the profile rather than trusting its presence.
  useEffect(() => {
    let cancelled = false;

    (async () => {
      try {
        const { accessToken, refreshToken } = await tokenStore.get();
        if (!accessToken && !refreshToken) return;

        const me = await auth.me();
        if (!cancelled) setUser(me);
      } catch {
        if (!cancelled) await tokenStore.clear();
      } finally {
        if (!cancelled) setRestoring(false);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, []);

  const signIn = useCallback(async (email: string, password: string) => {
    const res = await auth.login(email.trim(), password);
    await tokenStore.save(res.access_token, res.refresh_token);
    setUser(res.user);
  }, []);

  const signUp = useCallback(async (email: string, password: string, fullName: string) => {
    const res = await auth.register(email.trim(), password, fullName.trim());
    await tokenStore.save(res.access_token, res.refresh_token);
    setUser(res.user);
  }, []);

  const value = useMemo<AuthState>(
    () => ({ user, restoring, signIn, signUp, signOut, setUser }),
    [user, restoring, signIn, signUp, signOut],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used inside <AuthProvider>');
  return ctx;
}
