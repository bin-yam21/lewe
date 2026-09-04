/**
 * Design tokens for Lewe.
 *
 * Direction: warm, credible, and a little joyful. Strangers arrange to meet and
 * hand each other their possessions on this app, so it has to look like a place
 * run by people — not a dashboard.
 *
 * Three decisions carry most of that feeling:
 *
 *  - The ground is warm paper (#FBF9F6), never pure #FFF. Screen-white reads as
 *    clinical and is the single biggest tell of a generated-looking interface.
 *  - The primary is a confident green. Trade, reuse and "go" all live there, and
 *    it is a friendlier promise than corporate blue.
 *  - "Sun" amber is reserved for moments of delight — ratings, a successful
 *    trade, a new match. Never for chrome.
 *
 * Nothing in the app hardcodes a color. Screens read these through `useTheme()`,
 * which is what makes dark mode a token swap instead of a rewrite.
 */

export const palette = {
  light: {
    /** Warm paper, not screen-white. */
    bg: '#FBF9F6',
    bgSubtle: '#F4F0E9',
    surface: '#FFFFFF',
    surfaceAlt: '#F4F1EB',
    surfacePressed: '#EAE4DA',

    border: '#E9E2D8',
    borderStrong: '#D7CFC2',

    text: '#191614',
    textMuted: '#6E6761',
    textFaint: '#9C948B',
    /** Text and icons drawn on top of the accent color. */
    onAccent: '#FFFFFF',

    accent: '#0F9D63',
    accentPressed: '#0B7E4F',
    accentSubtle: '#E4F5EC',
    accentBorder: '#A9E1C6',

    /** Joy, warmth, reward. Ratings and celebratory moments only. */
    sun: '#E8890B',
    sunSubtle: '#FDF1DE',
    coral: '#F0603C',
    coralSubtle: '#FDE9E3',

    success: '#0F9D63',
    successSubtle: '#E4F5EC',
    warning: '#B06A08',
    warningSubtle: '#FDF1DE',
    danger: '#CE3B23',
    dangerSubtle: '#FCE7E2',

    overlay: 'rgba(25, 22, 20, 0.55)',
    /** Sits over photos so white text stays readable on a bright image. */
    scrim: 'rgba(15, 13, 12, 0.42)',
    skeleton: '#EDE7DE',
    shadow: '#4A3F33',
  },
  dark: {
    bg: '#121110',
    bgSubtle: '#181614',
    surface: '#1E1B19',
    surfaceAlt: '#272321',
    surfacePressed: '#312C29',

    border: '#2D2926',
    borderStrong: '#403935',

    text: '#F7F4F0',
    textMuted: '#A8A099',
    textFaint: '#7B746D',
    onAccent: '#06231A',

    accent: '#3ED598',
    accentPressed: '#2BBE84',
    accentSubtle: '#0F2A20',
    accentBorder: '#1F4C39',

    sun: '#F5B23C',
    sunSubtle: '#2A2011',
    coral: '#FF7C5C',
    coralSubtle: '#2E1913',

    success: '#3ED598',
    successSubtle: '#0F2A20',
    warning: '#F5B23C',
    warningSubtle: '#2A2011',
    danger: '#FF7C66',
    dangerSubtle: '#2E1614',

    overlay: 'rgba(0, 0, 0, 0.7)',
    scrim: 'rgba(0, 0, 0, 0.5)',
    skeleton: '#272321',
    shadow: '#000000',
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

/** Generous rounding — soft shapes read as approachable rather than technical. */
export const radius = {
  sm: 10,
  md: 14,
  lg: 20,
  xl: 28,
  full: 999,
} as const;

/**
 * A deliberately short type scale. Headings carry tight tracking; body text
 * stays at a comfortable 15px with generous line height for scanning listings.
 */
export const type = {
  display: { fontSize: 34, lineHeight: 40, fontWeight: '800', letterSpacing: -0.8 },
  title: { fontSize: 25, lineHeight: 31, fontWeight: '700', letterSpacing: -0.5 },
  heading: { fontSize: 18, lineHeight: 24, fontWeight: '700', letterSpacing: -0.2 },
  body: { fontSize: 15, lineHeight: 22, fontWeight: '400', letterSpacing: 0 },
  bodyStrong: { fontSize: 15, lineHeight: 22, fontWeight: '600', letterSpacing: 0 },
  caption: { fontSize: 13, lineHeight: 18, fontWeight: '400', letterSpacing: 0 },
  label: { fontSize: 11, lineHeight: 14, fontWeight: '700', letterSpacing: 0.6 },
} as const;

export type TypeVariant = keyof typeof type;

/** Motion: quick and springy. Press feedback matters more than transitions. */
export const motion = {
  fast: 120,
  base: 220,
  slow: 380,
  /** Shared spring config so everything that moves feels like one system. */
  spring: { damping: 16, stiffness: 220, mass: 0.6 },
} as const;

/**
 * One soft, warm shadow. Used sparingly — on photo cards and floating actions,
 * where lift genuinely helps, rather than on every surface.
 */
export const elevation = {
  card: {
    shadowOffset: { width: 0, height: 6 },
    shadowOpacity: 0.1,
    shadowRadius: 16,
    elevation: 3,
  },
  lifted: {
    shadowOffset: { width: 0, height: 10 },
    shadowOpacity: 0.16,
    shadowRadius: 24,
    elevation: 8,
  },
} as const;
