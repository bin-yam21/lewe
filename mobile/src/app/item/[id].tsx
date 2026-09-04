import { Ionicons } from '@expo/vector-icons';
import { Image } from 'expo-image';
import { useLocalSearchParams, useRouter } from 'expo-router';
import { Pressable, ScrollView, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { ApiError } from '@/api/client';
import { labelForCategory, labelForCondition } from '@/api/types';
import { Button } from '@/components/Button';
import { Text } from '@/components/Text';
import { Avatar, Badge, ErrorState, Skeleton } from '@/components/feedback';
import { Card, Divider, Row, Stack } from '@/components/layout';
import { useAuth } from '@/hooks/useAuth';
import { useItem, useUserRating } from '@/hooks/useItems';
import { useTheme } from '@/theme';

export default function ItemDetail() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const theme = useTheme();
  const router = useRouter();
  const insets = useSafeAreaInsets();
  const { user } = useAuth();

  const { data: item, isLoading, isError, error, refetch } = useItem(id);
  const { data: rating } = useUserRating(item?.user_id);

  const isMine = !!item && item.user_id === user?.id;
  const cover = item?.images?.[0];

  return (
    <View style={{ flex: 1, backgroundColor: theme.colors.bg }}>
      <ScrollView
        contentContainerStyle={{ paddingBottom: theme.space[9] }}
        showsVerticalScrollIndicator={false}
      >
        {/* Hero */}
        <View
          style={{
            height: 280,
            backgroundColor: theme.colors.surfaceAlt,
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          {cover ? (
            <Image
              source={{ uri: cover }}
              style={{ width: '100%', height: '100%' }}
              contentFit="cover"
              transition={200}
            />
          ) : (
            <Ionicons name="image-outline" size={40} color={theme.colors.textFaint} />
          )}

          <Pressable
            onPress={() => router.back()}
            hitSlop={12}
            style={{
              position: 'absolute',
              top: insets.top + theme.space[2],
              left: theme.space[4],
              width: 38,
              height: 38,
              borderRadius: theme.radius.full,
              backgroundColor: theme.colors.bg,
              borderWidth: 1,
              borderColor: theme.colors.border,
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <Ionicons name="arrow-back" size={20} color={theme.colors.text} />
          </Pressable>
        </View>

        <Stack gap={6} style={{ padding: theme.space[5] }}>
          {isLoading ? (
            <Stack gap={4}>
              <Skeleton height={28} width="70%" />
              <Skeleton height={18} width="40%" />
              <Skeleton height={90} />
            </Stack>
          ) : isError ? (
            <ErrorState
              message={error instanceof ApiError ? error.message : 'Could not load this item.'}
              onRetry={() => refetch()}
            />
          ) : item ? (
            <>
              <Stack gap={3}>
                <Row gap={2} wrap>
                  <Badge label={labelForCondition(item.condition)} />
                  <Badge label={labelForCategory(item.category)} tone="accent" />
                  {item.status !== 'active' ? (
                    <Badge
                      label={item.status}
                      tone={item.status === 'exchanged' ? 'success' : 'warning'}
                    />
                  ) : null}
                </Row>

                <Text variant="title">{item.title}</Text>

                {item.location ? (
                  <Row gap={1}>
                    <Ionicons name="location-outline" size={14} color={theme.colors.textFaint} />
                    <Text variant="caption" color="textFaint">
                      {item.location}
                    </Text>
                  </Row>
                ) : null}
              </Stack>

              <Text variant="body" color="textMuted">
                {item.description}
              </Text>

              <Divider />

              {/* Wants — the reason a trade is or isn't possible. */}
              <Stack gap={3}>
                <Text variant="heading">Looking to trade for</Text>
                <Stack gap={2}>
                  {item.wants.map((want) => (
                    <Card key={want.id}>
                      <Row gap={3}>
                        <View
                          style={{
                            width: 34,
                            height: 34,
                            borderRadius: theme.radius.sm,
                            backgroundColor: theme.colors.accentSubtle,
                            alignItems: 'center',
                            justifyContent: 'center',
                          }}
                        >
                          <Ionicons name="swap-horizontal" size={17} color={theme.colors.accent} />
                        </View>
                        <Stack gap={1} style={{ flex: 1 }}>
                          <Text variant="bodyStrong">{labelForCategory(want.category)}</Text>
                          {want.description ? (
                            <Text variant="caption" color="textMuted">
                              {want.description}
                            </Text>
                          ) : null}
                        </Stack>
                      </Row>
                    </Card>
                  ))}
                </Stack>
              </Stack>

              <Divider />

              <Stack gap={3}>
                <Text variant="heading">Owner</Text>
                <Row gap={3}>
                  <Avatar name={isMine ? (user?.full_name ?? 'You') : 'Trader'} />
                  <Stack gap={1} style={{ flex: 1 }}>
                    <Text variant="bodyStrong">{isMine ? 'You' : 'Lewe trader'}</Text>
                    <Row gap={1}>
                      <Ionicons name="star" size={12} color={theme.colors.warning} />
                      <Text variant="caption" color="textMuted">
                        {rating && rating.count > 0
                          ? `${rating.average.toFixed(1)} · ${rating.count} rating${rating.count === 1 ? '' : 's'}`
                          : 'No ratings yet'}
                      </Text>
                    </Row>
                  </Stack>
                </Row>
                {/* Owner names need GET /users/{id} — roadmap A1. */}
              </Stack>

              <View
                style={{
                  backgroundColor: theme.colors.surfaceAlt,
                  borderRadius: theme.radius.md,
                  padding: theme.space[4],
                }}
              >
                <Row gap={3} align="flex-start">
                  <Ionicons name="information-circle-outline" size={18} color={theme.colors.textMuted} />
                  <Text variant="caption" color="textMuted" style={{ flex: 1 }}>
                    {isMine
                      ? 'Matches are found from your own listing. Open it from your profile to search for trades.'
                      : 'To trade, list an item this owner wants — Lewe pairs you automatically when the interest is mutual.'}
                  </Text>
                </Row>
              </View>

              {!isMine ? (
                <Button
                  title="List something to trade"
                  block
                  size="lg"
                  onPress={() => router.push('/(tabs)/create')}
                />
              ) : null}
            </>
          ) : null}
        </Stack>
      </ScrollView>
    </View>
  );
}
