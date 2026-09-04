import { Ionicons } from '@expo/vector-icons';
import { Link, useRouter } from 'expo-router';
import { View } from 'react-native';

import { Button } from '@/components/Button';
import { Logo } from '@/components/Logo';
import { Text } from '@/components/Text';
import { Row, Screen, Stack } from '@/components/layout';
import { AppearFromBottom } from '@/components/motion';
import { useTheme } from '@/theme';

const POINTS = [
  {
    icon: 'pricetags-outline' as const,
    title: 'List what you have',
    body: 'Photograph it, name what you would take for it.',
  },
  {
    icon: 'swap-horizontal-outline' as const,
    title: 'Match both ways',
    body: 'Lewe only pairs you when the interest runs in both directions.',
  },
  {
    icon: 'shield-checkmark-outline' as const,
    title: 'Trade with confidence',
    body: 'Both sides confirm the exchange, then rate each other.',
  },
];

export default function Welcome() {
  const theme = useTheme();
  const router = useRouter();

  return (
    <Screen scroll contentStyle={{ flexGrow: 1, justifyContent: 'space-between' }}>
      <Stack gap={8}>
        <Stack gap={6} style={{ marginTop: theme.space[7] }}>
          <AppearFromBottom index={0}>
            <Logo size={44} />
          </AppearFromBottom>

          <AppearFromBottom index={1}>
            <Stack gap={3}>
              <Text variant="display">
                Trade what you{'\n'}no longer need
                <Text variant="display" style={{ color: theme.colors.accent }}>
                  .
                </Text>
              </Text>
              <Text variant="body" color="textMuted" style={{ maxWidth: 330 }}>
                No prices, no fees, no haggling over money. Just good things
                changing hands between people who both want the swap.
              </Text>
            </Stack>
          </AppearFromBottom>
        </Stack>

        <Stack gap={5}>
          {POINTS.map((point, i) => (
            <AppearFromBottom key={point.title} index={i + 2}>
            <Row gap={4} align="flex-start">
              <View
                style={{
                  width: 38,
                  height: 38,
                  borderRadius: theme.radius.md,
                  backgroundColor: theme.colors.accentSubtle,
                  alignItems: 'center',
                  justifyContent: 'center',
                }}
              >
                <Ionicons name={point.icon} size={19} color={theme.colors.accent} />
              </View>
              <Stack gap={1} style={{ flex: 1 }}>
                <Text variant="bodyStrong">{point.title}</Text>
                <Text variant="caption" color="textMuted">
                  {point.body}
                </Text>
                </Stack>
              </Row>
            </AppearFromBottom>
          ))}
        </Stack>
      </Stack>

      <Stack gap={4} style={{ marginTop: theme.space[9] }}>
        <Button
          title="Create an account"
          block
          size="lg"
          icon={<Ionicons name="arrow-forward" size={18} color={theme.colors.onAccent} />}
          onPress={() => router.push('/(auth)/register')}
        />
        <Row gap={2} justify="center">
          <Text variant="body" color="textMuted">
            Already have one?
          </Text>
          <Link href="/(auth)/login">
            <Text variant="bodyStrong" color="accent">
              Sign in
            </Text>
          </Link>
        </Row>
      </Stack>
    </Screen>
  );
}
