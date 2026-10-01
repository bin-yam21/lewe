import type { ReactNode } from 'react';
import { CONDITION_LABELS, type Condition, type MatchStatus } from './types';
import { haptic } from './telegram';

export function Page({ title, subtitle, children, action }: { title?: string; subtitle?: ReactNode; children: ReactNode; action?: ReactNode }) {
  return (
    <main className="page">
      {title && (
        <header className="page-head">
          <div>
            <h1>{title}</h1>
            {subtitle && <p className="hint">{subtitle}</p>}
          </div>
          {action}
        </header>
      )}
      {children}
    </main>
  );
}

export function Section({ title, footer, children }: { title?: string; footer?: ReactNode; children: ReactNode }) {
  return (
    <section className="section">
      {title && <h2 className="section-title">{title}</h2>}
      <div className="section-body">{children}</div>
      {footer && <p className="section-footer">{footer}</p>}
    </section>
  );
}

export function Cell({ title, subtitle, before, after, onClick }: { title: ReactNode; subtitle?: ReactNode; before?: ReactNode; after?: ReactNode; onClick?: () => void }) {
  const Tag = onClick ? 'button' : 'div';
  return (
    <Tag className={'cell' + (onClick ? ' tappable' : '')} onClick={onClick} type={onClick ? 'button' : undefined}>
      {before && <span className="cell-before">{before}</span>}
      <span className="cell-main">
        <span className="cell-title">{title}</span>
        {subtitle && <span className="cell-sub">{subtitle}</span>}
      </span>
      {after && <span className="cell-after">{after}</span>}
    </Tag>
  );
}

export function Button({ children, onClick, kind = 'primary', disabled, busy, type = 'button' }: {
  children: ReactNode; onClick?: () => void; kind?: 'primary' | 'secondary' | 'danger' | 'plain'; disabled?: boolean; busy?: boolean; type?: 'button' | 'submit';
}) {
  return (
    <button
      type={type}
      className={`btn btn-${kind}`}
      disabled={disabled || busy}
      onClick={() => {
        haptic.tap();
        onClick?.();
      }}
    >
      {busy ? <Spinner small /> : children}
    </button>
  );
}

export function Chip({ active, onClick, children }: { active?: boolean; onClick: () => void; children: ReactNode }) {
  return (
    <button type="button" className={'chip' + (active ? ' active' : '')} aria-pressed={active} onClick={() => { haptic.select(); onClick(); }}>
      {children}
    </button>
  );
}

export function Spinner({ small }: { small?: boolean }) {
  return <span className={'spinner' + (small ? ' small' : '')} role="status" aria-label="Loading" />;
}

export function Loading() {
  return <div className="center"><Spinner /></div>;
}

export function Empty({ icon, title, children }: { icon: ReactNode; title: string; children?: ReactNode }) {
  return (
    <div className="empty">
      <div className="empty-icon" aria-hidden="true">{icon}</div>
      <h3>{title}</h3>
      {children && <div className="hint">{children}</div>}
    </div>
  );
}

export function ErrorNote({ error, onRetry }: { error: Error; onRetry?: () => void }) {
  return (
    <div className="error-note" role="alert">
      <span>{error.message}</span>
      {onRetry && <button type="button" className="link" onClick={onRetry}>Try again</button>}
    </div>
  );
}

export function Photo({ src, alt, className }: { src?: string; alt: string; className?: string }) {
  if (!src) {
    return (
      <div className={'photo placeholder ' + (className ?? '')} aria-label={alt}>
        <svg viewBox="0 0 24 24" width="28" height="28" aria-hidden="true"><path fill="currentColor" d="M4 5h3l1.5-2h7L17 5h3a2 2 0 0 1 2 2v11a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2Zm8 3.5a4.5 4.5 0 1 0 0 9 4.5 4.5 0 0 0 0-9Zm0 2a2.5 2.5 0 1 1 0 5 2.5 2.5 0 0 1 0-5Z"/></svg>
      </div>
    );
  }
  return <img className={'photo ' + (className ?? '')} src={src} alt={alt} loading="lazy" />;
}

export function Avatar({ name, src, size = 40 }: { name: string; src?: string; size?: number }) {
  const initials = name.split(/\s+/).filter(Boolean).slice(0, 2).map((w) => w[0]?.toUpperCase()).join('');
  const hue = [...name].reduce((h, c) => (h * 31 + c.charCodeAt(0)) % 360, 0);
  return src ? (
    <img className="avatar" src={src} alt="" width={size} height={size} style={{ width: size, height: size }} />
  ) : (
    <span className="avatar" style={{ width: size, height: size, fontSize: size * 0.4, background: `hsl(${hue} 55% 52%)` }} aria-hidden="true">
      {initials || '?'}
    </span>
  );
}

export function ConditionTag({ c }: { c: Condition }) {
  return <span className="tag">{CONDITION_LABELS[c]}</span>;
}

const STATUS: Record<MatchStatus, { label: string; tone: string }> = {
  pending: { label: 'New match', tone: 'accent' },
  accepted: { label: 'Arranging', tone: 'warn' },
  completed: { label: 'Swapped', tone: 'ok' },
  declined: { label: 'Declined', tone: 'muted' },
  cancelled: { label: 'Cancelled', tone: 'muted' },
};

export function StatusPill({ status }: { status: MatchStatus }) {
  const s = STATUS[status];
  return <span className={`pill tone-${s.tone}`}>{s.label}</span>;
}

export function Stars({ value, onChange, size = 28 }: { value: number; onChange?: (v: number) => void; size?: number }) {
  return (
    <div className="stars" role={onChange ? 'radiogroup' : 'img'} aria-label={`${value} out of 5`}>
      {[1, 2, 3, 4, 5].map((n) => (
        <button
          key={n}
          type="button"
          className={'star' + (n <= value ? ' on' : '')}
          style={{ fontSize: size }}
          disabled={!onChange}
          aria-label={`${n} star${n > 1 ? 's' : ''}`}
          role={onChange ? 'radio' : undefined}
          aria-checked={onChange ? n === value : undefined}
          onClick={() => { haptic.select(); onChange?.(n); }}
        >
          ★
        </button>
      ))}
    </div>
  );
}

export function timeAgo(iso: string) {
  const s = Math.max(1, Math.round((Date.now() - new Date(iso).getTime()) / 1000));
  if (s < 60) return 'just now';
  const m = Math.round(s / 60);
  if (m < 60) return `${m} min ago`;
  const h = Math.round(m / 60);
  if (h < 24) return `${h} h ago`;
  const d = Math.round(h / 24);
  if (d < 7) return `${d} d ago`;
  return new Date(iso).toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
}
