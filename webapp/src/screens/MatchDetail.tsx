import { useEffect, useRef, useState } from 'react';
import { api } from '../api';
import { useAsync, useBackButton, useInterval, useMainButton } from '../hooks';
import { goBack, navigate } from '../router';
import { useSession } from '../session';
import { confirmAction, haptic } from '../telegram';
import { METHOD_LABELS, formatBirr, type ExchangeMethod, type Match } from '../types';
import { Button, Chip, ConditionTag, ErrorNote, Loading, Page, Photo, Section, Stars, StatusPill } from '../ui';
import { nextStep } from './Matches';

export function MatchDetail({ id }: { id: string }) {
  useBackButton(goBack);
  const { refreshUnread } = useSession();
  const m = useAsync(() => api.match(id), [id]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  useInterval(m.reload, 10000);
  useEffect(refreshUnread, [refreshUnread]);

  async function act(fn: () => Promise<Match | unknown>, opts: { confirm?: string } = {}) {
    if (opts.confirm && !(await confirmAction(opts.confirm))) return;
    setBusy(true);
    setError(null);
    try {
      const result = await fn();
      haptic.success();
      if (result && typeof result === 'object' && 'status' in result) m.setData(result as Match);
      else await m.reload();
    } catch (e) {
      haptic.error();
      setError(e as Error);
      await m.reload();
    } finally {
      setBusy(false);
    }
  }

  const match = m.data;
  const canAccept = match?.status === 'pending' && !match.you_accepted;
  const native = useMainButton('Accept swap', () => act(() => api.accept(id)), { visible: Boolean(canAccept), busy });

  if (m.error && !match) return <Page><ErrorNote error={m.error} onRetry={m.reload} /></Page>;
  if (!match) return <Loading />;
  const them = match.other_user.full_name.split(' ')[0];

  return (
    <Page>
      <div className="match-card" style={{ cursor: 'default' }}>
        <div className="top">
          <span>Swap with <b>{match.other_user.full_name}</b></span>
          <StatusPill status={match.status} />
        </div>
        <div className="swap">
          <Side label="You give" item={match.your_item} />
          <span className="swap-arrow" aria-hidden="true">⇄</span>
          <Side label="You get" item={match.their_item} />
        </div>
      </div>

      {['pending', 'accepted', 'completed'].includes(match.status) && <Progress m={match} />}

      {error && <ErrorNote error={error} />}

      {match.status === 'pending' && (
        <Section>
          <div className="panel">
            {canAccept ? (
              <>
                <p>{them} has something you want, and wants your {match.your_item.title}. {match.they_accepted ? `${them} has already accepted. ` : ''}Accepting reserves both items until the swap is done or cancelled.</p>
                {!native && <Button busy={busy} onClick={() => act(() => api.accept(id))}>Accept swap</Button>}
                <Button kind="danger" disabled={busy} onClick={() => act(() => api.decline(id), { confirm: 'Decline this swap? It won’t be suggested again.' })}>Decline</Button>
              </>
            ) : (
              <p>You accepted. Waiting for {them} to accept too. Use the chat below to ask questions.</p>
            )}
          </div>
        </Section>
      )}

      {match.status === 'accepted' && <Arrange m={match} busy={busy} act={act} them={them} />}

      {match.status === 'completed' && (
        match.you_rated ? (
          <Section><div className="panel"><p>Swap complete. Thanks for rating {them}.</p></div></Section>
        ) : (
          <RateForm them={them} onRate={(score, comment) => act(() => api.rate(id, score, comment))} busy={busy} />
        )
      )}

      {(match.status === 'declined' || match.status === 'cancelled') && (
        <Section><div className="panel"><p className="hint">This swap was {match.status}. The items are back on the market.</p></div></Section>
      )}

      <Chat matchId={id} open={match.status !== 'declined' && match.status !== 'cancelled'} them={them} />
    </Page>
  );
}

function Side({ label, item }: { label: string; item: Match['your_item'] }) {
  return (
    <button type="button" className="swap-side" style={{ background: 'none', border: 0, padding: 0, textAlign: 'left', cursor: 'pointer' }} onClick={() => navigate(`/items/${item.id}`)}>
      <Photo src={item.image_urls[0]} alt={item.title} />
      <span className="label">{label}</span>
      <span className="name">{item.title}</span>
      <span className="hint" style={{ display: 'flex', gap: 6, alignItems: 'center' }}><ConditionTag c={item.condition} />{formatBirr(item.estimated_value)}</span>
    </button>
  );
}

function Progress({ m }: { m: Match }) {
  const done = [
    true,
    m.status !== 'pending',
    m.status === 'completed' || Boolean(m.exchange),
    m.status === 'completed',
  ];
  const labels = ['Matched', 'Both accept', 'Plan swap', 'Swapped'];
  const now = done.indexOf(false);
  return (
    <Section title={nextStep(m).text}>
      <div className="steps">
        {labels.map((l, i) => (
          <div key={l} className={'step' + (done[i] ? ' done' : i === now ? ' now' : '')}>
            <i />
            {l}
          </div>
        ))}
      </div>
    </Section>
  );
}

function Arrange({ m, busy, act, them }: { m: Match; busy: boolean; act: (fn: () => Promise<unknown>, o?: { confirm?: string }) => void; them: string }) {
  const [method, setMethod] = useState<ExchangeMethod | null>(m.exchange?.method ?? null);
  const [details, setDetails] = useState(m.exchange?.details ?? '');
  const changed = method !== (m.exchange?.method ?? null) || details.trim() !== (m.exchange?.details ?? '');

  return (
    <>
      <Section title="How will you swap?" footer={m.exchange ? `${m.exchange.proposed_by_you ? 'You' : them} set this plan. Changing it asks both of you to confirm again.` : 'Agree on a plan in the chat, then save it here.'}>
        <div className="panel">
          <div className="segmented">
            {(Object.keys(METHOD_LABELS) as ExchangeMethod[]).map((k) => (
              <Chip key={k} active={method === k} onClick={() => setMethod(k)}>{METHOD_LABELS[k]}</Chip>
            ))}
          </div>
          <div className="field" style={{ padding: 0 }}>
            <label htmlFor="details">Where and when</label>
            <input id="details" value={details} maxLength={1000} placeholder="e.g. Tomoca, Piassa, Saturday 10:00" onChange={(e) => setDetails(e.target.value)} />
          </div>
          {changed && method && (
            <Button kind="secondary" busy={busy} onClick={() => act(() => api.setExchange(m.id, method, details.trim() || undefined))}>Save plan</Button>
          )}
        </div>
      </Section>

      {m.exchange && !changed && (
        <Section>
          <div className="panel">
            {m.you_completed ? (
              <p>You confirmed the hand-over. Waiting for {them} to confirm.</p>
            ) : (
              <>
                <p>{m.they_completed ? `${them} confirmed the hand-over. ` : ''}Confirm once you’ve handed over your {m.your_item.title}.</p>
                <Button busy={busy} onClick={() => act(() => api.complete(m.id))}>I’ve handed it over</Button>
              </>
            )}
          </div>
        </Section>
      )}

      <Button kind="plain" disabled={busy} onClick={() => act(() => api.cancel(m.id), { confirm: 'Cancel this swap? Both items go back on the market.' })}>
        <span style={{ color: 'var(--danger)' }}>Cancel swap</span>
      </Button>
    </>
  );
}

function RateForm({ them, onRate, busy }: { them: string; onRate: (score: number, comment?: string) => void; busy: boolean }) {
  const [score, setScore] = useState(0);
  const [comment, setComment] = useState('');
  return (
    <Section title={`How was swapping with ${them}?`}>
      <div className="panel" style={{ justifyItems: 'center' }}>
        <Stars value={score} onChange={setScore} size={34} />
        <div className="field" style={{ padding: 0, width: '100%' }}>
          <label htmlFor="comment">Comment (optional)</label>
          <input id="comment" value={comment} maxLength={1000} placeholder="Item as described? Easy to meet?" onChange={(e) => setComment(e.target.value)} />
        </div>
        <Button disabled={score === 0} busy={busy} onClick={() => onRate(score, comment.trim() || undefined)}>Submit rating</Button>
      </div>
    </Section>
  );
}

function Chat({ matchId, open, them }: { matchId: string; open: boolean; them: string }) {
  const msgs = useAsync(() => api.messages(matchId), [matchId]);
  const [text, setText] = useState('');
  const [sending, setSending] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const end = useRef<HTMLDivElement>(null);
  useInterval(msgs.reload, 5000);

  const list = msgs.data?.data ?? [];
  useEffect(() => end.current?.scrollIntoView({ block: 'nearest' }), [list.length]);

  async function send() {
    const body = text.trim();
    if (!body) return;
    setSending(true);
    setError(null);
    try {
      const sent = await api.sendMessage(matchId, body);
      msgs.setData({ data: [...list, sent], limit: 100, offset: 0 });
      setText('');
      haptic.tap();
    } catch (e) {
      setError(e as Error);
    } finally {
      setSending(false);
    }
  }

  return (
    <Section title={`Chat with ${them}`}>
      <div className="chat" aria-live="polite">
        {list.length === 0 && <p className="hint" style={{ textAlign: 'center', padding: 8 }}>No messages yet. Say hello and agree where to meet.</p>}
        {list.map((msg) => (
          <div key={msg.id} className={'bubble ' + (msg.from_you ? 'you' : 'them')}>
            {msg.body}
            <time dateTime={msg.created_at}>{new Date(msg.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</time>
          </div>
        ))}
        <div ref={end} />
      </div>
      {error && <div style={{ padding: '0 12px' }}><ErrorNote error={error} /></div>}
      {open && (
        <form className="composer" onSubmit={(e) => { e.preventDefault(); send(); }}>
          <textarea
            id="message"
            rows={1}
            value={text}
            maxLength={2000}
            placeholder="Message"
            aria-label="Message"
            onChange={(e) => setText(e.target.value)}
            onKeyDown={(e) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); send(); } }}
          />
          <button type="submit" disabled={sending || !text.trim()} aria-label="Send">
            <svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true"><path d="M4 12 20 4l-6 16-2.5-6.5L4 12Z" fill="currentColor" /></svg>
          </button>
        </form>
      )}
    </Section>
  );
}
