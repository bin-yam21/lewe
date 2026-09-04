import { Ionicons } from '@expo/vector-icons';
import { Image } from 'expo-image';
import { useRouter } from 'expo-router';
import { useState } from 'react';
import { ActivityIndicator, Modal, Pressable, ScrollView, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { ApiError } from '@/api/client';
import { resolveImageUrl } from '@/api/config';
import { labelForCategory, type Item } from '@/api/types';
import { useOfferTrade, useOfferableItems } from '@/hooks/useItems';
import { useTheme } from '@/theme';

import { Button } from './Button';
import { Input } from './Input';
import { Text } from './Text';
import { Badge, EmptyState } from './feedback';
import { Divider, Row, Stack } from './layout';
import { PressableScale } from './motion';

/**
 * Offer a trade straight to an owner.
 *
 * The matcher only pairs people whose wants line up in both directions, which
 * is a slow way to trade when you are looking right at something you want. This
 * is the direct path: pick something of yours and hand it over as a proposal.
 *
 * "Something not listed" creates a private item — a real item row that never
 * appears in the browse feed and exists only inside this offer.
 */
export function OfferSheet({
  visible,
  onClose,
  target,
  onOffered,
}: {
  visible: boolean;
  onClose: () => void;
  target: Item;
  onOffered: (matchId: string) => void;
}) {
  const theme = useTheme();
  const insets = useSafeAreaInsets();
  const router = useRouter();

  const { data: myItems, isLoading } = useOfferableItems();
  const offer = useOfferTrade();

  const [selected, setSelected] = useState<string | null>(null);
  const [message, setMessage] = useState('');
  const [error, setError] = useState<string | null>(null);

  function close() {
    setSelected(null);
    setMessage('');
    setError(null);
    onClose();
  }

  async function send() {
    if (!selected) return;
    setError(null);

    try {
      const match = await offer.mutateAsync({
        targetItemId: target.id,
        offerItemId: selected,
        message,
      });
      close();
      onOffered(match.id);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Could not send your offer.');
    }
  }

  return (
    <Modal visible={visible} animationType="slide" presentationStyle="pageSheet" onRequestClose={close}>
      <View style={{ flex: 1, backgroundColor: theme.colors.bg }}>
        {/* Header */}
        <Row
          justify="space-between"
          align="center"
          style={{
            paddingHorizontal: theme.space[5],
            paddingTop: theme.space[5],
            paddingBottom: theme.space[4],
          }}
        >
          <Stack gap={1} style={{ flex: 1 }}>
            <Text variant="heading">Offer a trade</Text>
            <Text variant="caption" color="textMuted" numberOfLines={1}>
              for {target.title}
            </Text>
          </Stack>
          <Pressable onPress={close} hitSlop={12}>
            <Ionicons name="close" size={24} color={theme.colors.textMuted} />
          </Pressable>
        </Row>

        <Divider />

        <ScrollView
          contentContainerStyle={{
            padding: theme.space[5],
            paddingBottom: theme.space[9],
            gap: theme.space[5],
          }}
          keyboardShouldPersistTaps="handled"
        >
          <Text variant="body" color="textMuted">
            Choose what you would give. They can accept or decline — nothing is
            agreed until you both confirm.
          </Text>

          {/* The "without listing it" path — creates a private item and sends
              it as this offer in one go. */}
          <PressableScale
            scaleTo={0.98}
            onPress={() => {
              close();
              router.push(`/(tabs)/create?private=1&offerFor=${target.id}`);
            }}
          >
            <Row
              gap={3}
              style={{
                padding: theme.space[4],
                borderRadius: theme.radius.lg,
                borderWidth: 2,
                borderStyle: 'dashed',
                borderColor: theme.colors.accentBorder,
                backgroundColor: theme.colors.accentSubtle,
              }}
            >
              <Ionicons name="add-circle" size={22} color={theme.colors.accent} />
              <Stack gap={1} style={{ flex: 1 }}>
                <Text variant="bodyStrong" color="accent">
                  Offer something not listed
                </Text>
                <Text variant="caption" color="textMuted">
                  Add it privately — it stays off the browse feed.
                </Text>
              </Stack>
            </Row>
          </PressableScale>

          {isLoading ? (
            <Row justify="center" style={{ paddingVertical: theme.space[7] }}>
              <ActivityIndicator color={theme.colors.accent} />
            </Row>
          ) : !myItems || myItems.length === 0 ? (
            <EmptyState
              icon="cube-outline"
              title="You have nothing to offer yet"
              message="Add an item and it becomes available to trade. It does not have to be listed publicly."
            />
          ) : (
            <Stack gap={3}>
              {myItems.map((item) => {
                const isSelected = selected === item.id;
                const cover = resolveImageUrl(item.images?.[0]);

                return (
                  <PressableScale
                    key={item.id}
                    scaleTo={0.98}
                    onPress={() => setSelected(isSelected ? null : item.id)}
                  >
                    <Row
                      gap={3}
                      style={{
                        padding: theme.space[3],
                        borderRadius: theme.radius.lg,
                        borderWidth: isSelected ? 2 : 1,
                        borderColor: isSelected ? theme.colors.accent : theme.colors.border,
                        backgroundColor: isSelected
                          ? theme.colors.accentSubtle
                          : theme.colors.surface,
                      }}
                    >
                      <View
                        style={{
                          width: 60,
                          height: 60,
                          borderRadius: theme.radius.md,
                          overflow: 'hidden',
                          backgroundColor: theme.colors.surfaceAlt,
                          alignItems: 'center',
                          justifyContent: 'center',
                        }}
                      >
                        {cover ? (
                          <Image
                            source={{ uri: cover }}
                            style={{ width: '100%', height: '100%' }}
                            contentFit="cover"
                          />
                        ) : (
                          <Ionicons name="image-outline" size={18} color={theme.colors.textFaint} />
                        )}
                      </View>

                      <Stack gap={1} style={{ flex: 1 }}>
                        <Text variant="bodyStrong" numberOfLines={1}>
                          {item.title}
                        </Text>
                        <Row gap={2}>
                          <Text variant="caption" color="textMuted">
                            {labelForCategory(item.category)}
                          </Text>
                          {item.status === 'private' ? <Badge label="Private" /> : null}
                        </Row>
                      </Stack>

                      <Ionicons
                        name={isSelected ? 'checkmark-circle' : 'ellipse-outline'}
                        size={22}
                        color={isSelected ? theme.colors.accent : theme.colors.borderStrong}
                      />
                    </Row>
                  </PressableScale>
                );
              })}
            </Stack>
          )}

          <Input
            label="Add a note (optional)"
            value={message}
            onChangeText={setMessage}
            placeholder="Happy to meet somewhere central this week"
            multiline
          />

          {error ? (
            <Row
              gap={2}
              align="flex-start"
              style={{
                backgroundColor: theme.colors.dangerSubtle,
                padding: theme.space[3],
                borderRadius: theme.radius.md,
              }}
            >
              <Ionicons name="alert-circle" size={18} color={theme.colors.danger} />
              <Text variant="caption" color="danger" style={{ flex: 1 }}>
                {error}
              </Text>
            </Row>
          ) : null}
        </ScrollView>

        {/* Footer action stays reachable above the home indicator. */}
        <View
          style={{
            padding: theme.space[5],
            paddingBottom: theme.space[5] + insets.bottom,
            borderTopWidth: 1,
            borderTopColor: theme.colors.border,
            backgroundColor: theme.colors.bg,
          }}
        >
          <Button
            title="Send offer"
            block
            size="lg"
            disabled={!selected}
            loading={offer.isPending}
            onPress={send}
          />
        </View>
      </View>
    </Modal>
  );
}
