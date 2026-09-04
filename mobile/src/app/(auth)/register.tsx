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

export default function Register() {
  const theme = useTheme();
  const router = useRouter();
  const { signUp } = useAuth();

  const [fullName, setFullName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [fields, setFields] = useState<Record<string, string>>({});

  // Mirrors the API's own rule (validator.ValidateMinLength(password, 8)) so the
  // user finds out before a round trip.
  const passwordTooShort = password.length > 0 && password.length < 8;
  const canSubmit =
    fullName.trim().length > 0 &&
    email.trim().length > 0 &&
    password.length >= 8 &&
    !submitting;

  async function onSubmit() {
    if (!canSubmit) return;
    setSubmitting(true);
    setError(null);
    setFields({});

    try {
      await signUp(email, password, fullName);
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
            <Text variant="title">Create your account</Text>
            <Text variant="body" color="textMuted">
              It takes about a minute. You can list your first item right after.
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
              label="Full name"
              value={fullName}
              onChangeText={setFullName}
              error={fields.full_name}
              placeholder="Alex Mercer"
              autoCapitalize="words"
              autoComplete="name"
            />

            <Input
              label="Email"
              value={email}
              onChangeText={setEmail}
              error={fields.email}
              placeholder="you@example.com"
              autoCapitalize="none"
              autoComplete="email"
              keyboardType="email-address"
            />

            <Input
              label="Password"
              value={password}
              onChangeText={setPassword}
              error={fields.password ?? (passwordTooShort ? 'Must be at least 8 characters' : undefined)}
              hint="At least 8 characters"
              placeholder="Choose a password"
              secureTextEntry={!showPassword}
              autoCapitalize="none"
              autoComplete="new-password"
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
            title="Create account"
            block
            size="lg"
            loading={submitting}
            disabled={!canSubmit}
            onPress={onSubmit}
          />

          <Row gap={2} justify="center">
            <Text variant="body" color="textMuted">
              Already have an account?
            </Text>
            <Link href="/(auth)/login" replace>
              <Text variant="bodyStrong" color="accent">
                Sign in
              </Text>
            </Link>
        </Row>
      </Stack>
    </Screen>
  );
}
