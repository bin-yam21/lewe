import { Link, useNavigate, useParams } from 'react-router-dom'
import { api } from '../api'
import { useAuth } from '../auth'
import { Badge, ErrorBox, Spinner } from '../components/ui'
import { label, useAsync } from '../hooks'
import { confirmAction } from '../telegram'

export default function ItemDetail() {
  const { id = '' } = useParams()
  const nav = useNavigate()
  const { user } = useAuth()
  const item = useAsync(() => api.item(id), [id])
  const owner = useAsync(async () => (item.data ? api.publicProfile(item.data.owner_id) : null), [item.data?.owner_id])

  if (item.loading && !item.data) return <Spinner />
  if (item.error || !item.data) return <ErrorBox message={item.error ?? 'Item not found'} />
  const i = item.data
  const mine = user?.id === i.owner_id

  async function remove() {
    if (!(await confirmAction('Withdraw this listing?'))) return
    await api.deleteItem(i.id)
    nav('/mine', { replace: true })
  }

  return (
    <div className="stack">
      {i.image_urls.length > 0 && (
        <div className="gallery">{i.image_urls.map((u) => <img key={u} src={u} alt="" />)}</div>
      )}
      <div className="row">
        <h2 className="grow">{i.title}</h2>
        <Badge status={i.status} />
      </div>
      <div className="muted">
        {label(i.category)} · {label(i.condition)}
        {i.estimated_value != null && ` · worth ~${i.estimated_value}`}
        {i.location && ` · ${i.location}`}
      </div>
      <p className="pre">{i.description}</p>
      {owner.data && (
        <Link to={`/users/${owner.data.id}`} className="card row">
          <div className="avatar">{owner.data.full_name.slice(0, 1)}</div>
          <div className="grow">
            <strong>{owner.data.full_name}</strong>
            <div className="muted small-text">
              {owner.data.rating.count > 0
                ? `★ ${owner.data.rating.average.toFixed(1)} (${owner.data.rating.count})`
                : 'No ratings yet'}
            </div>
          </div>
        </Link>
      )}
      {mine ? (
        <div className="row">
          <Link className="btn grow" to={`/items/${i.id}/edit`}>Edit</Link>
          <button className="btn danger" onClick={remove}>Withdraw</button>
        </div>
      ) : (
        <div className="card muted">
          Want this? Add a <Link to="/wants/new">want</Link> for {label(i.category)} and list something to offer —
          Lewe will find the swap.
        </div>
      )}
    </div>
  )
}
