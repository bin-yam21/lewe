import { useEffect, useState } from 'react';

// Hash routing: bot message buttons open URLs like https://…/#/matches/<id>,
// and a hash route never needs server-side rewrites.

function current(): string {
  const h = window.location.hash.replace(/^#/, '');
  return h.startsWith('/') ? h : '/';
}

export function navigate(path: string, opts: { replace?: boolean } = {}) {
  if (opts.replace) window.location.replace('#' + path);
  else window.location.hash = path;
}

export function goBack() {
  if (window.history.length > 1) window.history.back();
  else navigate('/', { replace: true });
}

export function usePath(): string {
  const [path, setPath] = useState(current);
  useEffect(() => {
    const on = () => setPath(current());
    window.addEventListener('hashchange', on);
    return () => window.removeEventListener('hashchange', on);
  }, []);
  return path;
}

/** Match "/items/:id" against a path; returns params or null. */
export function match(pattern: string, path: string): Record<string, string> | null {
  const p = pattern.split('/');
  const s = path.split('?')[0].split('/');
  if (p.length !== s.length) return null;
  const params: Record<string, string> = {};
  for (let i = 0; i < p.length; i++) {
    if (p[i].startsWith(':')) params[p[i].slice(1)] = decodeURIComponent(s[i]);
    else if (p[i] !== s[i]) return null;
  }
  return params;
}

export function query(path: string): URLSearchParams {
  return new URLSearchParams(path.split('?')[1] ?? '');
}
