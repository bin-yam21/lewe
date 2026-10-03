import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { label } from '../hooks'
import type { Item, ItemSummary } from '../types'

export function Spinner() {
  return <div className="center muted">Loading…</div>
}

export function ErrorBox({ message, retry }: { message: string; retry?: () => void }) {
  return (
    <div className="card error">
      <p>{message}</p>
      {retry && <button className="btn small" onClick={retry}>Try again</button>}
    </div>
  )
}

export function Empty({ title, hint, children }: { title: string; hint?: string; children?: ReactNode }) {
  return (
    <div className="empty">
      <h3>{title}</h3>
      {hint && <p className="muted">{hint}</p>}
      {children}
    </div>
  )
}

export function Badge({ status }: { status: string }) {
  return <span className={`badge ${status}`}>{label(status)}</span>
}

export function Thumb({ item }: { item: Pick<ItemSummary, 'image_urls' | 'title' | 'category'> }) {
  const src = item.image_urls[0]
  return src ? (
    <img className="thumb" src={src} alt="" loading="lazy" />
  ) : (
    <div className="thumb ph">{item.category.slice(0, 1).toUpperCase()}</div>
  )
}

export function ItemRow({ item, to }: { item: Item; to: string }) {
  return (
    <Link to={to} className="row card">
      <Thumb item={item} />
      <div className="grow">
        <strong>{item.title}</strong>
        <div className="muted small-text">
          {label(item.category)} · {label(item.condition)}
          {item.estimated_value != null && ` · ~${item.estimated_value}`}
        </div>
      </div>
      {item.status !== 'available' && <Badge status={item.status} />}
    </Link>
  )
}

export function Field({ label: l, error, children }: { label: string; error?: string; children: ReactNode }) {
  return (
    <label className="field">
      <span>{l}</span>
      {children}
      {error && <em className="err">{error}</em>}
    </label>
  )
}

export function Stars({ value, onChange }: { value: number; onChange?: (n: number) => void }) {
  return (
    <span className="stars">
      {[1, 2, 3, 4, 5].map((n) => (
        <button
          key={n}
          type="button"
          disabled={!onChange}
          className={n <= Math.round(value) ? 'on' : ''}
          onClick={() => onChange?.(n)}
          aria-label={`${n} star${n > 1 ? 's' : ''}`}
        >
          ★
        </button>
      ))}
    </span>
  )
}
