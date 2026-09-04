import { Ionicons } from '@expo/vector-icons';
import { useRouter } from 'expo-router';
import { Alert, ScrollView, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { API_BASE_URL } from '@/api/config';
import { labelForCategory } from '@/api/types';
import { Button } from '@/components/Button';
import { Text } from '@/components/Text';
import { Avatar, Badge, EmptyState, Skeleton } from '@/components/feedback';
import { Card, Divider, Row, Stack } from '@/components/layout';
import { useAuth } from '@/hooks/useAuth';
import { useMyItems, useUserRating } from '@/hooks/useItems';
import { useTheme } from '@/theme';

export default function Profile() {
  const theme = useTheme();
  const router = useRouter();
  const insets = useSafeAreaInsets();
  const { user, signOut } = useAuth();

  const { data: mine, isLoading } = useMyItems();
  const { data: rating } = useUserRating(user?.id);

  const items = mine?.items ?? [];

  function confirmSignOut() {
    Alert.alert('Sign out', 'You will need to sign in again to trade.', [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Sign out',
        style: 'destructive',
        onPress: async () => {
          await signOut();
          router.replace('/(auth)/welcome');
        },
      },
    ]);
  }

  return (
    <ScrollView
      style={{ flex: 1, backgroundColor: theme.colors.bgSubtle }}
      contentContainerStyle={{
        padding: theme.space[5],
        paddingTop: insets.top + theme.space[4],
        paddingBottom: theme.space[9],
        gap: theme.space[6],
      }}
      showsVerticalScrollIndicator={false}
    >
      <Row gap={4}>
        <Avatar name={user?.full_name ?? '?'} size={56} />
        <Stack gap={1} style={{ flex: 1 }}>
          <Text variant="heading">{user?.full_name}</Text>
          <Text variant="caption" color="textMuted">
            {user?.email}
          </Text>
          <Row gap={1}>
            <Ionicons name="star" size={12} color={theme.colors.warning} />
            <Text variant="caption" color="textMuted">
              {rating && rating.count > 0
                ? `${rating.average.toFixed(1)} · ${rating.count} rating${rating.count === 1 ? '' : 's'}`
                : 'No ratings yet'}
            </Text>
          </Row>
        </Stack>
      </Row>

      <Divider />

      <Stack gap={4}>
        <Row justify="space-between" align="center">
          <Text variant="heading">Your listings</Text>
          {items.length > 0 ? (
            <Text variant="caption" color="textFaint">
              {items.length}
            </Text>
          ) : null}
        </Row>

        {isLoading ? (
          <Stack gap={3}>
            <Skeleton height={72} radius={theme.radius.lg} />
            <Skeleton height={72} radius={theme.radius.lg} />
          </Stack>
        ) : items.length === 0 ? (
          <EmptyState
            icon="pricetags-outline"
            title="Nothing listed yet"
            message="List an item and Lewe starts looking for mutual trades."
            action={
              <Button title="List an item" onPress={() => router.push('/(tabs)/create')} />
            }
          />
        ) : (
          <Stack gap={3}>
            {items.map((item) => (
              <Card key={item.id} onPress={() => router.push(`/item/${item.id}`)}>
                <Row gap={3} justify="space-between">
                  <Stack gap={2} style={{ flex: 1 }}>
                    <Text variant="bodyStrong" numberOfLines={1}>
                      {item.title}
                    </Text>
                    <Row gap={2} wrap>
                      <Badge
                        label={item.status}
                        tone={
                          item.status === 'active'
                            ? 'success'
                            : item.status === 'exchanged'
                              ? 'accent'
                              : 'neutral'
                        }
                      />
                      <Text variant="caption" color="textFaint">
                        {labelForCategory(item.category)}
                      </Text>
                    </Row>
                  </Stack>
                  <Ionicons name="chevron-forward" size={18} color={theme.colors.textFaint} />
                </Row>
              </Card>
            ))}
          </Stack>
        )}
      </Stack>

      <Divider />

      <Stack gap={3}>
        <Text variant="label" color="textFaint" uppercase>
          Connected to
        </Text>
        <Text variant="caption" color="textMuted">
          {API_BASE_URL}
        </Text>
      </Stack>

      <Button title="Sign out" variant="danger" block onPress={confirmSignOut} />

      <View style={{ height: theme.space[5] }} />
    </ScrollView>
  );
}
