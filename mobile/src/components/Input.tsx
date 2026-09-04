import { forwardRef, useState } from 'react';
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
};

export const Input = forwardRef<TextInput, InputProps>(function Input(
  { label, error, hint, multiline, containerStyle, style, onFocus, onBlur, ...rest },
  ref,
) {
  const theme = useTheme();
  const [focused, setFocused] = useState(false);

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
          // A second ring on focus reads as attention without shifting layout.
          ...(focused && !error
            ? { shadowColor: theme.colors.accent, shadowOpacity: 0.18, shadowRadius: 6 }
            : null),
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
              textAlignVertical: multiline ? 'top' : 'center',
              ...theme.type.body,
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
});
