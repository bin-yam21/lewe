import { useState, type FormEvent } from 'react'
import { api, ApiError } from '../api'
import { useAuth } from '../auth'
import { Field } from '../components/ui'
import { haptic, inTelegram } from '../telegram'

export default function Profile() {
  const { user, setUser, signOut } = useAuth()
  const [name, setName] = useState(user?.full_name ?? '')
  const [phone, setPhone] = useState(user?.phone ?? '')
  const [location, setLocation] = useState(user?.location ?? '')
  const [bio, setBio] = useState(user?.bio ?? '')
  const [msg, setMsg] = useState<{ ok: boolean; text: string } | null>(null)
  const [fields, setFields] = useState<Record<string, string>>({})

  if (!user) return null

  async function save(e: FormEvent) {
    e.preventDefault()
    setFields({})
    try {
      setUser(await api.updateMe({
        full_name: name, phone: phone || undefined, location: location || undefined,
        bio: bio || undefined, avatar_url: user!.avatar_url,
      }))
      haptic('success')
      setMsg({ ok: true, text: 'Saved' })
    } catch (err) {
      haptic('error')
      setMsg({ ok: false, text: err instanceof Error ? err.message : 'Could not save' })
      if (err instanceof ApiError && err.fields) setFields(err.fields)
    }
  }

  return (
    <form onSubmit={save} className="stack">
      <h2>My profile</h2>
      <Field label="Name" error={fields.full_name}><input value={name} onChange={(e) => setName(e.target.value)} required /></Field>
      <Field label="Location" error={fields.location}><input value={location} onChange={(e) => setLocation(e.target.value)} /></Field>
      <Field label="Phone" error={fields.phone}><input value={phone} onChange={(e) => setPhone(e.target.value)} inputMode="tel" /></Field>
      <Field label="About me" error={fields.bio}><textarea rows={3} value={bio} onChange={(e) => setBio(e.target.value)} /></Field>
      {msg && <div className={`card ${msg.ok ? 'ok' : 'error'}`}>{msg.text}</div>}
      <button className="btn primary">Save</button>
      {!inTelegram && <button type="button" className="btn" onClick={signOut}>Sign out</button>}
    </form>
  )
}
