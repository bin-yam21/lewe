import { useEffect, useState, type ChangeEvent } from 'react';
import { ApiError, api, shrinkImage } from '../api';
import { useBackButton, useMainButton } from '../hooks';
import { goBack, navigate } from '../router';
import { useSession } from '../session';
import { haptic } from '../telegram';
import { CONDITION_LABELS, categoryLabel, type Condition } from '../types';
import { Button, Chip, ErrorNote, Loading, Page, Photo, Section, Spinner } from '../ui';

const MAX_PHOTOS = 6;

export function ItemForm({ id }: { id?: string }) {
  const { catalog, user } = useSession();
  useBackButton(id ? goBack : null);
  const [loaded, setLoaded] = useState(!id);
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [category, setCategory] = useState('');
  const [condition, setCondition] = useState<Condition>('good');
  const [value, setValue] = useState('');
  const [location, setLocation] = useState(user.city ?? '');
  const [photos, setPhotos] = useState<string[]>([]);
  const [uploading, setUploading] = useState(0);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [fields, setFields] = useState<Record<string, string>>({});

  useEffect(() => {
    if (!id) return;
    api.item(id).then((it) => {
      setTitle(it.title);
      setDescription(it.description);
      setCategory(it.category);
      setCondition(it.condition);
      setValue(it.estimated_value != null ? String(it.estimated_value) : '');
      setLocation(it.location ?? '');
      setPhotos(it.image_urls);
      setLoaded(true);
    }).catch(setError);
  }, [id]);

  async function addPhotos(e: ChangeEvent<HTMLInputElement>) {
    const files = Array.from(e.target.files ?? []).slice(0, MAX_PHOTOS - photos.length);
    e.target.value = '';
    setUploading((n) => n + files.length);
    for (const f of files) {
      try {
        const url = await api.upload(await shrinkImage(f));
        setPhotos((p) => [...p, url]);
      } catch (err) {
        setError(err as Error);
      } finally {
        setUploading((n) => n - 1);
      }
    }
  }

  const ready = title.trim() !== '' && category !== '' && uploading === 0;

  async function save() {
    if (!ready || busy) return;
    setBusy(true);
    setError(null);
    setFields({});
    const input = {
      title: title.trim(),
      description: description.trim(),
      category,
      condition,
      estimated_value: value.trim() === '' ? null : Number(value),
      location: location.trim() || null,
      image_urls: photos,
    };
    try {
      const it = id ? await api.updateItem(id, input) : await api.createItem(input);
      haptic.success();
      navigate(`/items/${it.id}`, { replace: true });
    } catch (e) {
      haptic.error();
      setError(e as Error);
      if (e instanceof ApiError && e.fields) setFields(e.fields);
      setBusy(false);
    }
  }

  const native = useMainButton(id ? 'Save changes' : 'Publish listing', save, { enabled: ready, busy, visible: loaded });

  if (!loaded) return error ? <Page><ErrorNote error={error} /></Page> : <Loading />;

  return (
    <Page title={id ? 'Edit listing' : 'List an item'} subtitle={id ? undefined : 'Photos first: listings with photos get matched faster.'}>
      <Section title={`Photos · ${photos.length}/${MAX_PHOTOS}`}>
        <div className="photos">
          {photos.map((src, i) => (
            <div className="slot" key={src}>
              <Photo src={src} alt={`Photo ${i + 1}`} />
              <button type="button" className="remove" aria-label={`Remove photo ${i + 1}`} onClick={() => setPhotos((p) => p.filter((x) => x !== src))}>×</button>
            </div>
          ))}
          {Array.from({ length: uploading }).map((_, i) => (
            <div className="slot" key={'u' + i}><div className="photo placeholder"><Spinner small /></div></div>
          ))}
          {photos.length + uploading < MAX_PHOTOS && (
            <label className="add" aria-label="Add photos">
              +
              <input id="photos" type="file" accept="image/jpeg,image/png,image/webp" multiple onChange={addPhotos} />
            </label>
          )}
        </div>
      </Section>

      <Section>
        <div className="field">
          <label htmlFor="title">Title</label>
          <input id="title" value={title} maxLength={120} placeholder="e.g. Canon EOS 250D camera" onChange={(e) => setTitle(e.target.value)} />
          {fields.title && <span className="field-error">Title {fields.title}</span>}
        </div>
        <div className="field">
          <label htmlFor="description">Description</label>
          <textarea id="description" value={description} maxLength={5000} placeholder="What's included, its history, any flaws" onChange={(e) => setDescription(e.target.value)} />
        </div>
      </Section>

      <Section title="Category">
        <div className="field"><div className="segmented">
          {catalog.categories.map((c) => <Chip key={c} active={category === c} onClick={() => setCategory(c)}>{categoryLabel(c)}</Chip>)}
        </div></div>
      </Section>

      <Section title="Condition">
        <div className="field"><div className="segmented">
          {catalog.conditions.map((c) => <Chip key={c} active={condition === c} onClick={() => setCondition(c)}>{CONDITION_LABELS[c]}</Chip>)}
        </div></div>
      </Section>

      <Section footer="The value helps us suggest fair swaps. It's never a price.">
        <div className="field">
          <label htmlFor="value">Estimated value (Birr)</label>
          <input id="value" inputMode="numeric" pattern="[0-9]*" value={value} placeholder="Optional" onChange={(e) => setValue(e.target.value.replace(/\D/g, ''))} />
        </div>
        <div className="field">
          <label htmlFor="location">Area</label>
          <input id="location" value={location} placeholder="e.g. Bole, Addis Ababa" onChange={(e) => setLocation(e.target.value)} />
        </div>
      </Section>

      {error && <ErrorNote error={error} />}
      {!native && <Button onClick={save} disabled={!ready} busy={busy}>{id ? 'Save changes' : 'Publish listing'}</Button>}
    </Page>
  );
}
