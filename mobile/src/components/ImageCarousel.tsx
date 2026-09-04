import { Ionicons } from '@expo/vector-icons';
import { Image } from 'expo-image';
import { useState } from 'react';
import { FlatList, View, useWindowDimensions } from 'react-native';
import Animated, { FadeIn } from 'react-native-reanimated';

import { resolveImageUrl } from '@/api/config';
import { useTheme } from '@/theme';

import { Text } from './Text';
import { Row } from './layout';

/**
 * Full-bleed, swipeable photos for the item detail screen.
 *
 * Paging rather than free scroll: each photo is a separate piece of evidence
 * about the item, so it gets the whole viewport and settles cleanly.
 */
export function ImageCarousel({ images, height = 340 }: { images: string[]; height?: number }) {
  const theme = useTheme();
  const { width } = useWindowDimensions();
  const [index, setIndex] = useState(0);

  if (!images || images.length === 0) {
    return (
      <View
        style={{
          height,
          width,
          backgroundColor: theme.colors.surfaceAlt,
          alignItems: 'center',
          justifyContent: 'center',
          gap: theme.space[2],
        }}
      >
        <Ionicons name="image-outline" size={34} color={theme.colors.textFaint} />
        <Text variant="caption" color="textFaint">
          No photos for this item
        </Text>
      </View>
    );
  }

  return (
    <View>
      <FlatList
        data={images}
        keyExtractor={(uri, i) => `${uri}-${i}`}
        horizontal
        pagingEnabled
        showsHorizontalScrollIndicator={false}
        onMomentumScrollEnd={(e) =>
          setIndex(Math.round(e.nativeEvent.contentOffset.x / width))
        }
        renderItem={({ item }) => (
          <Animated.View entering={FadeIn.duration(260)}>
            <Image
              source={{ uri: resolveImageUrl(item) }}
              style={{ width, height }}
              contentFit="cover"
              transition={220}
            />
          </Animated.View>
        )}
      />

      {images.length > 1 ? (
        <Row
          gap={2}
          justify="center"
          style={{
            position: 'absolute',
            bottom: theme.space[4],
            left: 0,
            right: 0,
          }}
        >
          {images.map((uri, i) => (
            <View
              key={`dot-${uri}-${i}`}
              style={{
                width: i === index ? 20 : 6,
                height: 6,
                borderRadius: 3,
                backgroundColor: i === index ? '#FFFFFF' : 'rgba(255,255,255,0.55)',
              }}
            />
          ))}
        </Row>
      ) : null}
    </View>
  );
}
