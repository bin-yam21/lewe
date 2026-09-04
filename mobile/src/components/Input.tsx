import { useState, type Ref } from 'react';
import { TextInput, View, type TextInputProps, type ViewStyle } from 'react-native';

import { useTheme } from '@/theme';

import { Text } from './Text';

export type InputProps = TextInputProps & {
  label?: string;
  /** Field-level validation message. Its presence drives the error styling. */
  error?: string;
  hint?: string;
  multiline?: boolean;
  containerStyle?: ViewStyle;
  /** React 19 passes ref as an ordinary prop — no forwardRef wrapper needed. */
  ref?: Ref<TextInput>;
};

export function Input({
  label,
  error,
  hint,
  multiline,
  containerStyle,
  style,
  onFocus,
  onBlur,
  ref,
  ...rest
}: InputProps) {
  const theme = useTheme();
  const [focused, setFocused] = useState(false);

  // `lineHeight` is pulled off the body type token deliberately. On Android a
  // lineHeight on a TextInput fights the fixed minHeight and vertical
  // alignment, and the entered text ends up clipped out of the visible box —
  // you type and nothing appears. Multiline fields need it for readable
  // wrapping and do not have the single-line centering conflict.
  const { lineHeight, ...bodyType } = theme.type.body;

  const borderColor = error
    ? theme.colors.danger
    : focused
      ? theme.colors.accent
      : theme.colors.border;

  return (
    <View style={[{ gap: theme.space[2] }, containerStyle]}>
      {label ? (
        <Text variant="caption" color="textMuted" style={{ fontWeight: '600' }}>
          {label}
        </Text>
      ) : null}

      <View
        style={{
          borderWidth: 1,
          borderColor,
          backgroundColor: theme.colors.surface,
          borderRadius: theme.radius.md,
          // Focus is signalled by border color alone. Adding/removing shadow
          // props here changed the native view's style on every focus change,
          // which can bounce focus back out of the field on Android.
        }}
      >
        <TextInput
          ref={ref}
          placeholderTextColor={theme.colors.textFaint}
          selectionColor={theme.colors.accent}
          multiline={multiline}
          onFocus={(e) => {
            setFocused(true);
            onFocus?.(e);
          }}
          onBlur={(e) => {
            setFocused(false);
            onBlur?.(e);
          }}
          style={[
            {
              paddingHorizontal: theme.space[4],
              paddingVertical: theme.space[3],
              minHeight: multiline ? 104 : 46,
              color: theme.colors.text,
              ...bodyType,
              ...(multiline ? { lineHeight, textAlignVertical: 'top' as const } : null),
            },
            style,
          ]}
          {...rest}
        />
      </View>

      {error ? (
        <Text variant="caption" color="danger">
          {error}
        </Text>
      ) : hint ? (
        <Text variant="caption" color="textFaint">
          {hint}
        </Text>
      ) : null}
    </View>
  );
}
