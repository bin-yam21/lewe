import { useNavigate } from 'react-router-dom'
import { api } from '../api'
import { Empty, ErrorBox, Spinner } from '../components/ui'
import { timeAgo, useAsync } from '../hooks'
import type { Notification } from '../types'

export default function Notifications() {
  const nav = useNavigate()
  const { data, loading, error, reload } = useAsync(() => api.notifications(), [])

  async function open(n: Notification) {
    if (!n.read) await api.readOne(n.id).catch(() => {})
    if (n.match_id) nav(`/matches/${n.match_id}`)
    else reload()
  }

  async function readAll() {
    await api.readAll()
    reload()
  }

  return (
    <div className="stack">
      <div className="row">
        <h2 className="grow">Notifications</h2>
        {data?.data.some((n) => !n.read) && <button className="btn small" onClick={readAll}>Mark all read</button>}
      </div>
      {error && <ErrorBox message={error} retry={reload} />}
      {loading && !data && <Spinner />}
      {data && data.data.length === 0 && <Empty title="All caught up" />}
      {data?.data.map((n) => (
        <button key={n.id} className={`card notif ${n.read ? '' : 'unread'}`} onClick={() => open(n)}>
          <span>{n.message}</span>
          <small className="muted">{timeAgo(n.created_at)}</small>
        </button>
      ))}
    </div>
  )
}
