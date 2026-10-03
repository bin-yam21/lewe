import { useState, type FormEvent } from 'react'
import { api } from '../api'
import { Badge, Empty, ErrorBox, Field, Spinner } from '../components/ui'
import { label, useAsync, useCatalog } from '../hooks'
import { confirmAction, haptic } from '../telegram'

export default function Wants() {
  const catalog = useCatalog()
  const { data, loading, error, reload } = useAsync(() => api.wants('active'), [])
  const [open, setOpen] = useState(false)
  const [category, setCategory] = useState('')
  const [keywords, setKeywords] = useState('')
  const [minCondition, setMinCondition] = useState('')
  const [formError, setFormError] = useState<string | null>(null)

  async function add(e: FormEvent) {
    e.preventDefault()
    setFormError(null)
    try {
      await api.createWant({
        category: category || catalog.categories[0],
        keywords: keywords.split(',').map((k) => k.trim()).filter(Boolean),
        min_condition: minCondition || null,
      })
      haptic('success')
      setOpen(false)
      setKeywords('')
      setMinCondition('')
      reload()
    } catch (err) {
      haptic('error')
      setFormError(err instanceof Error ? err.message : 'Could not save')
    }
  }

  async function remove(id: string) {
    if (!(await confirmAction('Remove this want?'))) return
    await api.deleteWant(id)
    reload()
  }

  return (
    <div className="stack">
      <div className="row">
        <h2 className="grow">What I want</h2>
        <button className="btn small primary" onClick={() => setOpen(!open)}>{open ? 'Close' : '+ Add want'}</button>
      </div>
      <p className="muted small-text">
        Lewe matches your wants against other people’s items and suggests swaps when they want something you have.
      </p>
      {open && (
        <form onSubmit={add} className="card stack">
          <Field label="Category">
            <select value={category || catalog.categories[0]} onChange={(e) => setCategory(e.target.value)}>
              {catalog.categories.map((c) => <option key={c} value={c}>{label(c)}</option>)}
            </select>
          </Field>
          <Field label="Keywords (comma separated, optional)">
            <input value={keywords} onChange={(e) => setKeywords(e.target.value)} placeholder="e.g. guitar, acoustic" />
          </Field>
          <Field label="Minimum condition">
            <select value={minCondition} onChange={(e) => setMinCondition(e.target.value)}>
              <option value="">Any</option>
              {catalog.conditions.map((c) => <option key={c} value={c}>{label(c)} or better</option>)}
            </select>
          </Field>
          {formError && <div className="card error">{formError}</div>}
          <button className="btn primary">Save want</button>
        </form>
      )}
      {error && <ErrorBox message={error} retry={reload} />}
      {loading && !data && <Spinner />}
      {data && data.data.length === 0 && <Empty title="No wants yet" hint="Tell Lewe what you’re looking for." />}
      {data?.data.map((w) => (
        <div key={w.id} className="card row">
          <div className="grow">
            <strong>{label(w.category)}</strong>
            <div className="muted small-text">
              {w.keywords.length > 0 ? w.keywords.join(', ') : 'Any item'}
              {w.min_condition && ` · ${label(w.min_condition)}+`}
            </div>
          </div>
          <Badge status={w.status} />
          <button className="icon-btn" aria-label="Remove" onClick={() => remove(w.id)}>🗑</button>
        </div>
      ))}
    </div>
  )
}
