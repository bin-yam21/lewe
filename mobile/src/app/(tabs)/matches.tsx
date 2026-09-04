import { Ionicons } from '@expo/vector-icons';
import { useRouter } from 'expo-router';
import { useMemo } from 'react';
import { Alert, ScrollView, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { ApiError } from '@/api/client';
import type { Match } from '@/api/types';
import { Button } from '@/components/Button';
import { Text } from '@/components/Text';
import { Badge, EmptyState, ErrorState, Skeleton } from '@/components/feedback';
import { Divider, Row, Stack } from '@/components/layout';
import { AppearFromBottom, PressableScale } from '@/components/motion';
import { useAuth } from '@/hooks/useAuth';
import {
  describeMatch,
  useCancelMatch,
  useCompleteMatch,
  useMyMatches,
  useRespondToMatch,
} from '@/hooks/useMatches';
import { useTheme } from '@/theme';

export default function Matches() {
  const theme = useTheme();
  const router = useRouter();
  const insets = useSafeAreaInsets();
  const { user } = useAuth();

  const { data, isLoading, isError, error, refetch, isRefetching } = useMyMatches();
  const respond = useRespondToMatch();
  const complete = useCompleteMatch();
  const cancel = useCancelMatch();

  const { open, done } = useMemo(() => {
    const all = data?.matches ?? [];
    return {
      open: all.filter((m) => m.status !== 'completed' && m.status !== 'cancelled'),
      done: all.filter((m) => m.status === 'completed' || m.status === 'cancelled'),
    };
  }, [data]);

  function onError(e: unknown) {
    Alert.alert('Something went wrong', e instanceof ApiError ? e.message : 'Please try again.');
  }

  return (
    <ScrollView
      style={{ flex: 1, backgroundColor: theme.colors.bgSubtle }}
      contentContainerStyle={{
        padding: theme.space[5],
        paddingTop: insets.top + theme.space[4],
        paddingBottom: theme.space[9],
        gap: theme.space[5],
      }}
      refreshControl={undefined}
      showsVerticalScrollIndicator={false}
    >
      <Stack gap={1}>
        <Text variant="title">Trades</Text>
        <Text variant="caption" color="textMuted">
          Offers you have sent and received.
        </Text>
      </Stack>

      {isLoading ? (
        <Stack gap={3}>
          <Skeleton height={150} radius={theme.radius.lg} />
          <Skeleton height={150} radius={theme.radius.lg} />
        </Stack>
      ) : isError ? (
        <ErrorState
          message={error instanceof ApiError ? error.message : 'Could not load your trades.'}
          onRetry={() => refetch()}
        />
      ) : open.length === 0 && done.length === 0 ? (
        <EmptyState
          icon="swap-horizontal-outline"
          title="No trades yet"
          message="Find something you like and offer one of your items for it."
          action={<Button title="Browse items" onPress={() => router.push('/(tabs)')} />}
        />
      ) : (
        <Stack gap={5}>
          {open.map((match, i) => (
            <AppearFromBottom key={match.id} index={i}>
              <MatchCard
                match={match}
                userId={user?.id}
                busy={respond.isPending || complete.isPending || cancel.isPending}
                onOpenItem={(itemId) => router.push(`/item/${itemId}`)}
                onAccept={() =>
                  respond.mutate({ id: match.id, accept: true }, { onError })
                }
                onDecline={() =>
                  Alert.alert('Decline this trade?', 'This cannot be undone.', [
                    { text: 'Keep', style: 'cancel' },
                    {
                      text: 'Decline',
                      style: 'destructive',
                      onPress: () => respond.mutate({ id: match.id, accept: false }, { onError }),
                    },
                  ])
                }
                onComplete={() =>
                  Alert.alert(
                    'Confirm the swap happened?',
                    'Only confirm once you actually have their item.',
                    [
                      { text: 'Not yet', style: 'cancel' },
                      {
                        text: 'Confirm',
                        onPress: () => complete.mutate({ id: match.id }, { onError }),
                      },
                    ],
                  )
                }
                onCancel={() => cancel.mutate({ id: match.id }, { onError })}
              />
            </AppearFromBottom>
          ))}

          {done.length > 0 ? (
            <Stack gap={4}>
              <Divider />
              <Text variant="label" color="textFaint" uppercase>
                Past trades
              </Text>
              {done.map((match) => (
                <MatchCard
                  key={match.id}
                  match={match}
                  userId={user?.id}
                  busy={false}
                  onOpenItem={(itemId) => router.push(`/item/${itemId}`)}
                />
              ))}
            </Stack>
          ) : null}
        </Stack>
      )}

      {isRefetching ? null : null}
    </ScrollView>
  );
}

function MatchCard({
  match,
  userId,
  busy,
  onOpenItem,
  onAccept,
  onDecline,
  onComplete,
  onCancel,
}: {
  match: Match;
  userId: string | undefined;
  busy: boolean;
  onOpenItem: (itemId: string) => void;
  onAccept?: () => void;
  onDecline?: () => void;
  onComplete?: () => void;
  onCancel?: () => void;
}) {
  const theme = useTheme();
  const d = describeMatch(match, userId);

  const tone =
    match.status === 'completed'
      ? 'success'
      : match.status === 'cancelled'
        ? 'neutral'
        : d.canRespond && !d.iAccepted
          ? 'accent'
          : 'warning';

  return (
    <View
      style={{
        borderRadius: theme.radius.xl,
        backgroundColor: theme.colors.surface,
        borderWidth: 1,
        borderColor: theme.colors.border,
        overflow: 'hidden',
      }}
    >
      <Row
        justify="space-between"
        align="center"
        style={{
          paddingHorizontal: theme.space[4],
          paddingVertical: theme.space[3],
          backgroundColor: theme.colors.surfaceAlt,
        }}
      >
        <Row gap={2}>
          <Ionicons
            name={match.origin === 'direct' ? 'paper-plane' : 'sparkles'}
            size={13}
            color={theme.colors.textMuted}
          />
          <Text variant="caption" color="textMuted">
            {match.origin === 'direct'
              ? d.iProposed
                ? 'You offered'
                : 'Direct offer'
              : 'Matched'}
          </Text>
        </Row>
        <Badge label={d.headline} tone={tone} />
      </Row>

      {/* The two items, side by side — the whole trade at a glance. */}
      <Row gap={0} align="stretch" style={{ padding: theme.space[4] }}>
        <TradeSide
          label="You give"
          title={d.mine.title}
          accepted={d.iAccepted}
          onPress={() => onOpenItem(d.mine.id)}
        />
        <View style={{ justifyContent: 'center', paddingHorizontal: theme.space[2] }}>
          <View
            style={{
              width: 30,
              height: 30,
              borderRadius: theme.radius.full,
              backgroundColor: theme.colors.accentSubtle,
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <Ionicons name="swap-horizontal" size={15} color={theme.colors.accent} />
          </View>
        </View>
        <TradeSide
          label="You get"
          title={d.theirs.title}
          accepted={d.theyAccepted}
          onPress={() => onOpenItem(d.theirs.id)}
        />
      </Row>

      {match.message ? (
        <View
          style={{
            marginHorizontal: theme.space[4],
            marginBottom: theme.space[4],
            padding: theme.space[3],
            borderRadius: theme.radius.md,
            backgroundColor: theme.colors.bgSubtle,
          }}
        >
          <Text variant="caption" color="textMuted">
            “{match.message}”
          </Text>
        </View>
      ) : null}

      {d.isOpen ? (
        <Row gap={3} style={{ paddingHorizontal: theme.space[4], paddingBottom: theme.space[4] }}>
          {d.canRespond ? (
            <>
              <Button title="Decline" variant="secondary" onPress={onDecline} disabled={busy} />
              <Button
                title="Accept trade"
                style={{ flex: 1 }}
                loading={busy}
                onPress={onAccept}
              />
            </>
          ) : d.canComplete ? (
            <Button
              title="We swapped — confirm"
              block
              loading={busy}
              onPress={onComplete}
            />
          ) : (
            <Button title="Cancel trade" variant="danger" block onPress={onCancel} disabled={busy} />
          )}
        </Row>
      ) : null}
    </View>
  );
}

function TradeSide({
  label,
  title,
  accepted,
  onPress,
}: {
  label: string;
  title: string;
  accepted: boolean;
  onPress: () => void;
}) {
  const theme = useTheme();

  return (
    <PressableScale onPress={onPress} scaleTo={0.97} style={{ flex: 1 }}>
      <Stack gap={1}>
        <Row gap={1}>
          <Text variant="label" color="textFaint" uppercase>
            {label}
          </Text>
          {accepted ? (
            <Ionicons name="checkmark-circle" size={12} color={theme.colors.accent} />
          ) : null}
        </Row>
        <Text variant="bodyStrong" numberOfLines={2}>
          {title}
        </Text>
      </Stack>
    </PressableScale>
  );
}
