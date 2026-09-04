import { Ionicons } from '@expo/vector-icons';
import { Image } from 'expo-image';
import { View } from 'react-native';

import { resolveImageUrl } from '@/api/config';
import { labelForCategory, labelForCondition, type Item } from '@/api/types';
import { useTheme } from '@/theme';

import { Text } from './Text';
import { Row, Stack } from './layout';
import { PressableScale } from './motion';

/**
 * One listing in the feed, built photo-first.
 *
 * The image is the headline, not a thumbnail: it is the only proof the item is
 * real. Underneath it the card answers the one question that decides whether a
 * trade is even possible — what this person wants back — so nobody has to open
 * a listing to rule it out.
 */
export function ItemCard({ item, onPress }: { item: Item; onPress: () => void }) {
  const theme = useTheme();
  const cover = resolveImageUrl(item.images?.[0]);
  const extraPhotos = Math.max(0, (item.images?.length ?? 0) - 1);
  const wants = item.wants ?? [];

  return (
    <PressableScale onPress={onPress} scaleTo={0.985}>
      <View
        style={{
          borderRadius: theme.radius.xl,
          backgroundColor: theme.colors.surface,
          borderWidth: 1,
          borderColor: theme.colors.border,
          overflow: 'hidden',
          ...theme.shadow('card'),
        }}
      >
        {/* Photo */}
        <View
          style={{
            height: 220,
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
              transition={220}
            />
          ) : (
            <Stack gap={2} style={{ alignItems: 'center' }}>
              <Ionicons name="image-outline" size={26} color={theme.colors.textFaint} />
              <Text variant="caption" color="textFaint">
                No photo yet
              </Text>
            </Stack>
          )}

          {/* Condition sits on the photo — it is part of what you are judging. */}
          <View
            style={{
              position: 'absolute',
              top: theme.space[3],
              left: theme.space[3],
              backgroundColor: theme.colors.surface,
              paddingHorizontal: theme.space[3],
              paddingVertical: theme.space[1] + 2,
              borderRadius: theme.radius.full,
            }}
          >
            <Text variant="label" uppercase color="text">
              {labelForCondition(item.condition)}
            </Text>
          </View>

          {extraPhotos > 0 ? (
            <Row
              gap={1}
              style={{
                position: 'absolute',
                top: theme.space[3],
                right: theme.space[3],
                backgroundColor: theme.colors.scrim,
                paddingHorizontal: theme.space[2] + 2,
                paddingVertical: theme.space[1] + 2,
                borderRadius: theme.radius.full,
              }}
            >
              <Ionicons name="images" size={11} color="#FFFFFF" />
              <Text variant="label" style={{ color: '#FFFFFF' }}>
                +{extraPhotos}
              </Text>
            </Row>
          ) : null}
        </View>

        {/* Detail */}
        <Stack gap={3} style={{ padding: theme.space[4] }}>
          <Stack gap={1}>
            <Text variant="heading" numberOfLines={1}>
              {item.title}
            </Text>
            <Row gap={2}>
              <Text variant="caption" color="textMuted">
                {labelForCategory(item.category)}
              </Text>
              {item.location ? (
                <>
                  <Text variant="caption" color="textFaint">
                    ·
                  </Text>
                  <Row gap={1}>
                    <Ionicons name="location" size={11} color={theme.colors.textFaint} />
                    <Text variant="caption" color="textFaint" numberOfLines={1}>
                      {item.location}
                    </Text>
                  </Row>
                </>
              ) : null}
            </Row>
          </Stack>

          {wants.length > 0 ? (
            <Row
              gap={2}
              align="center"
              style={{
                backgroundColor: theme.colors.accentSubtle,
                paddingHorizontal: theme.space[3],
                paddingVertical: theme.space[2] + 2,
                borderRadius: theme.radius.md,
              }}
            >
              <Ionicons name="swap-horizontal" size={15} color={theme.colors.accent} />
              <Text variant="caption" color="accent" numberOfLines={1} style={{ flex: 1 }}>
                Wants {wants.map((w) => labelForCategory(w.category)).join(' · ')}
              </Text>
            </Row>
          ) : null}
        </Stack>
      </View>
    </PressableScale>
  );
}
