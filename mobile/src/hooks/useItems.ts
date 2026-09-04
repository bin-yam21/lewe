import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';

import { items, offers, ratings, type ItemQuery } from '@/api/endpoints';
import type { CreateItemPayload, ItemList } from '@/api/types';

const PER_PAGE = 20;

export const itemKeys = {
  all: ['items'] as const,
  list: (filters: ItemQuery) => ['items', 'list', filters] as const,
  mine: () => ['items', 'mine'] as const,
  detail: (id: string) => ['items', 'detail', id] as const,
};

/**
 * The feed. The API paginates with page/per_page and returns a total, so the
 * next page exists exactly while we have fetched fewer rows than that total.
 *
 * (Roadmap A1 moves this to cursor pagination — offset paging duplicates and
 * skips rows when items are created while someone is scrolling.)
 */
export function useItemsFeed(filters: { q?: string; category?: string }) {
  return useInfiniteQuery({
    queryKey: itemKeys.list(filters),
    initialPageParam: 1,
    queryFn: ({ pageParam }) =>
      items.list({ ...filters, page: pageParam, per_page: PER_PAGE }),
    getNextPageParam: (lastPage: ItemList, allPages) => {
      const loaded = allPages.reduce((n, p) => n + p.items.length, 0);
      return loaded < lastPage.total ? lastPage.page + 1 : undefined;
    },
  });
}

export function useMyItems() {
  return useQuery({
    queryKey: itemKeys.mine(),
    queryFn: () => items.mine({ per_page: 50 }),
  });
}

export function useItem(id: string | undefined) {
  return useQuery({
    queryKey: itemKeys.detail(id ?? ''),
    queryFn: () => items.get(id!),
    enabled: !!id,
  });
}

export function useUserRating(userId: string | undefined) {
  return useQuery({
    queryKey: ['rating', userId],
    queryFn: () => ratings.forUser(userId!),
    enabled: !!userId,
  });
}

/** Other listings in the same category — shown under an item's detail. */
export function useSimilarItems(id: string | undefined) {
  return useQuery({
    queryKey: ['items', 'similar', id],
    queryFn: () => items.similar(id!),
    enabled: !!id,
    // Similarity is category-based and the feed changes slowly; no need to
    // refetch this every time someone flicks back to a listing.
    staleTime: 5 * 60_000,
  });
}

/** The items a user can put on the table: listed and active, or held private. */
export function useOfferableItems() {
  return useQuery({
    queryKey: ['items', 'offerable'],
    queryFn: async () => {
      const [active, priv] = await Promise.all([
        items.mine({ status: 'active', per_page: 50 }),
        items.mine({ status: 'private', per_page: 50 }),
      ]);
      return [...active.items, ...priv.items];
    },
  });
}

export function useOfferTrade() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      targetItemId,
      offerItemId,
      message,
    }: {
      targetItemId: string;
      offerItemId: string;
      message?: string;
    }) => offers.create(targetItemId, offerItemId, message),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['matches'] });
      queryClient.invalidateQueries({ queryKey: itemKeys.all });
    },
  });
}

export function useCreateItem() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (payload: CreateItemPayload) => items.create(payload),
    onSuccess: () => {
      // The new listing belongs in both the public feed and "my items".
      queryClient.invalidateQueries({ queryKey: itemKeys.all });
    },
  });
}

export function useArchiveItem() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => items.archive(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: itemKeys.all });
    },
  });
}
