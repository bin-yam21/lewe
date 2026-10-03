import { useState, type FormEvent } from 'react'
import { api } from '../api'
import { useAuth } from '../auth'
import { Field } from '../components/ui'
import { inTelegram } from '../telegram'

// Email sign-in is a fallback for opening the app outside Telegram.
export default function Login() {
  const { signIn, error: bootError } = useAuth()
  const [mode, setMode] = useState<'login' | 'register'>('login')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [name, setName] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function submit(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      signIn(mode === 'login' ? await api.login(email, password) : await api.register(email, password, name))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="page narrow">
      <h1 className="hero">Lewe</h1>
      <p className="muted center">Swap what you have for what you need.</p>
      {inTelegram && bootError && <div className="card error">{bootError}</div>}
      <form onSubmit={submit} className="stack">
        {mode === 'register' && (
          <Field label="Full name"><input value={name} onChange={(e) => setName(e.target.value)} required /></Field>
        )}
        <Field label="Email">
          <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} autoComplete="email" required />
        </Field>
        <Field label="Password">
          <input
            type="password" value={password} minLength={mode === 'register' ? 8 : undefined}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete={mode === 'login' ? 'current-password' : 'new-password'} required
          />
        </Field>
        {error && <div className="card error">{error}</div>}
        <button className="btn primary" disabled={busy}>{mode === 'login' ? 'Sign in' : 'Create account'}</button>
        <button type="button" className="btn link" onClick={() => setMode(mode === 'login' ? 'register' : 'login')}>
          {mode === 'login' ? 'New here? Create an account' : 'Have an account? Sign in'}
        </button>
      </form>
    </div>
  )
}
