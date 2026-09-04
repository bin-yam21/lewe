import { Text as RNText, type TextProps as RNTextProps } from 'react-native';

import { useTheme, type ColorName, type TypeVariant } from '@/theme';

export type TextProps = RNTextProps & {
  variant?: TypeVariant;
  color?: ColorName;
  /** Shorthand for `textAlign: 'center'` — by far the most common override. */
  center?: boolean;
  uppercase?: boolean;
};

/**
 * The only way text enters the app. Takes a type-scale variant and a token
 * color name, so no screen ever picks a font size or hex value by hand.
 */
export function Text({
  variant = 'body',
  color = 'text',
  center,
  uppercase,
  style,
  ...rest
}: TextProps) {
  const theme = useTheme();

  return (
    <RNText
      style={[
        theme.type[variant],
        { color: theme.colors[color] },
        center && { textAlign: 'center' },
        uppercase && { textTransform: 'uppercase' },
        style,
      ]}
      {...rest}
    />
  );
}
