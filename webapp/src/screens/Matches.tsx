import { useState } from 'react';
import { api } from '../api';
import { useAsync, useInterval } from '../hooks';
import { navigate } from '../router';
import type { Match } from '../types';
import { Chip, Empty, ErrorNote, Loading, Page, Photo, StatusPill } from '../ui';

type Filter = 'active' | 'done' | 'closed';
const GROUPS: Record<Filter, Match['status'][]> = {
  active: ['pending', 'accepted'],
  done: ['completed'],
  closed: ['declined', 'cancelled'],
};

/** What the viewer should do next, or who they are waiting on. */
export function nextStep(m: Match): { text: string; yours: boolean } {
  const them = m.other_user.full_name.split(' ')[0];
  switch (m.status) {
    case 'pending':
      return m.you_accepted ? { text: `Waiting for ${them} to accept`, yours: false } : { text: 'Your turn: accept or decline', yours: true };
    case 'accepted':
      if (!m.exchange) return { text: 'Agree how to swap', yours: true };
      return m.you_completed ? { text: `Waiting for ${them} to confirm`, yours: false } : { text: "Confirm once you've swapped", yours: true };
    case 'completed':
      return m.you_rated ? { text: 'Swapped', yours: false } : { text: `Rate ${them}`, yours: true };
    default:
      return { text: m.status === 'declined' ? 'Declined' : 'Cancelled', yours: false };
  }
}

export function Matches() {
  const [filter, setFilter] = useState<Filter>('active');
  const list = useAsync(() => api.matches(), []);
  useInterval(list.reload, 15000);

  const all = list.data?.data ?? [];
  const shown = all.filter((m) => GROUPS[filter].includes(m.status));
  const waiting = all.filter((m) => GROUPS.active.includes(m.status) && nextStep(m).yours).length;

  return (
    <Page title="Swaps" subtitle={waiting ? `${waiting} need${waiting === 1 ? 's' : ''} your answer` : 'Two-way matches found for you'}>
      <div className="chips" role="toolbar" aria-label="Filter swaps">
        <Chip active={filter === 'active'} onClick={() => setFilter('active')}>In progress</Chip>
        <Chip active={filter === 'done'} onClick={() => setFilter('done')}>Swapped</Chip>
        <Chip active={filter === 'closed'} onClick={() => setFilter('closed')}>Closed</Chip>
      </div>
      {list.error && <ErrorNote error={list.error} onRetry={list.reload} />}
      {list.loading && !list.data ? (
        <Loading />
      ) : shown.length === 0 ? (
        <Empty icon="⇄" title={filter === 'active' ? 'No swaps in progress' : filter === 'done' ? 'No finished swaps yet' : 'Nothing closed'}>
          {filter === 'active' && <span>List items and add wants under Me. When someone has what you want and wants what you have, the swap shows up here.</span>}
        </Empty>
      ) : (
        shown.map((m) => <MatchCard key={m.id} m={m} />)
      )}
    </Page>
  );
}

function MatchCard({ m }: { m: Match }) {
  const step = nextStep(m);
  return (
    <button type="button" className="match-card" onClick={() => navigate(`/matches/${m.id}`)}>
      <div className="top">
        <span>with <b>{m.other_user.full_name}</b></span>
        <StatusPill status={m.status} />
      </div>
      <div className="swap">
        <div className="swap-side">
          <Photo src={m.your_item.image_urls[0]} alt={m.your_item.title} />
          <span className="label">You give</span>
          <span className="name">{m.your_item.title}</span>
        </div>
        <span className="swap-arrow" aria-hidden="true">⇄</span>
        <div className="swap-side">
          <Photo src={m.their_item.image_urls[0]} alt={m.their_item.title} />
          <span className="label">You get</span>
          <span className="name">{m.their_item.title}</span>
        </div>
      </div>
      <span className="turn" style={{ color: step.yours ? 'var(--accent)' : 'var(--hint)' }}>{step.text}</span>
    </button>
  );
}
