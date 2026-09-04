import { Ionicons } from '@expo/vector-icons';
import { Image } from 'expo-image';
import { View } from 'react-native';

import { labelForCategory, labelForCondition, type Item } from '@/api/types';
import { useTheme } from '@/theme';

import { Text } from './Text';
import { Badge } from './feedback';
import { Card, Row, Stack } from './layout';

/**
 * One listing in the feed. Leads with the photo when there is one, and always
 * shows what the owner wants back — that is the information that decides whether
 * a trade is even possible, so it should never be a tap away.
 */
export function ItemCard({ item, onPress }: { item: Item; onPress: () => void }) {
  const theme = useTheme();
  const cover = item.images?.[0];
  const wants = item.wants ?? [];

  return (
    <Card onPress={onPress} padded={false}>
      <Row gap={0} align="stretch">
        <View
          style={{
            width: 104,
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
              transition={160}
            />
          ) : (
            <Ionicons name="image-outline" size={22} color={theme.colors.textFaint} />
          )}
        </View>

        <Stack gap={2} style={{ flex: 1, padding: theme.space[4] }}>
          <Row justify="space-between" align="flex-start" gap={2}>
            <Text variant="bodyStrong" numberOfLines={1} style={{ flex: 1 }}>
              {item.title}
            </Text>
            <Badge label={labelForCondition(item.condition)} />
          </Row>

          <Text variant="caption" color="textMuted" numberOfLines={2}>
            {item.description}
          </Text>

          <Row gap={2} wrap>
            <Row gap={1}>
              <Ionicons name="pricetag-outline" size={12} color={theme.colors.textFaint} />
              <Text variant="caption" color="textFaint">
                {labelForCategory(item.category)}
              </Text>
            </Row>

            {wants.length > 0 ? (
              <Row gap={1}>
                <Ionicons name="arrow-forward" size={12} color={theme.colors.accent} />
                <Text variant="caption" color="accent" numberOfLines={1}>
                  wants {wants.map((w) => labelForCategory(w.category)).join(', ')}
                </Text>
              </Row>
            ) : null}
          </Row>
        </Stack>
      </Row>
    </Card>
  );
}
