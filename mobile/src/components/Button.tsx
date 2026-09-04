import * as Haptics from 'expo-haptics';
import {
  ActivityIndicator,
  Platform,
  Pressable,
  View,
  type PressableProps,
  type ViewStyle,
} from 'react-native';

import { useTheme } from '@/theme';

import { Text } from './Text';

type Variant = 'primary' | 'secondary' | 'ghost' | 'danger';
type Size = 'sm' | 'md' | 'lg';

export type ButtonProps = Omit<PressableProps, 'style' | 'children'> & {
  title: string;
  variant?: Variant;
  size?: Size;
  loading?: boolean;
  /** Stretch to the container width — the default for form and sheet actions. */
  block?: boolean;
  icon?: React.ReactNode;
  style?: ViewStyle;
};

const HEIGHT: Record<Size, number> = { sm: 36, md: 46, lg: 54 };

export function Button({
  title,
  variant = 'primary',
  size = 'md',
  loading = false,
  block = false,
  icon,
  disabled,
  onPress,
  style,
  ...rest
}: ButtonProps) {
  const theme = useTheme();
  const isDisabled = disabled || loading;

  const colorsFor = (pressed: boolean) => {
    switch (variant) {
      case 'primary':
        return {
          bg: pressed ? theme.colors.accentPressed : theme.colors.accent,
          fg: theme.colors.onAccent,
          border: 'transparent',
        };
      case 'secondary':
        return {
          bg: pressed ? theme.colors.surfacePressed : theme.colors.surfaceAlt,
          fg: theme.colors.text,
          border: theme.colors.border,
        };
      case 'danger':
        return {
          bg: pressed ? theme.colors.dangerSubtle : 'transparent',
          fg: theme.colors.danger,
          border: theme.colors.border,
        };
      case 'ghost':
        return {
          bg: pressed ? theme.colors.surfaceAlt : 'transparent',
          fg: theme.colors.accent,
          border: 'transparent',
        };
    }
  };

  return (
    <Pressable
      accessibilityRole="button"
      accessibilityState={{ disabled: !!isDisabled, busy: loading }}
      disabled={isDisabled}
      onPress={(e) => {
        // A short tap tick makes the whole app feel more responsive than any
        // transition animation would.
        if (Platform.OS !== 'web') {
          Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light).catch(() => {});
        }
        onPress?.(e);
      }}
      style={({ pressed }) => {
        const c = colorsFor(pressed);
        return [
          {
            height: HEIGHT[size],
            paddingHorizontal: size === 'sm' ? theme.space[3] : theme.space[5],
            borderRadius: theme.radius.md,
            backgroundColor: c.bg,
            borderWidth: variant === 'primary' || variant === 'ghost' ? 0 : 1,
            borderColor: c.border,
            flexDirection: 'row',
            alignItems: 'center',
            justifyContent: 'center',
            gap: theme.space[2],
            alignSelf: block ? 'stretch' : 'flex-start',
            opacity: isDisabled ? 0.45 : 1,
          } satisfies ViewStyle,
          style,
        ];
      }}
      {...rest}
    >
      {({ pressed }: { pressed: boolean }) => {
        const c = colorsFor(pressed);
        return loading ? (
          <ActivityIndicator size="small" color={c.fg} />
        ) : (
          <>
            {icon ? <View>{icon}</View> : null}
            <Text
              variant={size === 'sm' ? 'caption' : 'bodyStrong'}
              style={{ color: c.fg, fontWeight: '600' }}
              numberOfLines={1}
            >
              {title}
            </Text>
          </>
        );
      }}
    </Pressable>
  );
}
