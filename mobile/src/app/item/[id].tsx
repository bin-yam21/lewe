import { Ionicons } from '@expo/vector-icons';
import { Image } from 'expo-image';
import { useLocalSearchParams, useRouter } from 'expo-router';
import { useState } from 'react';
import { Pressable, ScrollView, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { ApiError } from '@/api/client';
import { resolveImageUrl } from '@/api/config';
import { labelForCategory, labelForCondition } from '@/api/types';
import { Button } from '@/components/Button';
import { ImageCarousel } from '@/components/ImageCarousel';
import { OfferSheet } from '@/components/OfferSheet';
import { Text } from '@/components/Text';
import { Avatar, Badge, ErrorState, Skeleton } from '@/components/feedback';
import { Card, Divider, Row, Stack } from '@/components/layout';
import { AppearFromBottom, PressableScale } from '@/components/motion';
import { useAuth } from '@/hooks/useAuth';
import { useItem, useSimilarItems, useUserRating } from '@/hooks/useItems';
import { useTheme } from '@/theme';

export default function ItemDetail() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const theme = useTheme();
  const router = useRouter();
  const insets = useSafeAreaInsets();
  const { user } = useAuth();

  const { data: item, isLoading, isError, error, refetch } = useItem(id);
  const { data: rating } = useUserRating(item?.user_id);
  const { data: similar } = useSimilarItems(id);

  const [offerOpen, setOfferOpen] = useState(false);

  const isMine = !!item && item.user_id === user?.id;

  return (
    <View style={{ flex: 1, backgroundColor: theme.colors.bg }}>
      <ScrollView
        contentContainerStyle={{ paddingBottom: theme.space[9] }}
        showsVerticalScrollIndicator={false}
      >
        {/* Hero — the photos carry the listing, so they lead and go full bleed. */}
        <View>
          <ImageCarousel images={item?.images ?? []} height={360} />

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

              {!isMine ? (
                <Stack gap={3}>
                  <Button
                    title="Offer a trade"
                    block
                    size="lg"
                    icon={
                      <Ionicons name="swap-horizontal" size={19} color={theme.colors.onAccent} />
                    }
                    onPress={() => setOfferOpen(true)}
                  />
                  <Text variant="caption" color="textFaint" center>
                    Offer something of yours directly — no need to list it first.
                  </Text>
                </Stack>
              ) : (
                <View
                  style={{
                    backgroundColor: theme.colors.surfaceAlt,
                    borderRadius: theme.radius.md,
                    padding: theme.space[4],
                  }}
                >
                  <Row gap={3} align="flex-start">
                    <Ionicons
                      name="information-circle-outline"
                      size={18}
                      color={theme.colors.textMuted}
                    />
                    <Text variant="caption" color="textMuted" style={{ flex: 1 }}>
                      This is your listing. Offers from other people show up in your matches.
                    </Text>
                  </Row>
                </View>
              )}

              {/* Similar listings — the natural next thing to look at, and it
                  keeps a dead end from being a dead end. */}
              {similar && similar.items.length > 0 ? (
                <Stack gap={3}>
                  <Divider />
                  <Row justify="space-between" align="center">
                    <Text variant="heading">More like this</Text>
                    <Text variant="caption" color="textFaint">
                      {labelForCategory(item.category)}
                    </Text>
                  </Row>

                  <ScrollView
                    horizontal
                    showsHorizontalScrollIndicator={false}
                    contentContainerStyle={{ gap: theme.space[3], paddingRight: theme.space[5] }}
                    style={{ marginHorizontal: -theme.space[5], paddingHorizontal: theme.space[5] }}
                  >
                    {similar.items.map((s) => (
                      <PressableScale
                        key={s.id}
                        scaleTo={0.96}
                        onPress={() => router.push(`/item/${s.id}`)}
                      >
                        <Stack gap={2} style={{ width: 148 }}>
                          <View
                            style={{
                              width: 148,
                              height: 148,
                              borderRadius: theme.radius.lg,
                              overflow: 'hidden',
                              backgroundColor: theme.colors.surfaceAlt,
                              alignItems: 'center',
                              justifyContent: 'center',
                            }}
                          >
                            {resolveImageUrl(s.images?.[0]) ? (
                              <Image
                                source={{ uri: resolveImageUrl(s.images?.[0]) }}
                                style={{ width: '100%', height: '100%' }}
                                contentFit="cover"
                                transition={180}
                              />
                            ) : (
                              <Ionicons
                                name="image-outline"
                                size={20}
                                color={theme.colors.textFaint}
                              />
                            )}
                          </View>
                          <Text variant="caption" numberOfLines={1} style={{ fontWeight: '600' }}>
                            {s.title}
                          </Text>
                          <Text variant="caption" color="textFaint" numberOfLines={1}>
                            {labelForCondition(s.condition)}
                          </Text>
                        </Stack>
                      </PressableScale>
                    ))}
                  </ScrollView>
                </Stack>
              ) : null}
            </>
          ) : null}
        </Stack>
      </ScrollView>

      {item && !isMine ? (
        <OfferSheet
          visible={offerOpen}
          onClose={() => setOfferOpen(false)}
          target={item}
          onOffered={() => router.push('/(tabs)/matches')}
        />
      ) : null}
    </View>
  );
}
