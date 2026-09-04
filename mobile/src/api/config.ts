import Constants from 'expo-constants';

/**
 * Port the Go API listens on. Not 8080 (taken by an unrelated service on the
 * dev machine) and not 8081 (Metro's own dev server).
 */
const API_PORT = 8090;

/**
 * Resolving the API base URL is the one piece of config that bites everyone
 * running on a physical device: `localhost` on the phone means the phone.
 *
 * Order of preference:
 *   1. EXPO_PUBLIC_API_URL, if set — the escape hatch for staging/prod.
 *   2. The LAN IP of whatever machine is serving Metro. When you run
 *      `npx expo start`, the packager URL already contains this machine's
 *      address, so a device on the same Wi-Fi can reach the API without
 *      anyone hardcoding an IP that changes every time the router feels like it.
 *   3. localhost — correct for web and simulators.
 */
function resolveBaseUrl(): string {
  const explicit = process.env.EXPO_PUBLIC_API_URL;
  if (explicit) return explicit.replace(/\/+$/, '');

  const hostUri =
    Constants.expoConfig?.hostUri ??
    // `debuggerHost` exists in Expo Go but is not always in the public types.
    ((Constants.expoGoConfig as { debuggerHost?: string } | undefined)?.debuggerHost);

  const host = hostUri?.split(':')[0];
  if (host) return `http://${host}:${API_PORT}`;

  return `http://localhost:${API_PORT}`;
}

export const API_BASE_URL = resolveBaseUrl();
export const API_V1 = `${API_BASE_URL}/api/v1`;
