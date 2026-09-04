import { useMemo, type ReactNode } from 'react';
import { Pressable, ScrollView, View, type ViewStyle } from 'react-native';
import { SafeAreaView, type Edge } from 'react-native-safe-area-context';

import { useTheme } from '@/theme';

/** Module-level so the default is referentially stable across renders. */
const TOP_EDGE: Edge[] = ['top'];

/**
 * Every route's outermost element. Owns the safe area and the background token
 * so no screen has to think about either.
 *
 * Note the memoisation below: a fresh `contentContainerStyle` object on every
 * render makes the ScrollView re-layout on every keystroke, which on Android
 * fights the keyboard and can blur the focused input. Screens holding text
 * inputs re-render on each character, so this has to stay stable.
 */
export function Screen({
  children,
  scroll = false,
  padded = true,
  tinted = false,
  edges = TOP_EDGE,
  grow = false,
  contentStyle,
}: {
  children: ReactNode;
  scroll?: boolean;
  padded?: boolean;
  /** Use the subtly tinted ground — for screens made of flat cards. */
  tinted?: boolean;
  edges?: Edge[];
  /** Let scroll content fill the viewport, so footers can sit at the bottom. */
  grow?: boolean;
  contentStyle?: ViewStyle;
}) {
  const theme = useTheme();
  const background = tinted ? theme.colors.bgSubtle : theme.colors.bg;

  const inner = useMemo<ViewStyle>(
    () => ({
      flex: scroll ? undefined : 1,
      flexGrow: scroll && grow ? 1 : undefined,
      padding: padded ? theme.space[5] : 0,
      ...contentStyle,
    }),
    // `contentStyle` is intentionally not a dependency: callers pass object
    // literals, which would defeat the memo entirely. Pass a stable reference
    // (a module constant or a useMemo) if a screen needs a dynamic one.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [scroll, grow, padded, theme.space],
  );

  const containerStyle = useMemo(
    () => ({ flex: 1, backgroundColor: background }),
    [background],
  );

  return (
    <SafeAreaView style={containerStyle} edges={edges}>
      {scroll ? (
        <ScrollView
          contentContainerStyle={inner}
          keyboardShouldPersistTaps="handled"
          keyboardDismissMode="none"
          // Lets iOS handle keyboard insets natively instead of wrapping the
          // screen in a KeyboardAvoidingView that competes with the ScrollView.
          automaticallyAdjustKeyboardInsets
          showsVerticalScrollIndicator={false}
        >
          {children}
        </ScrollView>
      ) : (
        <View style={inner}>{children}</View>
      )}
    </SafeAreaView>
  );
}

/** A flat surface with a hairline border — no shadow, by design. */
export function Card({
  children,
  onPress,
  style,
  padded = true,
}: {
  children: ReactNode;
  onPress?: () => void;
  style?: ViewStyle;
  padded?: boolean;
}) {
  const theme = useTheme();

  const base = (pressed: boolean): ViewStyle => ({
    backgroundColor: pressed ? theme.colors.surfacePressed : theme.colors.surface,
    borderRadius: theme.radius.lg,
    borderWidth: 1,
    borderColor: theme.colors.border,
    padding: padded ? theme.space[4] : 0,
    overflow: 'hidden',
    ...style,
  });

  if (!onPress) return <View style={base(false)}>{children}</View>;

  return (
    <Pressable onPress={onPress} style={({ pressed }) => base(pressed)}>
      {children}
    </Pressable>
  );
}

export function Row({
  children,
  gap = 3,
  align = 'center',
  justify = 'flex-start',
  wrap = false,
  style,
}: {
  children: ReactNode;
  gap?: 0 | 1 | 2 | 3 | 4 | 5 | 6;
  align?: ViewStyle['alignItems'];
  justify?: ViewStyle['justifyContent'];
  wrap?: boolean;
  style?: ViewStyle;
}) {
  const theme = useTheme();
  return (
    <View
      style={[
        {
          flexDirection: 'row',
          alignItems: align,
          justifyContent: justify,
          gap: theme.space[gap],
          flexWrap: wrap ? 'wrap' : 'nowrap',
        },
        style,
      ]}
    >
      {children}
    </View>
  );
}

export function Stack({
  children,
  gap = 4,
  style,
}: {
  children: ReactNode;
  gap?: 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8;
  style?: ViewStyle;
}) {
  const theme = useTheme();
  return <View style={[{ gap: theme.space[gap] }, style]}>{children}</View>;
}

export function Divider() {
  const theme = useTheme();
  return <View style={{ height: 1, backgroundColor: theme.colors.border }} />;
}
