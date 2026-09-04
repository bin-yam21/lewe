import { Ionicons } from '@expo/vector-icons';
import { useRouter } from 'expo-router';
import { useState } from 'react';
import { KeyboardAvoidingView, Platform, Pressable, ScrollView, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { ApiError } from '@/api/client';
import {
  CATEGORIES,
  CONDITIONS,
  EXCHANGE_METHODS,
  labelForCategory,
  type ExchangeMethod,
  type ItemCondition,
} from '@/api/types';
import { Button } from '@/components/Button';
import { Input } from '@/components/Input';
import { Text } from '@/components/Text';
import { Chip } from '@/components/feedback';
import { Divider, Row, Stack } from '@/components/layout';
import { useCreateItem } from '@/hooks/useItems';
import { useTheme } from '@/theme';

type Step = 'details' | 'wants' | 'review';

const STEPS: { key: Step; label: string }[] = [
  { key: 'details', label: 'Item' },
  { key: 'wants', label: 'Wants' },
  { key: 'review', label: 'Review' },
];

export default function CreateListing() {
  const theme = useTheme();
  const router = useRouter();
  const insets = useSafeAreaInsets();
  const createItem = useCreateItem();

  const [step, setStep] = useState<Step>('details');

  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [category, setCategory] = useState<string | undefined>();
  const [condition, setCondition] = useState<ItemCondition>('good');
  const [exchangeMethod, setExchangeMethod] = useState<ExchangeMethod>('either');
  const [location, setLocation] = useState('');
  const [wantCategories, setWantCategories] = useState<string[]>([]);
  const [wantNote, setWantNote] = useState('');

  const [error, setError] = useState<string | null>(null);
  const [fields, setFields] = useState<Record<string, string>>({});

  const detailsValid =
    title.trim().length > 0 && description.trim().length > 0 && !!category;
  const wantsValid = wantCategories.length > 0;

  function reset() {
    setTitle('');
    setDescription('');
    setCategory(undefined);
    setCondition('good');
    setExchangeMethod('either');
    setLocation('');
    setWantCategories([]);
    setWantNote('');
    setStep('details');
    setError(null);
    setFields({});
  }

  function toggleWant(value: string) {
    setWantCategories((current) =>
      current.includes(value) ? current.filter((c) => c !== value) : [...current, value],
    );
  }

  async function onPublish() {
    if (!detailsValid || !wantsValid) return;
    setError(null);
    setFields({});

    try {
      const created = await createItem.mutateAsync({
        title: title.trim(),
        description: description.trim(),
        category: category!,
        condition,
        exchange_method: exchangeMethod,
        location: location.trim() || undefined,
        images: [],
        wants: wantCategories.map((c, i) => ({
          category: c,
          // The note applies to the listing as a whole; attach it to the first
          // want, which is where the API expects free-text detail.
          description: i === 0 && wantNote.trim() ? wantNote.trim() : undefined,
        })),
      });

      reset();
      router.push(`/item/${created.id}`);
    } catch (e) {
      if (e instanceof ApiError) {
        setError(e.message);
        setFields(e.fields ?? {});
      } else {
        setError('Could not publish your listing. Please try again.');
      }
    }
  }

  return (
    <KeyboardAvoidingView
      style={{ flex: 1, backgroundColor: theme.colors.bg }}
      behavior={Platform.OS === 'ios' ? 'padding' : undefined}
    >
      <ScrollView
        contentContainerStyle={{
          padding: theme.space[5],
          paddingTop: insets.top + theme.space[4],
          paddingBottom: theme.space[9],
          gap: theme.space[6],
        }}
        keyboardShouldPersistTaps="handled"
        showsVerticalScrollIndicator={false}
      >
        <Stack gap={4}>
          <Text variant="title">List an item</Text>

          {/* Step indicator — a plain progress rail, no wizard chrome. */}
          <Row gap={2}>
            {STEPS.map((s, i) => {
              const currentIndex = STEPS.findIndex((x) => x.key === step);
              const done = i < currentIndex;
              const active = i === currentIndex;
              return (
                <Stack key={s.key} gap={2} style={{ flex: 1 }}>
                  <View
                    style={{
                      height: 3,
                      borderRadius: 2,
                      backgroundColor:
                        done || active ? theme.colors.accent : theme.colors.border,
                    }}
                  />
                  <Text
                    variant="label"
                    color={active ? 'accent' : done ? 'textMuted' : 'textFaint'}
                    uppercase
                  >
                    {s.label}
                  </Text>
                </Stack>
              );
            })}
          </Row>
        </Stack>

        {error ? (
          <View
            style={{
              backgroundColor: theme.colors.dangerSubtle,
              borderRadius: theme.radius.md,
              padding: theme.space[3],
            }}
          >
            <Row gap={2} align="flex-start">
              <Ionicons name="alert-circle" size={18} color={theme.colors.danger} />
              <Text variant="caption" color="danger" style={{ flex: 1 }}>
                {error}
              </Text>
            </Row>
          </View>
        ) : null}

        {step === 'details' ? (
          <Stack gap={5}>
            <Input
              label="Title"
              value={title}
              onChangeText={setTitle}
              error={fields.title}
              placeholder="Acoustic guitar"
              maxLength={200}
            />

            <Input
              label="Description"
              value={description}
              onChangeText={setDescription}
              error={fields.description}
              placeholder="Condition, age, anything a trader should know…"
              multiline
            />

            <Stack gap={3}>
              <Text variant="caption" color="textMuted" style={{ fontWeight: '600' }}>
                Category
              </Text>
              <Row gap={2} wrap>
                {CATEGORIES.map((c) => (
                  <Chip
                    key={c}
                    label={labelForCategory(c)}
                    selected={category === c}
                    onPress={() => setCategory(c)}
                  />
                ))}
              </Row>
            </Stack>

            <Stack gap={3}>
              <Text variant="caption" color="textMuted" style={{ fontWeight: '600' }}>
                Condition
              </Text>
              <Row gap={2} wrap>
                {CONDITIONS.map((c) => (
                  <Chip
                    key={c.value}
                    label={c.label}
                    selected={condition === c.value}
                    onPress={() => setCondition(c.value)}
                  />
                ))}
              </Row>
            </Stack>

            <Stack gap={3}>
              <Text variant="caption" color="textMuted" style={{ fontWeight: '600' }}>
                How would you exchange it?
              </Text>
              <Row gap={2} wrap>
                {EXCHANGE_METHODS.map((m) => (
                  <Chip
                    key={m.value}
                    label={m.label}
                    selected={exchangeMethod === m.value}
                    onPress={() => setExchangeMethod(m.value)}
                  />
                ))}
              </Row>
            </Stack>

            <Input
              label="Location (optional)"
              value={location}
              onChangeText={setLocation}
              placeholder="Addis Ababa"
            />

            <Button
              title="Continue"
              block
              size="lg"
              disabled={!detailsValid}
              onPress={() => setStep('wants')}
            />
          </Stack>
        ) : null}

        {step === 'wants' ? (
          <Stack gap={5}>
            <Stack gap={2}>
              <Text variant="heading">What would you take for it?</Text>
              <Text variant="body" color="textMuted">
                Lewe only creates a match when the interest runs both ways — so
                pick every category you would genuinely consider.
              </Text>
            </Stack>

            <Row gap={2} wrap>
              {CATEGORIES.map((c) => (
                <Chip
                  key={c}
                  label={labelForCategory(c)}
                  selected={wantCategories.includes(c)}
                  onPress={() => toggleWant(c)}
                />
              ))}
            </Row>

            <Input
              label="Anything specific? (optional)"
              value={wantNote}
              onChangeText={setWantNote}
              placeholder="Ideally a mirrorless camera, but open to offers"
              multiline
            />

            <Row gap={3}>
              <Button title="Back" variant="secondary" size="lg" onPress={() => setStep('details')} />
              <Button
                title="Review"
                size="lg"
                style={{ flex: 1 }}
                disabled={!wantsValid}
                onPress={() => setStep('review')}
              />
            </Row>
          </Stack>
        ) : null}

        {step === 'review' ? (
          <Stack gap={5}>
            <Text variant="heading">Look right?</Text>

            <Stack gap={4}>
              <ReviewRow label="Title" value={title} />
              <Divider />
              <ReviewRow label="Description" value={description} />
              <Divider />
              <ReviewRow label="Category" value={labelForCategory(category ?? '')} />
              <Divider />
              <ReviewRow
                label="Condition"
                value={CONDITIONS.find((c) => c.value === condition)?.label ?? condition}
              />
              <Divider />
              <ReviewRow
                label="Exchange"
                value={EXCHANGE_METHODS.find((m) => m.value === exchangeMethod)?.label ?? exchangeMethod}
              />
              {location.trim() ? (
                <>
                  <Divider />
                  <ReviewRow label="Location" value={location} />
                </>
              ) : null}
              <Divider />
              <ReviewRow
                label="Wants"
                value={wantCategories.map(labelForCategory).join(', ')}
              />
              {wantNote.trim() ? (
                <>
                  <Divider />
                  <ReviewRow label="Note" value={wantNote} />
                </>
              ) : null}
            </Stack>

            <Row gap={3}>
              <Button title="Back" variant="secondary" size="lg" onPress={() => setStep('wants')} />
              <Button
                title="Publish listing"
                size="lg"
                style={{ flex: 1 }}
                loading={createItem.isPending}
                onPress={onPublish}
              />
            </Row>

            <Pressable onPress={reset} hitSlop={8} style={{ alignSelf: 'center' }}>
              <Text variant="caption" color="textFaint">
                Start over
              </Text>
            </Pressable>
          </Stack>
        ) : null}
      </ScrollView>
    </KeyboardAvoidingView>
  );
}

function ReviewRow({ label, value }: { label: string; value: string }) {
  return (
    <Stack gap={1}>
      <Text variant="label" color="textFaint" uppercase>
        {label}
      </Text>
      <Text variant="body">{value}</Text>
    </Stack>
  );
}
