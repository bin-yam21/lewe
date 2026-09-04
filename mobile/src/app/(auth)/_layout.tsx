import { Redirect, Stack } from 'expo-router';

import { useAuth } from '@/hooks/useAuth';
import { useTheme } from '@/theme';

export default function AuthLayout() {
  const { user } = useAuth();
  const theme = useTheme();

  // Signing in mid-stack should never leave an auth screen behind you.
  if (user) return <Redirect href="/(tabs)" />;

  return (
    <Stack
      screenOptions={{
        headerShown: false,
        contentStyle: { backgroundColor: theme.colors.bg },
      }}
    />
  );
}
