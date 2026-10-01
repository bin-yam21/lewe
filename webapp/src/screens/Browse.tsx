import { useEffect, useState } from 'react';
import { api } from '../api';
import { useAsync } from '../hooks';
import { navigate } from '../router';
import { useSession } from '../session';
import { categoryLabel, formatBirr, type Item } from '../types';
import { Button, Chip, ConditionTag, Empty, ErrorNote, Loading, Page, Photo } from '../ui';

const PAGE = 20;

export function Browse() {
  const { user, catalog } = useSession();
  const [q, setQ] = useState('');
  const [debounced, setDebounced] = useState('');
  const [category, setCategory] = useState('');
  const [items, setItems] = useState<Item[]>([]);
  const [hasMore, setHasMore] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);

  useEffect(() => {
    const t = setTimeout(() => setDebounced(q.trim()), 300);
    return () => clearTimeout(t);
  }, [q]);

  const first = useAsync(() => api.items({ q: debounced, category, limit: PAGE }), [debounced, category]);
  useEffect(() => {
    if (first.data) {
      setItems(first.data.data);
      setHasMore(first.data.data.length === PAGE);
    }
  }, [first.data]);

  async function more() {
    setLoadingMore(true);
    try {
      const next = await api.items({ q: debounced, category, limit: PAGE, offset: items.length });
      setItems((prev) => [...prev, ...next.data]);
      setHasMore(next.data.length === PAGE);
    } finally {
      setLoadingMore(false);
    }
  }

  return (
    <Page title="Browse" subtitle={user.city ? `Swaps around ${user.city}` : 'Things people want to swap'}>
      {!user.city && (
        <button type="button" className="error-note" style={{ color: 'var(--accent)', background: 'var(--card)', border: 0, textAlign: 'left', cursor: 'pointer' }} onClick={() => navigate('/me')}>
          <span>Set your city so we can match you with people nearby.</span>
          <span aria-hidden="true">›</span>
        </button>
      )}
      <label className="search">
        <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true"><path d="m21 21-4.3-4.3M10.5 18a7.5 7.5 0 1 1 0-15 7.5 7.5 0 0 1 0 15Z" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" /></svg>
        <input id="search" type="search" placeholder="Search listings" value={q} onChange={(e) => setQ(e.target.value)} aria-label="Search listings" />
      </label>
      <div className="chips" role="toolbar" aria-label="Categories">
        <Chip active={category === ''} onClick={() => setCategory('')}>All</Chip>
        {catalog.categories.map((c) => (
          <Chip key={c} active={category === c} onClick={() => setCategory(category === c ? '' : c)}>{categoryLabel(c)}</Chip>
        ))}
      </div>

      {first.error && <ErrorNote error={first.error} onRetry={first.reload} />}
      {first.loading && items.length === 0 ? (
        <Loading />
      ) : items.length === 0 ? (
        <Empty icon="🔍" title={debounced || category ? 'Nothing matches that' : 'No listings yet'}>
          <span>{debounced || category ? 'Try another search or category.' : 'Be the first: list something you no longer need.'}</span>
          <div style={{ width: 200 }}><Button onClick={() => navigate('/new')}>List an item</Button></div>
        </Empty>
      ) : (
        <>
          <div className="grid">
            {items.map((it) => <ItemCard key={it.id} item={it} mine={it.owner_id === user.id} />)}
          </div>
          {hasMore && <Button kind="secondary" busy={loadingMore} onClick={more}>Show more</Button>}
        </>
      )}
    </Page>
  );
}

export function ItemCard({ item, mine, status }: { item: Item; mine?: boolean; status?: boolean }) {
  return (
    <button type="button" className="card" onClick={() => navigate(`/items/${item.id}`)}>
      <div className="card-photo">
        <Photo src={item.image_urls[0]} alt={item.title} />
        {status && item.status !== 'available' && <span className="card-status pill tone-muted" style={{ background: 'var(--card)' }}>{item.status === 'reserved' ? 'Reserved' : 'Swapped'}</span>}
        {mine && !status && <span className="card-status pill tone-accent" style={{ background: 'var(--card)' }}>Yours</span>}
      </div>
      <div className="card-body">
        <span className="card-title">{item.title}</span>
        <span className="card-meta">
          <ConditionTag c={item.condition} />
          <span>{formatBirr(item.estimated_value)}</span>
        </span>
      </div>
    </button>
  );
}
