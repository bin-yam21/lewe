// A thin, typed layer over window.Telegram.WebApp. Everything degrades to a
// plain-browser fallback so the app also runs outside Telegram (development).

type Haptic = {
  impactOccurred(style: 'light' | 'medium' | 'heavy' | 'rigid' | 'soft'): void;
  notificationOccurred(type: 'error' | 'success' | 'warning'): void;
  selectionChanged(): void;
};

type BottomButton = {
  setText(text: string): void;
  show(): void;
  hide(): void;
  enable(): void;
  disable(): void;
  showProgress(leaveActive?: boolean): void;
  hideProgress(): void;
  onClick(cb: () => void): void;
  offClick(cb: () => void): void;
};

type WebApp = {
  initData: string;
  initDataUnsafe: { user?: { id: number; first_name: string; allows_write_to_pm?: boolean }; start_param?: string };
  version: string;
  platform: string;
  colorScheme: 'light' | 'dark';
  ready(): void;
  expand(): void;
  isVersionAtLeast(v: string): boolean;
  setHeaderColor?(color: string): void;
  setBackgroundColor?(color: string): void;
  BackButton: { show(): void; hide(): void; onClick(cb: () => void): void; offClick(cb: () => void): void };
  MainButton: BottomButton;
  HapticFeedback: Haptic;
  showConfirm(message: string, cb: (ok: boolean) => void): void;
  requestWriteAccess?(cb?: (granted: boolean) => void): void;
  openTelegramLink?(url: string): void;
};

declare global {
  interface Window {
    Telegram?: { WebApp?: WebApp };
  }
}

export const tg: WebApp | undefined = window.Telegram?.WebApp;

/** True when running inside Telegram with signed launch data. */
export const inTelegram = Boolean(tg && tg.initData);

export function initTelegram() {
  if (!tg) return;
  tg.ready();
  tg.expand();
  try {
    tg.setHeaderColor?.('secondary_bg_color');
    tg.setBackgroundColor?.('secondary_bg_color');
  } catch {
    // Older clients don't support these.
  }
}

/** Ask once for permission to send the user messages (match alerts). */
export function requestMessagesPermission() {
  if (!inTelegram || !tg?.requestWriteAccess) return;
  if (tg.initDataUnsafe.user?.allows_write_to_pm) return;
  if (!tg.isVersionAtLeast('6.9')) return;
  try {
    tg.requestWriteAccess();
  } catch {
    // Not supported on this client.
  }
}

export const haptic = {
  tap: () => tg?.HapticFeedback?.impactOccurred('light'),
  success: () => tg?.HapticFeedback?.notificationOccurred('success'),
  error: () => tg?.HapticFeedback?.notificationOccurred('error'),
  select: () => tg?.HapticFeedback?.selectionChanged(),
};

/** Telegram's native confirm dialog, or the browser's outside Telegram. */
export function confirmAction(message: string): Promise<boolean> {
  return new Promise((resolve) => {
    if (inTelegram && tg && tg.isVersionAtLeast('6.2')) {
      tg.showConfirm(message, resolve);
    } else {
      resolve(window.confirm(message));
    }
  });
}

/** Route to open on launch: a bot message button (#/matches/…) or a t.me start parameter. */
export function launchRoute(): string | null {
  const start = tg?.initDataUnsafe.start_param;
  if (start?.startsWith('match_')) return `/matches/${start.slice('match_'.length)}`;
  if (start?.startsWith('item_')) return `/items/${start.slice('item_'.length)}`;
  return null;
}
