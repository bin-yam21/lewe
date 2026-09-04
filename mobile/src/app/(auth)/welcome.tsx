import { Ionicons } from '@expo/vector-icons';
import { Link, useRouter } from 'expo-router';
import { View } from 'react-native';

import { Button } from '@/components/Button';
import { Text } from '@/components/Text';
import { Row, Screen, Stack } from '@/components/layout';
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
        <Stack gap={4} style={{ marginTop: theme.space[8] }}>
          <View
            style={{
              width: 52,
              height: 52,
              borderRadius: theme.radius.lg,
              backgroundColor: theme.colors.accent,
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <Ionicons name="swap-horizontal" size={28} color={theme.colors.onAccent} />
          </View>

          <Stack gap={3}>
            <Text variant="display">Trade what you{'\n'}no longer need.</Text>
            <Text variant="body" color="textMuted" style={{ maxWidth: 320 }}>
              Lewe is a barter marketplace. No prices, no listings fees — just
              things swapping hands between people who both want the trade.
            </Text>
          </Stack>
        </Stack>

        <Stack gap={5}>
          {POINTS.map((point) => (
            <Row key={point.title} gap={4} align="flex-start">
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
          ))}
        </Stack>
      </Stack>

      <Stack gap={4} style={{ marginTop: theme.space[9] }}>
        <Button title="Create an account" block size="lg" onPress={() => router.push('/(auth)/register')} />
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
