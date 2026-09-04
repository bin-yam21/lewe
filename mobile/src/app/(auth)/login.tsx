import { Ionicons } from '@expo/vector-icons';
import { Link, useRouter } from 'expo-router';
import { useState } from 'react';
import { Pressable, View } from 'react-native';

import { ApiError } from '@/api/client';
import { Button } from '@/components/Button';
import { Input } from '@/components/Input';
import { Text } from '@/components/Text';
import { Row, Screen, Stack } from '@/components/layout';
import { useAuth } from '@/hooks/useAuth';
import { useTheme } from '@/theme';

export default function Login() {
  const theme = useTheme();
  const router = useRouter();
  const { signIn } = useAuth();

  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [fields, setFields] = useState<Record<string, string>>({});

  const canSubmit = email.trim().length > 0 && password.length > 0 && !submitting;

  async function onSubmit() {
    if (!canSubmit) return;
    setSubmitting(true);
    setError(null);
    setFields({});

    try {
      await signIn(email, password);
      router.replace('/(tabs)');
    } catch (e) {
      if (e instanceof ApiError) {
        setError(e.message);
        setFields(e.fields ?? {});
      } else {
        setError('Something went wrong. Please try again.');
      }
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Screen scroll grow>
      <Stack gap={7}>
          <Pressable onPress={() => router.back()} hitSlop={12} style={{ alignSelf: 'flex-start' }}>
            <Ionicons name="arrow-back" size={24} color={theme.colors.text} />
          </Pressable>

          <Stack gap={2}>
            <Text variant="title">Welcome back</Text>
            <Text variant="body" color="textMuted">
              Sign in to pick up where you left off.
            </Text>
          </Stack>

          {error ? (
            <View
              style={{
                backgroundColor: theme.colors.dangerSubtle,
                borderRadius: theme.radius.md,
                padding: theme.space[3],
              }}
            >
              <Row gap={2} align="flex-start">
                <Ionicons name="alert-circle" size={18} color={theme.colors.danger} />
                <Text variant="caption" color="danger" style={{ flex: 1 }}>
                  {error}
                </Text>
              </Row>
            </View>
          ) : null}

          <Stack gap={5}>
            <Input
              label="Email"
              value={email}
              onChangeText={setEmail}
              error={fields.email}
              placeholder="you@example.com"
              autoCapitalize="none"
              autoComplete="email"
              keyboardType="email-address"
              returnKeyType="next"
            />

            <Input
              label="Password"
              value={password}
              onChangeText={setPassword}
              error={fields.password}
              placeholder="Your password"
              secureTextEntry={!showPassword}
              autoCapitalize="none"
              autoComplete="current-password"
              returnKeyType="go"
              onSubmitEditing={onSubmit}
            />

            <Pressable onPress={() => setShowPassword((v) => !v)} hitSlop={8}>
              <Text variant="caption" color="accent">
                {showPassword ? 'Hide password' : 'Show password'}
              </Text>
            </Pressable>
          </Stack>

          <Button
            title="Sign in"
            block
            size="lg"
            loading={submitting}
            disabled={!canSubmit}
            onPress={onSubmit}
          />

          <Row gap={2} justify="center">
            <Text variant="body" color="textMuted">
              New here?
            </Text>
            <Link href="/(auth)/register" replace>
              <Text variant="bodyStrong" color="accent">
                Create an account
              </Text>
            </Link>
        </Row>
      </Stack>
    </Screen>
  );
}
