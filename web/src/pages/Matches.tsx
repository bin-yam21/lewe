import { Link } from 'react-router-dom'
import { api } from '../api'
import { Badge, Empty, ErrorBox, Spinner, Thumb } from '../components/ui'
import { useAsync } from '../hooks'
import type { Match } from '../types'

export function matchHint(m: Match): string {
  switch (m.status) {
    case 'pending':
      return m.you_accepted ? `Waiting for ${m.other_user.full_name}` : m.they_accepted ? 'They accepted — your turn' : 'New match — review it'
    case 'accepted':
      if (!m.exchange) return 'Choose how to exchange'
      return m.you_completed ? 'Waiting for their confirmation' : 'Confirm once handed over'
    case 'completed':
      return m.you_rated ? 'Completed' : 'Completed — leave a rating'
    default:
      return ''
  }
}

export default function Matches() {
  const { data, loading, error, reload } = useAsync(() => api.matches(), [])
  const active = data?.data.filter((m) => m.status === 'pending' || m.status === 'accepted') ?? []
  const past = data?.data.filter((m) => m.status !== 'pending' && m.status !== 'accepted') ?? []

  const card = (m: Match) => (
    <Link key={m.id} to={`/matches/${m.id}`} className="card stack tight">
      <div className="swap">
        <Thumb item={m.your_item} />
        <span>⇄</span>
        <Thumb item={m.their_item} />
        <div className="grow right"><Badge status={m.status} /></div>
      </div>
      <div><strong>{m.your_item.title}</strong> for <strong>{m.their_item.title}</strong></div>
      <div className="muted small-text">with {m.other_user.full_name} · {matchHint(m)}</div>
    </Link>
  )

  return (
    <div className="stack">
      <h2>Matches</h2>
      {error && <ErrorBox message={error} retry={reload} />}
      {loading && !data && <Spinner />}
      {data && data.data.length === 0 && (
        <Empty
          title="No matches yet"
          hint="List an item and add a want. When someone has what you want and wants what you have, it shows up here."
        />
      )}
      {active.map(card)}
      {past.length > 0 && <h3 className="muted">Past</h3>}
      {past.map(card)}
    </div>
  )
}
