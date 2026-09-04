import { createContext, useContext, useMemo, type ReactNode } from 'react';
import { useColorScheme } from 'react-native';

import { motion, palette, radius, space, type, type Colors } from './tokens';

export * from './tokens';

type Theme = {
  colors: Colors;
  space: typeof space;
  radius: typeof radius;
  type: typeof type;
  motion: typeof motion;
  scheme: 'light' | 'dark';
  isDark: boolean;
};

const ThemeContext = createContext<Theme | null>(null);

export function ThemeProvider({ children }: { children: ReactNode }) {
  const scheme = useColorScheme() === 'dark' ? 'dark' : 'light';

  const value = useMemo<Theme>(
    () => ({
      colors: palette[scheme],
      space,
      radius,
      type,
      motion,
      scheme,
      isDark: scheme === 'dark',
    }),
    [scheme],
  );

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): Theme {
  const ctx = useContext(ThemeContext);
  if (!ctx) throw new Error('useTheme must be used inside <ThemeProvider>');
  return ctx;
}
