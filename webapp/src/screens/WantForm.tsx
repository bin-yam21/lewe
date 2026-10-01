import { useState } from 'react';
import { ApiError, api } from '../api';
import { useBackButton, useMainButton } from '../hooks';
import { goBack, navigate, query, usePath } from '../router';
import { useSession } from '../session';
import { haptic } from '../telegram';
import { CONDITION_LABELS, categoryLabel, type Condition } from '../types';
import { Button, Chip, ErrorNote, Page, Section } from '../ui';

export function WantForm() {
  const { catalog, user } = useSession();
  useBackButton(goBack);
  const q = query(usePath());
  const [category, setCategory] = useState(q.get('category') ?? '');
  const [keywords, setKeywords] = useState(q.get('keyword') ?? '');
  const [minCondition, setMinCondition] = useState<Condition | ''>('');
  const [anyCity, setAnyCity] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  async function save() {
    if (!category || busy) return;
    setBusy(true);
    setError(null);
    try {
      await api.createWant({
        category,
        keywords: keywords.split(',').map((k) => k.trim()).filter(Boolean),
        min_condition: minCondition || null,
        any_city: anyCity,
      });
      haptic.success();
      navigate('/me', { replace: true });
    } catch (e) {
      haptic.error();
      setError(e instanceof ApiError && e.fields ? new Error(Object.entries(e.fields).map(([k, v]) => `${k} ${v}`).join('; ')) : (e as Error));
      setBusy(false);
    }
  }

  const native = useMainButton('Save want', save, { enabled: category !== '', busy });

  return (
    <Page title="What do you want?" subtitle="We’ll match you with people who have it and want something you listed.">
      <Section title="Category">
        <div className="field"><div className="segmented">
          {catalog.categories.map((c) => <Chip key={c} active={category === c} onClick={() => setCategory(c)}>{categoryLabel(c)}</Chip>)}
        </div></div>
      </Section>

      <Section footer="Separate words with commas. Leave empty to match anything in the category.">
        <div className="field">
          <label htmlFor="keywords">Keywords</label>
          <input id="keywords" value={keywords} placeholder="e.g. guitar, krar" onChange={(e) => setKeywords(e.target.value)} />
        </div>
      </Section>

      <Section title="At least in this condition">
        <div className="field"><div className="segmented">
          <Chip active={minCondition === ''} onClick={() => setMinCondition('')}>Any</Chip>
          {catalog.conditions.filter((c) => c !== 'poor').map((c) => (
            <Chip key={c} active={minCondition === c} onClick={() => setMinCondition(c)}>{CONDITION_LABELS[c]}</Chip>
          ))}
        </div></div>
      </Section>

      <Section footer={user.city ? `Off: only people in ${user.city}.` : 'Set your city under Me to keep matches local.'}>
        <label className="toggle" htmlFor="anycity">
          <span>Match people in other cities<br /><span className="hint">For swaps you’re happy to ship</span></span>
          <input id="anycity" type="checkbox" role="switch" checked={anyCity} onChange={(e) => { haptic.select(); setAnyCity(e.target.checked); }} />
        </label>
      </Section>

      {error && <ErrorNote error={error} />}
      {!native && <Button disabled={!category} busy={busy} onClick={save}>Save want</Button>}
    </Page>
  );
}
