import { useParams } from 'react-router-dom'
import { api } from '../api'
import { ErrorBox, ItemRow, Spinner, Stars } from '../components/ui'
import { timeAgo, useAsync } from '../hooks'

export default function UserProfile() {
  const { id = '' } = useParams()
  const user = useAsync(() => api.publicProfile(id), [id])
  const ratings = useAsync(() => api.userRatings(id), [id])
  const items = useAsync(() => api.items({ owner_id: id, limit: 50 }), [id])

  if (user.loading && !user.data) return <Spinner />
  if (user.error || !user.data) return <ErrorBox message={user.error ?? 'User not found'} />
  const u = user.data
  return (
    <div className="stack">
      <div className="center stack tight">
        <div className="avatar big">{u.full_name.slice(0, 1)}</div>
        <h2>{u.full_name}</h2>
        {u.location && <div className="muted">{u.location}</div>}
        <div>
          <Stars value={u.rating.average} /> <span className="muted">{u.rating.count > 0 ? `${u.rating.average.toFixed(1)} (${u.rating.count})` : 'No ratings yet'}</span>
        </div>
      </div>
      {u.bio && <p className="pre">{u.bio}</p>}
      {items.data && items.data.data.length > 0 && <h3>Items</h3>}
      {items.data?.data.map((i) => <ItemRow key={i.id} item={i} to={`/items/${i.id}`} />)}
      {ratings.data && ratings.data.data.length > 0 && <h3>Reviews</h3>}
      {ratings.data?.data.map((r) => (
        <div key={r.id} className="card stack tight">
          <div className="row"><Stars value={r.score} /><span className="muted small-text">{r.rater.full_name} · {timeAgo(r.created_at)}</span></div>
          {r.comment && <p>{r.comment}</p>}
        </div>
      ))}
    </div>
  )
}
