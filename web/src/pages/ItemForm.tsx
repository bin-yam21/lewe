import { useEffect, useState, type FormEvent } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { api, ApiError } from '../api'
import { Field, Spinner } from '../components/ui'
import { haptic } from '../telegram'
import { label, useCatalog } from '../hooks'
import type { ItemInput } from '../types'

const empty: ItemInput = {
  title: '', description: '', category: '', condition: 'good',
  estimated_value: null, location: null, image_urls: [],
}

export default function ItemForm() {
  const { id } = useParams()
  const nav = useNavigate()
  const catalog = useCatalog()
  const [form, setForm] = useState<ItemInput>(empty)
  const [images, setImages] = useState('')
  const [loading, setLoading] = useState(Boolean(id))
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [fields, setFields] = useState<Record<string, string>>({})

  useEffect(() => {
    if (!id) return
    api.item(id).then((i) => {
      setForm({
        title: i.title, description: i.description, category: i.category, condition: i.condition,
        estimated_value: i.estimated_value ?? null, location: i.location ?? null, image_urls: i.image_urls,
      })
      setImages(i.image_urls.join('\n'))
      setLoading(false)
    }, (e) => { setError(e.message); setLoading(false) })
  }, [id])

  useEffect(() => {
    if (!form.category && catalog.categories.length) setForm((f) => ({ ...f, category: catalog.categories[0] }))
  }, [catalog.categories, form.category])

  const set = <K extends keyof ItemInput>(k: K, v: ItemInput[K]) => setForm((f) => ({ ...f, [k]: v }))

  async function submit(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    setFields({})
    const body = { ...form, image_urls: images.split('\n').map((s) => s.trim()).filter(Boolean) }
    try {
      const saved = id ? await api.updateItem(id, body) : await api.createItem(body)
      haptic('success')
      nav(`/items/${saved.id}`, { replace: true })
    } catch (err) {
      haptic('error')
      setError(err instanceof Error ? err.message : 'Could not save')
      if (err instanceof ApiError && err.fields) setFields(err.fields)
    } finally {
      setBusy(false)
    }
  }

  if (loading) return <Spinner />
  return (
    <form onSubmit={submit} className="stack">
      <h2>{id ? 'Edit item' : 'List an item'}</h2>
      <Field label="Title" error={fields.title}>
        <input value={form.title} onChange={(e) => set('title', e.target.value)} maxLength={120} required />
      </Field>
      <Field label="Description" error={fields.description}>
        <textarea rows={4} value={form.description} onChange={(e) => set('description', e.target.value)} />
      </Field>
      <div className="row">
        <Field label="Category" error={fields.category}>
          <select value={form.category} onChange={(e) => set('category', e.target.value)}>
            {catalog.categories.map((c) => <option key={c} value={c}>{label(c)}</option>)}
          </select>
        </Field>
        <Field label="Condition" error={fields.condition}>
          <select value={form.condition} onChange={(e) => set('condition', e.target.value)}>
            {catalog.conditions.map((c) => <option key={c} value={c}>{label(c)}</option>)}
          </select>
        </Field>
      </div>
      <div className="row">
        <Field label="Estimated value" error={fields.estimated_value}>
          <input
            type="number" min={0} inputMode="numeric" value={form.estimated_value ?? ''}
            onChange={(e) => set('estimated_value', e.target.value === '' ? null : Number(e.target.value))}
          />
        </Field>
        <Field label="Location" error={fields.location}>
          <input value={form.location ?? ''} onChange={(e) => set('location', e.target.value || null)} />
        </Field>
      </div>
      <Field label="Image URLs (one per line)" error={fields.image_urls}>
        <textarea rows={3} value={images} onChange={(e) => setImages(e.target.value)} placeholder="https://…" />
      </Field>
      {error && <div className="card error">{error}</div>}
      <button className="btn primary" disabled={busy}>{id ? 'Save changes' : 'List item'}</button>
    </form>
  )
}
