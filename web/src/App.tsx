import { HashRouter, Navigate, Route, Routes } from 'react-router-dom'
import { useAuth } from './auth'
import Layout from './components/Layout'
import { Spinner } from './components/ui'
import Browse from './pages/Browse'
import ItemDetail from './pages/ItemDetail'
import ItemForm from './pages/ItemForm'
import Login from './pages/Login'
import MatchDetail from './pages/MatchDetail'
import Matches from './pages/Matches'
import MyItems from './pages/MyItems'
import Notifications from './pages/Notifications'
import Profile from './pages/Profile'
import UserProfile from './pages/UserProfile'
import Wants from './pages/Wants'

export default function App() {
  const { user, loading } = useAuth()
  if (loading) return <Spinner />
  if (!user) return <Login />

  // Hash routing: works from any static host without server-side rewrites.
  return (
    <HashRouter>
      <Routes>
        <Route element={<Layout />}>
          <Route index element={<Browse />} />
          <Route path="items/new" element={<ItemForm />} />
          <Route path="items/:id" element={<ItemDetail />} />
          <Route path="items/:id/edit" element={<ItemForm />} />
          <Route path="mine" element={<MyItems />} />
          <Route path="wants" element={<Wants />} />
          <Route path="wants/new" element={<Navigate to="/wants" replace />} />
          <Route path="matches" element={<Matches />} />
          <Route path="matches/:id" element={<MatchDetail />} />
          <Route path="notifications" element={<Notifications />} />
          <Route path="profile" element={<Profile />} />
          <Route path="users/:id" element={<UserProfile />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Route>
      </Routes>
    </HashRouter>
  )
}
