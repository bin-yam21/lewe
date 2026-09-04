import { Redirect } from 'expo-router';
import { ActivityIndicator, View } from 'react-native';

import { useAuth } from '@/hooks/useAuth';
import { useTheme } from '@/theme';

/**
 * The session gate. Every cold launch lands here, waits for the stored token to
 * be validated, then goes to the feed or the welcome screen. Keeping this in one
 * place is what stops the "flash of the wrong screen" on startup.
 */
export default function Index() {
  const { user, restoring } = useAuth();
  const theme = useTheme();

  if (restoring) {
    return (
      <View
        style={{
          flex: 1,
          alignItems: 'center',
          justifyContent: 'center',
          backgroundColor: theme.colors.bg,
        }}
      >
        <ActivityIndicator color={theme.colors.accent} />
      </View>
    );
  }

  return <Redirect href={user ? '/(tabs)' : '/(auth)/welcome'} />;
}
