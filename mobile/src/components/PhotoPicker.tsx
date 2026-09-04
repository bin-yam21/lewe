import { Ionicons } from '@expo/vector-icons';
import * as Haptics from 'expo-haptics';
import { Image } from 'expo-image';
import * as ImagePicker from 'expo-image-picker';
import { useState } from 'react';
import { ActivityIndicator, Alert, Platform, Pressable, ScrollView, View } from 'react-native';

import { resolveImageUrl } from '@/api/config';
import { uploads } from '@/api/endpoints';
import { useTheme } from '@/theme';

import { Text } from './Text';
import { Row, Stack } from './layout';
import { PressableScale } from './motion';

/**
 * The photo step of a listing.
 *
 * A traded object is sold by its picture — it is the only evidence the other
 * person has that the thing is real and in the condition claimed. So this is
 * the first thing asked for, it gets the most space, and the first photo is
 * explicitly labelled as the cover.
 *
 * Uploads happen as soon as a photo is chosen rather than on submit: by the
 * time someone reaches the review step the images are already on the server,
 * and publishing stays instant.
 */
export function PhotoPicker({
  value,
  onChange,
  max = 6,
}: {
  /** Host-relative paths already stored on the server. */
  value: string[];
  onChange: (next: string[]) => void;
  max?: number;
}) {
  const theme = useTheme();
  const [uploading, setUploading] = useState(0);

  const atLimit = value.length + uploading >= max;

  async function pick(from: 'library' | 'camera') {
    try {
      if (from === 'camera') {
        const perm = await ImagePicker.requestCameraPermissionsAsync();
        if (!perm.granted) {
          Alert.alert('Camera access needed', 'Allow camera access to photograph your item.');
          return;
        }
      }

      const options: ImagePicker.ImagePickerOptions = {
        mediaTypes: ['images'],
        quality: 0.7,
        // Downscaling here keeps uploads fast on the kind of connection people
        // actually list things on, and well under the API's 8MB cap.
        allowsEditing: from === 'camera',
        aspect: [4, 3],
      };

      const result =
        from === 'camera'
          ? await ImagePicker.launchCameraAsync(options)
          : await ImagePicker.launchImageLibraryAsync({
              ...options,
              allowsMultipleSelection: true,
              selectionLimit: max - value.length,
            });

      if (result.canceled || !result.assets?.length) return;

      const assets = result.assets.slice(0, max - value.length);
      setUploading((n) => n + assets.length);

      if (Platform.OS !== 'web') {
        Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Medium).catch(() => {});
      }

      const urls: string[] = [];
      for (const asset of assets) {
        try {
          urls.push(await uploads.image(asset.uri, asset.mimeType ?? 'image/jpeg'));
        } catch {
          Alert.alert('Upload failed', 'That photo could not be uploaded. Please try again.');
        } finally {
          setUploading((n) => Math.max(0, n - 1));
        }
      }

      if (urls.length) onChange([...value, ...urls]);
    } catch {
      setUploading(0);
      Alert.alert('Something went wrong', 'Could not open your photos.');
    }
  }

  function remove(url: string) {
    if (Platform.OS !== 'web') {
      Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light).catch(() => {});
    }
    onChange(value.filter((u) => u !== url));
  }

  function makeCover(url: string) {
    if (value[0] === url) return;
    if (Platform.OS !== 'web') {
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success).catch(() => {});
    }
    onChange([url, ...value.filter((u) => u !== url)]);
  }

  return (
    <Stack gap={4}>
      {value.length === 0 && uploading === 0 ? (
        // Empty state doubles as the primary call to action.
        <PressableScale onPress={() => pick('library')}>
          <View
            style={{
              borderRadius: theme.radius.lg,
              borderWidth: 2,
              borderStyle: 'dashed',
              borderColor: theme.colors.accentBorder,
              backgroundColor: theme.colors.accentSubtle,
              paddingVertical: theme.space[8],
              alignItems: 'center',
              gap: theme.space[3],
            }}
          >
            <View
              style={{
                width: 58,
                height: 58,
                borderRadius: theme.radius.full,
                backgroundColor: theme.colors.accent,
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <Ionicons name="camera" size={27} color={theme.colors.onAccent} />
            </View>
            <Stack gap={1}>
              <Text variant="bodyStrong" center>
                Add photos of your item
              </Text>
              <Text variant="caption" color="textMuted" center style={{ maxWidth: 260 }}>
                Good photos are the whole trade. Show it from a few angles, in daylight.
              </Text>
            </Stack>
          </View>
        </PressableScale>
      ) : (
        <ScrollView
          horizontal
          showsHorizontalScrollIndicator={false}
          contentContainerStyle={{ gap: theme.space[3] }}
        >
          {value.map((url, index) => (
            <View key={url}>
              <PressableScale onPress={() => makeCover(url)} scaleTo={0.95}>
                <View
                  style={{
                    width: 132,
                    height: 168,
                    borderRadius: theme.radius.lg,
                    overflow: 'hidden',
                    backgroundColor: theme.colors.surfaceAlt,
                    borderWidth: index === 0 ? 2 : 1,
                    borderColor: index === 0 ? theme.colors.accent : theme.colors.border,
                  }}
                >
                  <Image
                    source={{ uri: resolveImageUrl(url) }}
                    style={{ width: '100%', height: '100%' }}
                    contentFit="cover"
                    transition={180}
                  />
                  {index === 0 ? (
                    <View
                      style={{
                        position: 'absolute',
                        left: theme.space[2],
                        bottom: theme.space[2],
                        backgroundColor: theme.colors.accent,
                        paddingHorizontal: theme.space[2],
                        paddingVertical: 3,
                        borderRadius: theme.radius.sm,
                      }}
                    >
                      <Text variant="label" uppercase style={{ color: theme.colors.onAccent }}>
                        Cover
                      </Text>
                    </View>
                  ) : null}
                </View>
              </PressableScale>

              <Pressable
                onPress={() => remove(url)}
                hitSlop={10}
                style={{
                  position: 'absolute',
                  top: -6,
                  right: -6,
                  width: 26,
                  height: 26,
                  borderRadius: theme.radius.full,
                  backgroundColor: theme.colors.text,
                  alignItems: 'center',
                  justifyContent: 'center',
                }}
              >
                <Ionicons name="close" size={15} color={theme.colors.bg} />
              </Pressable>
            </View>
          ))}

          {Array.from({ length: uploading }).map((_, i) => (
            <View
              key={`uploading-${i}`}
              style={{
                width: 132,
                height: 168,
                borderRadius: theme.radius.lg,
                backgroundColor: theme.colors.surfaceAlt,
                borderWidth: 1,
                borderColor: theme.colors.border,
                alignItems: 'center',
                justifyContent: 'center',
                gap: theme.space[2],
              }}
            >
              <ActivityIndicator color={theme.colors.accent} />
              <Text variant="caption" color="textFaint">
                Uploading
              </Text>
            </View>
          ))}

          {!atLimit ? (
            <PressableScale onPress={() => pick('library')} scaleTo={0.95}>
              <View
                style={{
                  width: 132,
                  height: 168,
                  borderRadius: theme.radius.lg,
                  borderWidth: 2,
                  borderStyle: 'dashed',
                  borderColor: theme.colors.borderStrong,
                  alignItems: 'center',
                  justifyContent: 'center',
                  gap: theme.space[2],
                }}
              >
                <Ionicons name="add" size={26} color={theme.colors.textMuted} />
                <Text variant="caption" color="textMuted">
                  Add more
                </Text>
              </View>
            </PressableScale>
          ) : null}
        </ScrollView>
      )}

      <Row gap={3}>
        <PressableScale onPress={() => pick('camera')} disabled={atLimit}>
          <Row
            gap={2}
            style={{
              paddingHorizontal: theme.space[4],
              paddingVertical: theme.space[3],
              borderRadius: theme.radius.md,
              backgroundColor: theme.colors.surfaceAlt,
              borderWidth: 1,
              borderColor: theme.colors.border,
              opacity: atLimit ? 0.5 : 1,
            }}
          >
            <Ionicons name="camera-outline" size={17} color={theme.colors.text} />
            <Text variant="caption" style={{ fontWeight: '600' }}>
              Take photo
            </Text>
          </Row>
        </PressableScale>

        <PressableScale onPress={() => pick('library')} disabled={atLimit}>
          <Row
            gap={2}
            style={{
              paddingHorizontal: theme.space[4],
              paddingVertical: theme.space[3],
              borderRadius: theme.radius.md,
              backgroundColor: theme.colors.surfaceAlt,
              borderWidth: 1,
              borderColor: theme.colors.border,
              opacity: atLimit ? 0.5 : 1,
            }}
          >
            <Ionicons name="images-outline" size={17} color={theme.colors.text} />
            <Text variant="caption" style={{ fontWeight: '600' }}>
              Choose photos
            </Text>
          </Row>
        </PressableScale>
      </Row>

      {value.length > 1 ? (
        <Text variant="caption" color="textFaint">
          Tap a photo to make it the cover · {value.length} of {max}
        </Text>
      ) : null}
    </Stack>
  );
}
