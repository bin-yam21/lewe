import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react';
import { api } from './api';
import { useInterval } from './hooks';
import type { Catalog, User } from './types';

type Session = {
  user: User;
  setUser: (u: User) => void;
  catalog: Catalog;
  unread: number;
  refreshUnread: () => void;
};

const Ctx = createContext<Session | null>(null);

export function useSession(): Session {
  const s = useContext(Ctx);
  if (!s) throw new Error('useSession outside SessionProvider');
  return s;
}

export function SessionProvider({ user: initial, catalog, children }: { user: User; catalog: Catalog; children: ReactNode }) {
  const [user, setUser] = useState(initial);
  const [unread, setUnread] = useState(0);

  const refreshUnread = useCallback(() => {
    api.unreadCount().then((r) => setUnread(r.unread)).catch(() => {});
  }, []);
  useEffect(refreshUnread, [refreshUnread]);
  useInterval(refreshUnread, 20000);

  return <Ctx.Provider value={{ user, setUser, catalog, unread, refreshUnread }}>{children}</Ctx.Provider>;
}
