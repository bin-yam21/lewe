import { useEffect, useState } from 'react'
import { api } from '../api'
import { ErrorBox, Empty, ItemRow, Spinner } from '../components/ui'
import { label, useAsync, useCatalog } from '../hooks'

export default function Browse() {
  const catalog = useCatalog()
  const [q, setQ] = useState('')
  const [debounced, setDebounced] = useState('')
  const [category, setCategory] = useState('')

  useEffect(() => {
    const t = setTimeout(() => setDebounced(q.trim()), 300)
    return () => clearTimeout(t)
  }, [q])

  const { data, loading, error, reload } = useAsync(
    () => api.items({ q: debounced, category, limit: 50 }),
    [debounced, category],
  )

  return (
    <div className="stack">
      <input className="search" placeholder="Search items…" value={q} onChange={(e) => setQ(e.target.value)} />
      <div className="chips">
        <button className={!category ? 'on' : ''} onClick={() => setCategory('')}>All</button>
        {catalog.categories.map((c) => (
          <button key={c} className={category === c ? 'on' : ''} onClick={() => setCategory(c)}>{label(c)}</button>
        ))}
      </div>
      {error && <ErrorBox message={error} retry={reload} />}
      {loading && !data && <Spinner />}
      {data && data.data.length === 0 && <Empty title="Nothing here yet" hint="Try another search or category." />}
      {data?.data.map((i) => <ItemRow key={i.id} item={i} to={`/items/${i.id}`} />)}
    </div>
  )
}
