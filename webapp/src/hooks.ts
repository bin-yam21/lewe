import { useCallback, useEffect, useRef, useState } from 'react';
import { tg } from './telegram';

/** Load data with an async function; re-runs when deps change. */
export function useAsync<T>(fn: () => Promise<T>, deps: unknown[]) {
  const [data, setData] = useState<T | undefined>(undefined);
  const [error, setError] = useState<Error | null>(null);
  const [loading, setLoading] = useState(true);
  const seq = useRef(0);

  const reload = useCallback(async () => {
    const id = ++seq.current;
    setLoading(true);
    try {
      const value = await fn();
      if (id === seq.current) {
        setData(value);
        setError(null);
      }
    } catch (e) {
      if (id === seq.current) setError(e as Error);
    } finally {
      if (id === seq.current) setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);

  useEffect(() => {
    reload();
  }, [reload]);

  return { data, error, loading, reload, setData };
}

/** Call fn every `ms` while the page is visible. */
export function useInterval(fn: () => void, ms: number) {
  const saved = useRef(fn);
  saved.current = fn;
  useEffect(() => {
    const id = setInterval(() => {
      if (document.visibilityState === 'visible') saved.current();
    }, ms);
    return () => clearInterval(id);
  }, [ms]);
}

/**
 * Drive Telegram's bottom MainButton. Outside Telegram it returns false and
 * the screen renders its own button instead.
 */
export function useMainButton(text: string, onClick: () => void, opts: { enabled?: boolean; busy?: boolean; visible?: boolean } = {}) {
  const { enabled = true, busy = false, visible = true } = opts;
  const handler = useRef(onClick);
  handler.current = onClick;
  const native = Boolean(tg?.initData);

  useEffect(() => {
    if (!native || !tg) return;
    const mb = tg.MainButton;
    const cb = () => handler.current();
    mb.onClick(cb);
    return () => {
      mb.offClick(cb);
      mb.hide();
    };
  }, [native]);

  useEffect(() => {
    if (!native || !tg) return;
    const mb = tg.MainButton;
    mb.setText(text);
    if (visible) mb.show();
    else mb.hide();
    if (enabled && !busy) mb.enable();
    else mb.disable();
    if (busy) mb.showProgress(false);
    else mb.hideProgress();
  }, [native, text, enabled, busy, visible]);

  return native;
}

/** Show Telegram's BackButton while mounted. */
export function useBackButton(onBack: (() => void) | null) {
  const handler = useRef(onBack);
  handler.current = onBack;
  useEffect(() => {
    if (!tg?.initData || !onBack) return;
    const bb = tg.BackButton;
    const cb = () => handler.current?.();
    bb.onClick(cb);
    bb.show();
    return () => {
      bb.offClick(cb);
      bb.hide();
    };
  }, [onBack !== null]); // eslint-disable-line react-hooks/exhaustive-deps
}
