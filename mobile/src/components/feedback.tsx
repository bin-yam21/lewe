import { Ionicons } from '@expo/vector-icons';
import { useEffect, useRef, type ReactNode } from 'react';
import { Animated, Pressable, View, type ViewStyle } from 'react-native';

import { useTheme, type ColorName } from '@/theme';

import { Text } from './Text';

/** A filter pill. Selected state is the one place accent color earns its keep. */
export function Chip({
  label,
  selected = false,
  onPress,
}: {
  label: string;
  selected?: boolean;
  onPress?: () => void;
}) {
  const theme = useTheme();

  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="button"
      accessibilityState={{ selected }}
      style={({ pressed }) => ({
        paddingHorizontal: theme.space[3],
        paddingVertical: theme.space[2],
        borderRadius: theme.radius.full,
        borderWidth: 1,
        borderColor: selected ? theme.colors.accent : theme.colors.border,
        backgroundColor: selected
          ? theme.colors.accent
          : pressed
            ? theme.colors.surfacePressed
            : theme.colors.surface,
      })}
    >
      <Text
        variant="caption"
        style={{
          fontWeight: '600',
          color: selected ? theme.colors.onAccent : theme.colors.textMuted,
        }}
      >
        {label}
      </Text>
    </Pressable>
  );
}

/** A static status label — condition, item state, exchange method. */
export function Badge({
  label,
  tone = 'neutral',
}: {
  label: string;
  tone?: 'neutral' | 'accent' | 'success' | 'warning' | 'danger';
}) {
  const theme = useTheme();

  const map: Record<string, { bg: string; fg: ColorName }> = {
    neutral: { bg: theme.colors.surfaceAlt, fg: 'textMuted' },
    accent: { bg: theme.colors.accentSubtle, fg: 'accent' },
    success: { bg: theme.colors.successSubtle, fg: 'success' },
    warning: { bg: theme.colors.warningSubtle, fg: 'warning' },
    danger: { bg: theme.colors.dangerSubtle, fg: 'danger' },
  };
  const c = map[tone];

  return (
    <View
      style={{
        backgroundColor: c.bg,
        paddingHorizontal: theme.space[2],
        paddingVertical: 3,
        borderRadius: theme.radius.sm,
        alignSelf: 'flex-start',
      }}
    >
      <Text variant="label" color={c.fg} uppercase>
        {label}
      </Text>
    </View>
  );
}

/** A pulsing placeholder. Loading a shape beats spinning a wheel. */
export function Skeleton({
  width = '100%',
  height = 16,
  radius,
  style,
}: {
  width?: number | `${number}%`;
  height?: number;
  radius?: number;
  style?: ViewStyle;
}) {
  const theme = useTheme();
  const opacity = useRef(new Animated.Value(0.5)).current;

  useEffect(() => {
    const loop = Animated.loop(
      Animated.sequence([
        Animated.timing(opacity, { toValue: 1, duration: 700, useNativeDriver: true }),
        Animated.timing(opacity, { toValue: 0.45, duration: 700, useNativeDriver: true }),
      ]),
    );
    loop.start();
    return () => loop.stop();
  }, [opacity]);

  return (
    <Animated.View
      style={[
        {
          width,
          height,
          opacity,
          borderRadius: radius ?? theme.radius.sm,
          backgroundColor: theme.colors.skeleton,
        },
        style,
      ]}
    />
  );
}

export function EmptyState({
  icon = 'cube-outline',
  title,
  message,
  action,
}: {
  icon?: keyof typeof Ionicons.glyphMap;
  title: string;
  message?: string;
  action?: ReactNode;
}) {
  const theme = useTheme();

  return (
    <View
      style={{
        alignItems: 'center',
        justifyContent: 'center',
        paddingVertical: theme.space[9],
        paddingHorizontal: theme.space[5],
        gap: theme.space[3],
      }}
    >
      <View
        style={{
          width: 56,
          height: 56,
          borderRadius: theme.radius.full,
          backgroundColor: theme.colors.surfaceAlt,
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        <Ionicons name={icon} size={26} color={theme.colors.textFaint} />
      </View>

      <Text variant="heading" center>
        {title}
      </Text>

      {message ? (
        <Text variant="body" color="textMuted" center style={{ maxWidth: 300 }}>
          {message}
        </Text>
      ) : null}

      {action ? <View style={{ marginTop: theme.space[2] }}>{action}</View> : null}
    </View>
  );
}

/** Inline error with a retry — used wherever a query can fail. */
export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  const theme = useTheme();

  return (
    <View style={{ padding: theme.space[5], alignItems: 'center', gap: theme.space[3] }}>
      <Ionicons name="cloud-offline-outline" size={28} color={theme.colors.danger} />
      <Text variant="body" color="textMuted" center>
        {message}
      </Text>
      {onRetry ? (
        <Pressable onPress={onRetry} hitSlop={8}>
          <Text variant="bodyStrong" color="accent">
            Try again
          </Text>
        </Pressable>
      ) : null}
    </View>
  );
}

/** Initials avatar — there is no avatar upload yet (roadmap A1). */
export function Avatar({ name, size = 40 }: { name: string; size?: number }) {
  const theme = useTheme();

  const initials = name
    .split(' ')
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase())
    .join('');

  return (
    <View
      style={{
        width: size,
        height: size,
        borderRadius: theme.radius.full,
        backgroundColor: theme.colors.accentSubtle,
        borderWidth: 1,
        borderColor: theme.colors.accentBorder,
        alignItems: 'center',
        justifyContent: 'center',
      }}
    >
      <Text style={{ color: theme.colors.accent, fontWeight: '700', fontSize: size * 0.36 }}>
        {initials || '?'}
      </Text>
    </View>
  );
}
