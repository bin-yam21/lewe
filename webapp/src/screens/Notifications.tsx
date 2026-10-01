import { useEffect } from 'react';
import { api } from '../api';
import { useAsync, useBackButton } from '../hooks';
import { goBack, navigate } from '../router';
import { useSession } from '../session';
import { Cell, Empty, ErrorNote, Loading, Page, Section, timeAgo } from '../ui';

const ICON: Record<string, string> = {
  match_found: '✨',
  match_accepted: '👍',
  match_confirmed: '🤝',
  match_declined: '✋',
  match_cancelled: '↩️',
  exchange_updated: '📍',
  match_completed: '✅',
  message_received: '💬',
  rating_received: '⭐',
};

export function Notifications() {
  useBackButton(goBack);
  const { refreshUnread } = useSession();
  const list = useAsync(() => api.notifications(), []);

  // Opening the list counts as reading it.
  useEffect(() => {
    if (list.data?.data.some((n) => !n.read)) api.readAll().then(refreshUnread).catch(() => {});
  }, [list.data, refreshUnread]);

  return (
    <Page title="Notifications">
      {list.error && <ErrorNote error={list.error} onRetry={list.reload} />}
      {!list.data ? (
        <Loading />
      ) : list.data.data.length === 0 ? (
        <Empty icon="🔔" title="All quiet">New matches, messages and ratings show up here and in your Telegram chat with the bot.</Empty>
      ) : (
        <Section>
          {list.data.data.map((n) => (
            <Cell
              key={n.id}
              before={<span style={{ fontSize: 22, width: 28, textAlign: 'center' }} aria-hidden="true">{ICON[n.type] ?? '•'}</span>}
              title={<span style={{ fontWeight: n.read ? 400 : 600 }}>{n.message}</span>}
              subtitle={timeAgo(n.created_at)}
              after={n.match_id ? '›' : undefined}
              onClick={n.match_id ? () => navigate(`/matches/${n.match_id}`) : undefined}
            />
          ))}
        </Section>
      )}
    </Page>
  );
}
