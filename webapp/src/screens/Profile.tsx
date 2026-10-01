import { useState } from 'react';
import { api } from '../api';
import { useAsync } from '../hooks';
import { navigate } from '../router';
import { useSession } from '../session';
import { confirmAction, haptic } from '../telegram';
import { CONDITION_LABELS, categoryLabel } from '../types';
import { Avatar, Cell, Empty, ErrorNote, Page, Section, Spinner } from '../ui';
import { ItemCard } from './Browse';

export function Profile() {
  const { user, setUser, catalog, unread } = useSession();
  const profile = useAsync(() => api.profile(user.id), [user.id]);
  const wants = useAsync(() => api.wants(), []);
  const items = useAsync(() => api.myItems(), []);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  async function setCity(city: string) {
    setSaving(true);
    setError(null);
    try {
      const updated = await api.updateMe({
        full_name: user.full_name,
        phone: user.phone,
        location: user.location,
        bio: user.bio,
        avatar_url: user.avatar_url,
        city: city || undefined,
      });
      setUser(updated);
      haptic.success();
    } catch (e) {
      setError(e as Error);
    } finally {
      setSaving(false);
    }
  }

  async function cancelWant(id: string) {
    if (!(await confirmAction('Remove this want? Pending matches from it will be cancelled.'))) return;
    try {
      await api.cancelWant(id);
      wants.reload();
    } catch (e) {
      setError(e as Error);
    }
  }

  const rating = profile.data?.rating;
  const visibleItems = (items.data?.data ?? []).filter((i) => i.status !== 'withdrawn');

  return (
    <Page>
      <div className="profile">
        <Avatar name={user.full_name} src={user.avatar_url} size={84} />
        <h1>{user.full_name}</h1>
        {user.telegram_username && <span className="hint">@{user.telegram_username}</span>}
        <span className="rating">
          {rating && rating.count > 0 ? <><span style={{ color: '#f5b301' }}>★</span><b>{rating.average.toFixed(1)}</b> from {rating.count} swap{rating.count > 1 ? 's' : ''}</> : 'No ratings yet'}
        </span>
      </div>

      {error && <ErrorNote error={error} />}

      <Section footer="Matches stay in your city unless a want allows any city.">
        <div className="cell">
          <span className="cell-main"><label htmlFor="city" className="cell-title">City</label></span>
          <span className="cell-after">
            {saving && <Spinner small />}
            <select id="city" value={user.city ?? ''} onChange={(e) => setCity(e.target.value)} style={{ border: 0, background: 'transparent', color: 'var(--link)', textAlign: 'right' }}>
              <option value="">Not set</option>
              {catalog.cities.map((c) => <option key={c} value={c}>{c}</option>)}
            </select>
          </span>
        </div>
        <Cell title="Notifications" onClick={() => navigate('/notifications')} after={<>{unread > 0 && <span className="badge">{unread}</span>}›</>} />
      </Section>

      <Section title="What I’m looking for">
        {(wants.data?.data ?? []).map((w) => (
          <Cell
            key={w.id}
            title={categoryLabel(w.category) + (w.keywords.length ? `: ${w.keywords.join(', ')}` : '')}
            subtitle={[w.min_condition && `${CONDITION_LABELS[w.min_condition]} or better`, w.any_city ? 'Any city' : 'My city'].filter(Boolean).join(' · ')}
            after={<button type="button" className="link" style={{ color: 'var(--danger)' }} onClick={() => cancelWant(w.id)}>Remove</button>}
          />
        ))}
        <Cell onClick={() => navigate('/wants/new')} title={<span style={{ color: 'var(--link)' }}>+ Add a want</span>} />
      </Section>

      <Section title={`My listings · ${visibleItems.length}`}>
        {visibleItems.length === 0 ? (
          <Empty icon="📦" title="Nothing listed yet">
            <button type="button" className="link" onClick={() => navigate('/new')}>List your first item</button>
          </Empty>
        ) : (
          <div className="grid" style={{ padding: 12, background: 'var(--bg)' }}>
            {visibleItems.map((it) => <ItemCard key={it.id} item={it} status />)}
          </div>
        )}
      </Section>
    </Page>
  );
}
