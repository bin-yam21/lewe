/**
 * Design tokens for Lewe.
 *
 * Direction: clean & modern. High-contrast neutrals, a single vivid accent used
 * only for primary actions and active state, flat surfaces on a tinted ground
 * rather than cards floating on shadows.
 *
 * Nothing in the app hardcodes a color. Screens read these through `useTheme()`,
 * which is what makes dark mode a token swap instead of a rewrite.
 */

export const palette = {
  light: {
    /** App background — the paper everything sits on. */
    bg: '#FFFFFF',
    /** Slightly tinted ground, so flat cards separate without shadows. */
    bgSubtle: '#F5F6F8',
    surface: '#FFFFFF',
    surfaceAlt: '#F0F2F5',
    surfacePressed: '#E8EBF0',

    border: '#E4E7EC',
    borderStrong: '#D0D5DD',

    text: '#0B0B0F',
    textMuted: '#667085',
    textFaint: '#98A2B3',
    /** Text/icons drawn on top of the accent color. */
    onAccent: '#FFFFFF',

    accent: '#4F46E5',
    accentPressed: '#4338CA',
    accentSubtle: '#EEEEFF',
    accentBorder: '#C7C5FF',

    success: '#067647',
    successSubtle: '#E6F6EE',
    warning: '#B54708',
    warningSubtle: '#FEF0C7',
    danger: '#D92D20',
    dangerSubtle: '#FEE4E2',

    overlay: 'rgba(11, 11, 15, 0.55)',
    skeleton: '#EAECF0',
  },
  dark: {
    bg: '#0B0B0F',
    bgSubtle: '#101017',
    surface: '#16161D',
    surfaceAlt: '#1E1E27',
    surfacePressed: '#262630',

    border: '#26262F',
    borderStrong: '#383843',

    text: '#F5F6F8',
    textMuted: '#9BA1AC',
    textFaint: '#6B7280',
    onAccent: '#FFFFFF',

    accent: '#6366F1',
    accentPressed: '#575AE8',
    accentSubtle: '#1A1A2E',
    accentBorder: '#33335C',

    success: '#3DD68C',
    successSubtle: '#0C1F17',
    warning: '#F59E0B',
    warningSubtle: '#241A0B',
    danger: '#F97066',
    dangerSubtle: '#2A1211',

    overlay: 'rgba(0, 0, 0, 0.7)',
    skeleton: '#1E1E27',
  },
} as const;

/**
 * Widened to `string`: `as const` above gives each hex a literal type, which
 * would make the light and dark palettes mutually incompatible.
 */
export type Colors = { [K in keyof typeof palette.light]: string };
export type ColorName = keyof Colors;

/** 4px base scale. Every margin and pad in the app comes from here. */
export const space = {
  0: 0,
  1: 4,
  2: 8,
  3: 12,
  4: 16,
  5: 20,
  6: 24,
  7: 32,
  8: 40,
  9: 48,
  10: 64,
} as const;

export const radius = {
  sm: 8,
  md: 12,
  lg: 16,
  xl: 22,
  full: 999,
} as const;

/**
 * A deliberately short type scale. Headings carry tight tracking; body text
 * stays at a comfortable 15px with generous line height for scanning listings.
 */
export const type = {
  display: { fontSize: 32, lineHeight: 38, fontWeight: '700', letterSpacing: -0.6 },
  title: { fontSize: 24, lineHeight: 30, fontWeight: '700', letterSpacing: -0.4 },
  heading: { fontSize: 18, lineHeight: 24, fontWeight: '600', letterSpacing: -0.2 },
  body: { fontSize: 15, lineHeight: 22, fontWeight: '400', letterSpacing: 0 },
  bodyStrong: { fontSize: 15, lineHeight: 22, fontWeight: '600', letterSpacing: 0 },
  caption: { fontSize: 13, lineHeight: 18, fontWeight: '400', letterSpacing: 0 },
  label: { fontSize: 11, lineHeight: 14, fontWeight: '700', letterSpacing: 0.6 },
} as const;

export type TypeVariant = keyof typeof type;

/** Motion: short and functional. Press feedback matters more than transitions. */
export const motion = {
  fast: 120,
  base: 180,
  slow: 260,
} as const;
