import { useEffect, useRef, useState, type FormEvent } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api } from '../api'
import { Badge, ErrorBox, Field, Spinner, Stars, Thumb } from '../components/ui'
import { label, timeAgo, useAsync, useCatalog } from '../hooks'
import { confirmAction, haptic } from '../telegram'
import type { Match } from '../types'
import { matchHint } from './Matches'

export default function MatchDetail() {
  const { id = '' } = useParams()
  const { data: m, loading, error, reload, setData } = useAsync(() => api.match(id), [id])
  const [actionError, setActionError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  if (loading && !m) return <Spinner />
  if (error || !m) return <ErrorBox message={error ?? 'Match not found'} retry={reload} />

  // Runs a match action and swaps in the updated match it returns.
  async function act(fn: () => Promise<Match>, confirm?: string) {
    if (confirm && !(await confirmAction(confirm))) return
    setBusy(true)
    setActionError(null)
    try {
      setData(await fn())
      haptic('success')
    } catch (e) {
      haptic('error')
      setActionError(e instanceof Error ? e.message : 'Action failed')
    } finally {
      setBusy(false)
    }
  }

  const live = m.status === 'pending' || m.status === 'accepted'
  return (
    <div className="stack">
      <div className="row">
        <h2 className="grow">Swap with {m.other_user.full_name}</h2>
        <Badge status={m.status} />
      </div>
      <p className="muted">{matchHint(m)}</p>

      <div className="card swap-detail">
        <Link to={`/items/${m.your_item.id}`}><Thumb item={m.your_item} /><small>You give</small><b>{m.your_item.title}</b></Link>
        <span>⇄</span>
        <Link to={`/items/${m.their_item.id}`}><Thumb item={m.their_item} /><small>You get</small><b>{m.their_item.title}</b></Link>
      </div>
      <Link to={`/users/${m.other_user.id}`} className="muted small-text">View {m.other_user.full_name}’s profile →</Link>

      {actionError && <div className="card error">{actionError}</div>}

      {m.status === 'pending' && (
        <div className="row">
          {!m.you_accepted && (
            <button className="btn primary grow" disabled={busy} onClick={() => act(() => api.accept(m.id))}>Accept</button>
          )}
          <button className="btn danger" disabled={busy} onClick={() => act(() => api.decline(m.id), 'Decline this swap?')}>
            Decline
          </button>
        </div>
      )}

      {m.status === 'accepted' && (
        <>
          <ExchangeCard m={m} onSaved={setData} />
          <div className="row">
            {m.exchange && !m.you_completed && (
              <button
                className="btn primary grow" disabled={busy}
                onClick={() => act(() => api.complete(m.id), 'Confirm the hand-over happened?')}
              >
                Confirm hand-over
              </button>
            )}
            <button className="btn danger" disabled={busy} onClick={() => act(() => api.cancel(m.id), 'Cancel this swap?')}>
              Cancel swap
            </button>
          </div>
        </>
      )}

      {m.status === 'completed' && !m.you_rated && <RateCard m={m} onRated={reload} />}

      {m.status !== 'declined' && m.status !== 'cancelled' && live && <Chat matchId={m.id} />}
      {m.status === 'completed' && <Chat matchId={m.id} />}
    </div>
  )
}

function ExchangeCard({ m, onSaved }: { m: Match; onSaved: (m: Match) => void }) {
  const catalog = useCatalog()
  const [method, setMethod] = useState(m.exchange?.method ?? '')
  const [details, setDetails] = useState(m.exchange?.details ?? '')
  const [editing, setEditing] = useState(!m.exchange)
  const [error, setError] = useState<string | null>(null)

  async function save(e: FormEvent) {
    e.preventDefault()
    setError(null)
    try {
      onSaved(await api.setExchange(m.id, method || catalog.exchange_methods[0], details.trim() || null))
      setEditing(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not save')
    }
  }

  if (!editing && m.exchange) {
    return (
      <div className="card stack tight">
        <div className="row">
          <strong className="grow">Exchange: {label(m.exchange.method)}</strong>
          <button className="btn small" onClick={() => setEditing(true)}>Change</button>
        </div>
        {m.exchange.details && <p className="pre">{m.exchange.details}</p>}
        <div className="muted small-text">
          {m.exchange.proposed_by_you ? 'Proposed by you' : 'Proposed by them'}
          {' · '}You {m.you_completed ? '✓' : '…'} · Them {m.they_completed ? '✓' : '…'}
        </div>
      </div>
    )
  }
  return (
    <form onSubmit={save} className="card stack">
      <strong>How will you exchange?</strong>
      <Field label="Method">
        <select value={method || catalog.exchange_methods[0]} onChange={(e) => setMethod(e.target.value)}>
          {catalog.exchange_methods.map((x) => <option key={x} value={x}>{label(x)}</option>)}
        </select>
      </Field>
      <Field label="Details (where / when)">
        <textarea rows={2} value={details} onChange={(e) => setDetails(e.target.value)} />
      </Field>
      {error && <div className="card error">{error}</div>}
      <button className="btn primary">Save</button>
    </form>
  )
}

function RateCard({ m, onRated }: { m: Match; onRated: () => void }) {
  const [score, setScore] = useState(5)
  const [comment, setComment] = useState('')
  const [error, setError] = useState<string | null>(null)

  async function submit(e: FormEvent) {
    e.preventDefault()
    try {
      await api.rate(m.id, score, comment.trim() || null)
      haptic('success')
      onRated()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not rate')
    }
  }
  return (
    <form onSubmit={submit} className="card stack">
      <strong>Rate {m.other_user.full_name}</strong>
      <Stars value={score} onChange={setScore} />
      <textarea rows={2} placeholder="Comment (optional)" value={comment} onChange={(e) => setComment(e.target.value)} />
      {error && <div className="card error">{error}</div>}
      <button className="btn primary">Submit rating</button>
    </form>
  )
}

function Chat({ matchId }: { matchId: string }) {
  const { data, reload } = useAsync(() => api.messages(matchId), [matchId])
  const [body, setBody] = useState('')
  const [error, setError] = useState<string | null>(null)
  const end = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const t = setInterval(reload, 10000)
    return () => clearInterval(t)
  }, [reload])
  useEffect(() => { end.current?.scrollIntoView({ block: 'nearest' }) }, [data?.data.length])

  async function send(e: FormEvent) {
    e.preventDefault()
    const text = body.trim()
    if (!text) return
    setError(null)
    try {
      await api.sendMessage(matchId, text)
      setBody('')
      reload()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not send')
    }
  }

  // The API returns newest first or oldest first depending on version; sort to be safe.
  const msgs = [...(data?.data ?? [])].sort((a, b) => a.created_at.localeCompare(b.created_at))
  return (
    <div className="stack tight">
      <h3>Chat</h3>
      <div className="chat">
        {msgs.length === 0 && <p className="muted center small-text">Say hello and agree on details.</p>}
        {msgs.map((x) => (
          <div key={x.id} className={`bubble ${x.from_you ? 'me' : 'them'}`}>
            {x.body}
            <small>{timeAgo(x.created_at)}</small>
          </div>
        ))}
        <div ref={end} />
      </div>
      {error && <div className="card error">{error}</div>}
      <form onSubmit={send} className="row">
        <input className="grow" value={body} onChange={(e) => setBody(e.target.value)} placeholder="Message…" maxLength={2000} />
        <button className="btn primary small" disabled={!body.trim()}>Send</button>
      </form>
    </div>
  )
}
