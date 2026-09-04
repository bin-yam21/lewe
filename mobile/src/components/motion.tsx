import * as Haptics from 'expo-haptics';
import { type ReactNode } from 'react';
import { Platform, Pressable, type ViewStyle } from 'react-native';
import Animated, {
  FadeInDown,
  useAnimatedStyle,
  useSharedValue,
  withSpring,
  withTiming,
} from 'react-native-reanimated';

import { motion } from '@/theme';

const AnimatedPressable = Animated.createAnimatedComponent(Pressable);

/**
 * A pressable that dips slightly under the finger.
 *
 * This is the app's most-felt animation. A spring on press is the difference
 * between a screen that reacts to you and one that merely navigates, and it
 * costs nothing in perceived speed because it runs on the UI thread.
 */
export function PressableScale({
  children,
  onPress,
  style,
  scaleTo = 0.97,
  haptic = true,
  disabled,
}: {
  children: ReactNode;
  onPress?: () => void;
  style?: ViewStyle | ViewStyle[];
  scaleTo?: number;
  haptic?: boolean;
  disabled?: boolean;
}) {
  const pressed = useSharedValue(0);

  const animatedStyle = useAnimatedStyle(() => ({
    transform: [{ scale: withSpring(1 - pressed.value * (1 - scaleTo), motion.spring) }],
    opacity: withTiming(1 - pressed.value * 0.06, { duration: motion.fast }),
  }));

  return (
    <AnimatedPressable
      disabled={disabled}
      onPressIn={() => {
        pressed.value = 1;
      }}
      onPressOut={() => {
        pressed.value = 0;
      }}
      onPress={() => {
        if (haptic && Platform.OS !== 'web') {
          Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light).catch(() => {});
        }
        onPress?.();
      }}
      style={[style, animatedStyle]}
    >
      {children}
    </AnimatedPressable>
  );
}

/**
 * Staggered entrance for list items and stacked sections.
 *
 * The delay is capped: past a handful of items the stagger stops reading as
 * choreography and starts reading as lag.
 */
export function AppearFromBottom({
  children,
  index = 0,
  style,
}: {
  children: ReactNode;
  index?: number;
  style?: ViewStyle;
}) {
  return (
    <Animated.View
      entering={FadeInDown.duration(motion.slow)
        .delay(Math.min(index, 8) * 55)
        .springify()
        .damping(18)}
      style={style}
    >
      {children}
    </Animated.View>
  );
}

export { Animated };
