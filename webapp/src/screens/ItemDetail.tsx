import { useRef, useState } from 'react';
import { api } from '../api';
import { useAsync, useBackButton } from '../hooks';
import { goBack, navigate } from '../router';
import { useSession } from '../session';
import { confirmAction, haptic } from '../telegram';
import { categoryLabel, formatBirr } from '../types';
import { Avatar, Button, Cell, ConditionTag, ErrorNote, Loading, Page, Photo, Section, timeAgo } from '../ui';

export function ItemDetail({ id }: { id: string }) {
  const { user } = useSession();
  useBackButton(goBack);
  const item = useAsync(() => api.item(id), [id]);
  const owner = useAsync(() => (item.data ? api.profile(item.data.owner_id) : Promise.resolve(undefined)), [item.data?.owner_id]);
  const [slide, setSlide] = useState(0);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const gallery = useRef<HTMLDivElement>(null);

  if (item.error) return <Page><ErrorNote error={item.error} onRetry={item.reload} /></Page>;
  if (!item.data) return <Loading />;
  const it = item.data;
  const mine = it.owner_id === user.id;
  const photos = it.image_urls.length ? it.image_urls : [undefined];

  async function withdraw() {
    if (!(await confirmAction('Withdraw this listing? Pending matches for it will be cancelled.'))) return;
    setBusy(true);
    try {
      await api.withdrawItem(id);
      haptic.success();
      navigate('/me', { replace: true });
    } catch (e) {
      setError(e as Error);
      setBusy(false);
    }
  }

  return (
    <Page>
      <div
        className="gallery"
        ref={gallery}
        onScroll={(e) => setSlide(Math.round(e.currentTarget.scrollLeft / e.currentTarget.clientWidth))}
      >
        {photos.map((src, i) => <Photo key={i} src={src} alt={`${it.title}, photo ${i + 1}`} />)}
      </div>
      {photos.length > 1 && (
        <div className="dots" aria-hidden="true">{photos.map((_, i) => <span key={i} className={i === slide ? 'on' : ''} />)}</div>
      )}

      <div className="item-head">
        <h1>{it.title}</h1>
        <div className="item-meta">
          <ConditionTag c={it.condition} />
          <span>{categoryLabel(it.category)}</span>
          {it.location && <span>· {it.location}</span>}
          <span>· {timeAgo(it.created_at)}</span>
        </div>
        {it.estimated_value != null && <div className="value">≈ {formatBirr(it.estimated_value)}</div>}
        {it.status !== 'available' && <div><span className="pill tone-warn">{it.status === 'reserved' ? 'Reserved for a swap' : it.status}</span></div>}
      </div>

      {it.description && (
        <Section title="Description"><p className="prose">{it.description}</p></Section>
      )}

      {owner.data && (
        <Section title={mine ? 'You listed this' : 'Listed by'}>
          <Cell
            before={<Avatar name={owner.data.full_name} src={owner.data.avatar_url} />}
            title={owner.data.full_name}
            subtitle={[owner.data.city, owner.data.rating.count ? `★ ${owner.data.rating.average.toFixed(1)} · ${owner.data.rating.count} swap${owner.data.rating.count > 1 ? 's' : ''}` : 'No ratings yet'].filter(Boolean).join(' · ')}
          />
        </Section>
      )}

      {error && <ErrorNote error={error} />}
      {mine ? (
        it.status === 'available' && (
          <div className="btn-row">
            <Button kind="secondary" onClick={() => navigate(`/items/${id}/edit`)}>Edit</Button>
            <Button kind="danger" busy={busy} onClick={withdraw}>Withdraw</Button>
          </div>
        )
      ) : (
        <Section footer="Lewe matches you automatically when you have something they want too.">
          <Cell
            onClick={() => navigate(`/wants/new?category=${encodeURIComponent(it.category)}&keyword=${encodeURIComponent(it.title.split(/\s+/).slice(0, 2).join(' '))}`)}
            title={<span style={{ color: 'var(--link)' }}>I want something like this</span>}
            after="›"
          />
        </Section>
      )}
    </Page>
  );
}
