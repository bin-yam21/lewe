// Thin wrapper over window.Telegram.WebApp so the app also runs in a plain browser.
interface WebApp {
  initData: string
  ready(): void
  expand(): void
  close(): void
  colorScheme: 'light' | 'dark'
  HapticFeedback?: {
    impactOccurred(style: string): void
    notificationOccurred(type: 'error' | 'success' | 'warning'): void
  }
  BackButton: { show(): void; hide(): void; onClick(cb: () => void): void; offClick(cb: () => void): void }
  showConfirm?(message: string, cb: (ok: boolean) => void): void
  showAlert?(message: string, cb?: () => void): void
}

declare global {
  interface Window {
    Telegram?: { WebApp?: WebApp }
  }
}

export const tg: WebApp | undefined = window.Telegram?.WebApp

export const inTelegram = Boolean(tg && tg.initData)

export function initTelegram() {
  tg?.ready()
  tg?.expand()
}

export function haptic(type: 'success' | 'error' | 'warning' = 'success') {
  tg?.HapticFeedback?.notificationOccurred(type)
}

export function confirmAction(message: string): Promise<boolean> {
  return new Promise((resolve) => {
    if (tg?.showConfirm) tg.showConfirm(message, resolve)
    else resolve(window.confirm(message))
  })
}
