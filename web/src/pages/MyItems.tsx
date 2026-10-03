import { Link } from 'react-router-dom'
import { api } from '../api'
import { Empty, ErrorBox, ItemRow, Spinner } from '../components/ui'
import { useAsync } from '../hooks'

export default function MyItems() {
  const { data, loading, error, reload } = useAsync(() => api.myItems(), [])
  return (
    <div className="stack">
      <div className="row">
        <h2 className="grow">My items</h2>
        <Link to="/items/new" className="btn small primary">+ List item</Link>
      </div>
      {error && <ErrorBox message={error} retry={reload} />}
      {loading && !data && <Spinner />}
      {data && data.data.length === 0 && (
        <Empty title="No items yet" hint="List something you’d give away to start getting matches." />
      )}
      {data?.data.map((i) => <ItemRow key={i.id} item={i} to={`/items/${i.id}`} />)}
    </div>
  )
}
