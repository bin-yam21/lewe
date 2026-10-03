import { useCallback, useEffect, useState } from 'react'
import { api } from './api'
import type { Catalog } from './types'

// Runs an async loader and tracks loading/error; `reload` refetches.
export function useAsync<T>(fn: () => Promise<T>, deps: unknown[]) {
  const [data, setData] = useState<T | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [tick, setTick] = useState(0)

  useEffect(() => {
    let live = true
    setLoading(true)
    fn().then(
      (d) => { if (live) { setData(d); setError(null); setLoading(false) } },
      (e) => { if (live) { setError(e instanceof Error ? e.message : String(e)); setLoading(false) } },
    )
    return () => { live = false }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, tick])

  const reload = useCallback(() => setTick((t) => t + 1), [])
  return { data, error, loading, reload, setData }
}

let catalogPromise: Promise<Catalog> | null = null
export function useCatalog(): Catalog {
  const [c, setC] = useState<Catalog>({ categories: [], conditions: [], exchange_methods: [] })
  useEffect(() => {
    catalogPromise ??= api.catalog()
    catalogPromise.then(setC).catch(() => { catalogPromise = null })
  }, [])
  return c
}

export const label = (s: string) => s.replace(/_/g, ' ').replace(/^./, (c) => c.toUpperCase())

export function timeAgo(iso: string) {
  const s = Math.max(1, Math.floor((Date.now() - new Date(iso).getTime()) / 1000))
  if (s < 60) return 'just now'
  const m = Math.floor(s / 60)
  if (m < 60) return `${m}m ago`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}h ago`
  const d = Math.floor(h / 24)
  return d < 30 ? `${d}d ago` : new Date(iso).toLocaleDateString()
}
