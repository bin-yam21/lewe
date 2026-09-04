import { Ionicons } from '@expo/vector-icons';
import { View } from 'react-native';

import { useTheme } from '@/theme';

import { Text } from './Text';

/**
 * The Lewe mark: two tiles caught mid-swap.
 *
 * Built from views rather than an SVG asset so it inherits theme colors, scales
 * cleanly at any size, and costs no extra dependency. The offset back tile is
 * the whole idea of the product in one shape — two things changing places.
 */
export function LogoMark({ size = 44 }: { size?: number }) {
  const theme = useTheme();
  const tile = size * 0.78;
  const offset = size - tile;

  return (
    <View style={{ width: size, height: size }}>
      {/* Back tile — the other person's item. */}
      <View
        style={{
          position: 'absolute',
          right: 0,
          bottom: 0,
          width: tile,
          height: tile,
          borderRadius: tile * 0.32,
          backgroundColor: theme.colors.sun,
        }}
      />
      {/* Front tile — yours, carrying the swap glyph. */}
      <View
        style={{
          position: 'absolute',
          left: 0,
          top: 0,
          width: tile,
          height: tile,
          borderRadius: tile * 0.32,
          backgroundColor: theme.colors.accent,
          alignItems: 'center',
          justifyContent: 'center',
          borderWidth: offset * 0.5,
          borderColor: theme.colors.bg,
        }}
      >
        <Ionicons name="swap-horizontal" size={tile * 0.52} color={theme.colors.onAccent} />
      </View>
    </View>
  );
}

/** Mark plus wordmark, for the welcome screen and headers. */
export function Logo({
  size = 40,
  showWordmark = true,
}: {
  size?: number;
  showWordmark?: boolean;
}) {
  const theme = useTheme();

  return (
    <View style={{ flexDirection: 'row', alignItems: 'center', gap: theme.space[3] }}>
      <LogoMark size={size} />
      {showWordmark ? (
        <Text
          style={{
            fontSize: size * 0.62,
            fontWeight: '800',
            letterSpacing: -size * 0.03,
            color: theme.colors.text,
          }}
        >
          lewe
        </Text>
      ) : null}
    </View>
  );
}
