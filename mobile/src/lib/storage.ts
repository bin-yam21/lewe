import * as SecureStore from 'expo-secure-store';
import { Platform } from 'react-native';

/**
 * Token storage. Refresh tokens are long-lived credentials, so on device they
 * go to the Keychain/Keystore rather than AsyncStorage.
 *
 * expo-secure-store has no web implementation; on web we fall back to
 * localStorage, which is acceptable for development but is *not* where a
 * production web build should keep a refresh token.
 */

const ACCESS = 'lewe.access_token';
const REFRESH = 'lewe.refresh_token';

const isWeb = Platform.OS === 'web';

async function setItem(key: string, value: string | null): Promise<void> {
  if (isWeb) {
    try {
      if (value === null) globalThis.localStorage?.removeItem(key);
      else globalThis.localStorage?.setItem(key, value);
    } catch {
      // Private mode / storage disabled — the session simply won't persist.
    }
    return;
  }

  if (value === null) await SecureStore.deleteItemAsync(key);
  else await SecureStore.setItemAsync(key, value);
}

async function getItem(key: string): Promise<string | null> {
  if (isWeb) {
    try {
      return globalThis.localStorage?.getItem(key) ?? null;
    } catch {
      return null;
    }
  }
  return SecureStore.getItemAsync(key);
}

export const tokenStore = {
  get: async () => ({
    accessToken: await getItem(ACCESS),
    refreshToken: await getItem(REFRESH),
  }),

  save: async (accessToken: string, refreshToken: string) => {
    await Promise.all([setItem(ACCESS, accessToken), setItem(REFRESH, refreshToken)]);
  },

  clear: async () => {
    await Promise.all([setItem(ACCESS, null), setItem(REFRESH, null)]);
  },
};
