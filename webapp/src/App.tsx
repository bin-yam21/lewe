import { useEffect, useState, type FormEvent } from 'react';
import { api, emailSignIn, hasSession, telegramSignIn } from './api';
import { match, navigate, usePath } from './router';
import { SessionProvider, useSession } from './session';
import { inTelegram, launchRoute, requestMessagesPermission } from './telegram';
import type { Catalog, User } from './types';
import { Button, ErrorNote, Loading } from './ui';
import { Browse } from './screens/Browse';
import { ItemDetail } from './screens/ItemDetail';
import { ItemForm } from './screens/ItemForm';
import { Matches } from './screens/Matches';
import { MatchDetail } from './screens/MatchDetail';
import { Profile } from './screens/Profile';
import { WantForm } from './screens/WantForm';
import { Notifications } from './screens/Notifications';

type Boot = { user: User; catalog: Catalog } | { error: Error } | null;

export function App() {
  const [boot, setBoot] = useState<Boot>(null);
  const [needsLogin, setNeedsLogin] = useState(false);

  async function start(user?: User) {
    try {
      const [me, catalog] = await Promise.all([user ? Promise.resolve(user) : api.me(), api.catalog()]);
      setBoot({ user: me, catalog });
      const route = launchRoute();
      if (route) navigate(route, { replace: true });
      requestMessagesPermission();
    } catch (e) {
      setBoot({ error: e as Error });
    }
  }

  useEffect(() => {
    if (inTelegram) {
      telegramSignIn().then((r) => start(r.user)).catch((e) => setBoot({ error: e }));
    } else if (hasSession()) {
      start();
    } else {
      setNeedsLogin(true);
    }
  }, []);

  if (needsLogin && !boot) return <Welcome onSignedIn={(u) => { setNeedsLogin(false); start(u); }} />;
  if (!boot) return <Loading />;
  if ('error' in boot) {
    return (
      <main className="page">
        <ErrorNote error={boot.error} onRetry={() => window.location.reload()} />
      </main>
    );
  }
  return (
    <SessionProvider user={boot.user} catalog={boot.catalog}>
      <Routes />
    </SessionProvider>
  );
}

function Routes() {
  const path = usePath();
  const p = path.split('?')[0];
  let screen;
  let params: Record<string, string> | null;

  if (p === '/' || p === '') screen = <Browse />;
  else if (p === '/new') screen = <ItemForm />;
  else if ((params = match('/items/:id/edit', p))) screen = <ItemForm id={params.id} />;
  else if ((params = match('/items/:id', p))) screen = <ItemDetail id={params.id} key={params.id} />;
  else if (p === '/matches') screen = <Matches />;
  else if ((params = match('/matches/:id', p))) screen = <MatchDetail id={params.id} key={params.id} />;
  else if (p === '/me') screen = <Profile />;
  else if (p === '/wants/new') screen = <WantForm />;
  else if (p === '/notifications') screen = <Notifications />;
  else screen = <Browse />;

  useEffect(() => window.scrollTo(0, 0), [p]);

  const tab = p === '/' ? 'browse' : p === '/new' ? 'new' : p.startsWith('/matches') ? 'matches' : p === '/me' || p === '/notifications' || p === '/wants/new' ? 'me' : null;
  return (
    <>
      {screen}
      <TabBar active={tab} />
    </>
  );
}

const ICONS = {
  browse: <path d="M4 4h7v7H4zM13 4h7v7h-7zM4 13h7v7H4zM13 13h7v7h-7z" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinejoin="round" />,
  matches: <path d="M7 7h11l-3-3M17 17H6l3 3" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />,
  new: <path d="M12 5v14M5 12h14" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />,
  me: <path d="M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8Zm-7 8c0-3.3 3.1-5.5 7-5.5s7 2.2 7 5.5" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />,
};

function TabBar({ active }: { active: string | null }) {
  const { unread } = useSession();
  const tabs: [keyof typeof ICONS, string, string][] = [
    ['browse', 'Browse', '/'],
    ['matches', 'Swaps', '/matches'],
    ['new', 'List item', '/new'],
    ['me', 'Me', '/me'],
  ];
  return (
    <nav className="tabbar" aria-label="Main">
      {tabs.map(([key, label, href]) => (
        <button key={key} type="button" className={'tab' + (active === key ? ' active' : '')} aria-current={active === key ? 'page' : undefined} onClick={() => navigate(href)}>
          <svg viewBox="0 0 24 24" aria-hidden="true">{ICONS[key]}</svg>
          {label}
          {key === 'me' && unread > 0 && <span className="badge">{unread > 99 ? '99+' : unread}</span>}
        </button>
      ))}
    </nav>
  );
}

/** Shown only outside Telegram: explains how to open the app, with an email sign-in for development. */
function Welcome({ onSignedIn }: { onSignedIn: (u: User) => void }) {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const r = await emailSignIn(email, password);
      onSignedIn(r.user);
    } catch (err) {
      setError(err as Error);
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="welcome">
      <div className="mark" aria-hidden="true">ለ</div>
      <div>
        <h1>Lewe</h1>
        <p className="hint">Swap what you have for what you want. Open Lewe from its Telegram bot to sign in with your Telegram account.</p>
      </div>
      <form onSubmit={submit}>
        <p className="hint">Or sign in with email</p>
        <input id="email" type="email" autoComplete="email" placeholder="Email" value={email} onChange={(e) => setEmail(e.target.value)} required />
        <input id="password" type="password" autoComplete="current-password" placeholder="Password" value={password} onChange={(e) => setPassword(e.target.value)} required />
        {error && <ErrorNote error={error} />}
        <Button type="submit" busy={busy}>Sign in</Button>
      </form>
    </main>
  );
}
