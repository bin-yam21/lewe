import { Ionicons } from '@expo/vector-icons';
import { useRouter } from 'expo-router';
import { useMemo, useState } from 'react';
import { ActivityIndicator, FlatList, ScrollView, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { ApiError } from '@/api/client';
import { CATEGORIES, labelForCategory, type Item } from '@/api/types';
import { Input } from '@/components/Input';
import { ItemCard } from '@/components/ItemCard';
import { Text } from '@/components/Text';
import { Chip, EmptyState, ErrorState, Skeleton } from '@/components/feedback';
import { Row, Stack } from '@/components/layout';
import { useDebounced } from '@/hooks/useDebounced';
import { useItemsFeed } from '@/hooks/useItems';
import { useAuth } from '@/hooks/useAuth';
import { useTheme } from '@/theme';

export default function Feed() {
  const theme = useTheme();
  const router = useRouter();
  const insets = useSafeAreaInsets();
  const { user } = useAuth();

  const [search, setSearch] = useState('');
  const [category, setCategory] = useState<string | undefined>();

  // Without this every keystroke would fire a request.
  const debouncedSearch = useDebounced(search, 350);

  const query = useItemsFeed(
    useMemo(() => ({ q: debouncedSearch || undefined, category }), [debouncedSearch, category]),
  );

  const items = useMemo<Item[]>(
    () => query.data?.pages.flatMap((page) => page.items) ?? [],
    [query.data],
  );

  const firstName = user?.full_name?.split(' ')[0] ?? 'there';

  return (
    <View style={{ flex: 1, backgroundColor: theme.colors.bgSubtle, paddingTop: insets.top }}>
      <FlatList
        data={items}
        keyExtractor={(item) => item.id}
        contentContainerStyle={{
          padding: theme.space[5],
          paddingBottom: theme.space[9],
          gap: theme.space[3],
        }}
        showsVerticalScrollIndicator={false}
        refreshing={query.isRefetching && !query.isFetchingNextPage}
        onRefresh={() => query.refetch()}
        onEndReachedThreshold={0.4}
        onEndReached={() => {
          if (query.hasNextPage && !query.isFetchingNextPage) query.fetchNextPage();
        }}
        ListHeaderComponent={
          <Stack gap={5} style={{ marginBottom: theme.space[2] }}>
            <Stack gap={1}>
              <Text variant="caption" color="textMuted">
                Hey {firstName}
              </Text>
              <Text variant="title">Find your next trade</Text>
            </Stack>

            <Input
              value={search}
              onChangeText={setSearch}
              placeholder="Search items…"
              autoCapitalize="none"
              autoCorrect={false}
              returnKeyType="search"
              containerStyle={{ marginBottom: 0 }}
            />

            <ScrollView
              horizontal
              showsHorizontalScrollIndicator={false}
              contentContainerStyle={{ gap: theme.space[2], paddingRight: theme.space[5] }}
              style={{ marginHorizontal: -theme.space[5], paddingHorizontal: theme.space[5] }}
            >
              <Chip label="All" selected={!category} onPress={() => setCategory(undefined)} />
              {CATEGORIES.map((c) => (
                <Chip
                  key={c}
                  label={labelForCategory(c)}
                  selected={category === c}
                  onPress={() => setCategory(category === c ? undefined : c)}
                />
              ))}
            </ScrollView>
          </Stack>
        }
        ListEmptyComponent={
          query.isLoading ? (
            <Stack gap={3}>
              {[0, 1, 2, 3].map((i) => (
                <Skeleton key={i} height={116} radius={theme.radius.lg} />
              ))}
            </Stack>
          ) : query.isError ? (
            <ErrorState
              message={
                query.error instanceof ApiError
                  ? query.error.message
                  : 'Could not load listings.'
              }
              onRetry={() => query.refetch()}
            />
          ) : search || category ? (
            <EmptyState
              icon="search-outline"
              title="Nothing matches that"
              message="Try a different search, or clear the category filter."
            />
          ) : (
            <EmptyState
              icon="cube-outline"
              title="No listings yet"
              message="Be the first — list something you would trade and see who bites."
            />
          )
        }
        renderItem={({ item }) => (
          <ItemCard item={item} onPress={() => router.push(`/item/${item.id}`)} />
        )}
        ListFooterComponent={
          query.isFetchingNextPage ? (
            <Row justify="center" style={{ paddingVertical: theme.space[5] }}>
              <ActivityIndicator color={theme.colors.accent} />
            </Row>
          ) : items.length > 0 && !query.hasNextPage ? (
            <Row justify="center" style={{ paddingVertical: theme.space[5] }}>
              <Row gap={2}>
                <Ionicons name="checkmark-circle-outline" size={14} color={theme.colors.textFaint} />
                <Text variant="caption" color="textFaint">
                  That is everything
                </Text>
              </Row>
            </Row>
          ) : null
        }
      />
    </View>
  );
}
