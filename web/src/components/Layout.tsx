import { useEffect } from 'react'
import { Link, NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { api } from '../api'
import { useAsync } from '../hooks'
import { tg } from '../telegram'

const tabs = [
  { to: '/', label: 'Browse', icon: '🔍', end: true },
  { to: '/matches', label: 'Matches', icon: '🔁' },
  { to: '/mine', label: 'My items', icon: '📦' },
  { to: '/wants', label: 'Wants', icon: '⭐' },
  { to: '/profile', label: 'Me', icon: '👤' },
]

export default function Layout() {
  const nav = useNavigate()
  const { pathname } = useLocation()
  const isRoot = tabs.some((t) => t.to === pathname)

  // Poll the unread count; the API has no push channel yet.
  const unread = useAsync(() => api.unreadCount(), [pathname])
  useEffect(() => {
    const t = setInterval(unread.reload, 30000)
    return () => clearInterval(t)
  }, [unread.reload])

  // Telegram's native back button on sub-pages.
  useEffect(() => {
    const webApp = tg
    if (!webApp) return
    const back = () => nav(-1)
    if (isRoot) webApp.BackButton.hide()
    else {
      webApp.BackButton.show()
      webApp.BackButton.onClick(back)
    }
    return () => webApp.BackButton.offClick(back)
  }, [isRoot, nav])

  const count = unread.data?.unread ?? 0
  return (
    <div className="shell">
      <header className="top">
        {!isRoot && <button className="icon-btn" onClick={() => nav(-1)} aria-label="Back">←</button>}
        <Link to="/" className="brand">Lewe</Link>
        <Link to="/notifications" className="icon-btn bell" aria-label="Notifications">
          🔔{count > 0 && <b>{count > 9 ? '9+' : count}</b>}
        </Link>
      </header>
      <main className="page"><Outlet /></main>
      <nav className="tabs">
        {tabs.map((t) => (
          <NavLink key={t.to} to={t.to} end={t.end} className={({ isActive }) => (isActive ? 'active' : '')}>
            <span>{t.icon}</span>
            {t.label}
          </NavLink>
        ))}
      </nav>
    </div>
  )
}
