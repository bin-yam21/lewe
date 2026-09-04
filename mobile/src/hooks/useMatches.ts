import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { matches } from '@/api/endpoints';
import type { Match } from '@/api/types';

export const matchKeys = {
  all: ['matches'] as const,
  mine: () => ['matches', 'mine'] as const,
};

export function useMyMatches() {
  return useQuery({
    queryKey: matchKeys.mine(),
    queryFn: () => matches.mine(),
    // Offers arrive from other people, so this view goes stale on its own.
    refetchOnWindowFocus: true,
    staleTime: 15_000,
  });
}

function useMatchMutation<TArgs>(fn: (args: TArgs) => Promise<Match>) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: fn,
    onSuccess: () => {
      // A transition moves item statuses too, so both trees are invalidated.
      queryClient.invalidateQueries({ queryKey: matchKeys.all });
      queryClient.invalidateQueries({ queryKey: ['items'] });
    },
  });
}

export function useRespondToMatch() {
  return useMatchMutation(({ id, accept }: { id: string; accept: boolean }) =>
    matches.respond(id, accept),
  );
}

export function useCompleteMatch() {
  return useMatchMutation(({ id }: { id: string }) => matches.complete(id));
}

export function useCancelMatch() {
  return useMatchMutation(({ id }: { id: string }) => matches.cancel(id));
}

/**
 * Which side of a match the viewer is on, and therefore what they can do next.
 *
 * The API models the trade as item_a/item_b rather than proposer/recipient, so
 * every screen would otherwise repeat this reasoning.
 */
export function describeMatch(match: Match, userId: string | undefined) {
  const isA = match.item_a.user_id === userId;
  const mine = isA ? match.item_a : match.item_b;
  const theirs = isA ? match.item_b : match.item_a;

  const iAccepted =
    match.status === (isA ? 'accepted_a' : 'accepted_b') ||
    match.status === 'confirmed' ||
    match.status === 'completed';
  const theyAccepted =
    match.status === (isA ? 'accepted_b' : 'accepted_a') ||
    match.status === 'confirmed' ||
    match.status === 'completed';

  // On a direct offer item_a is always the offered item, so side A proposed it.
  const iProposed = match.origin === 'direct' && isA;

  const canRespond =
    match.status === 'pending' ||
    (match.status === 'accepted_a' && !isA) ||
    (match.status === 'accepted_b' && isA);

  const iConfirmed = isA ? match.confirmed_a : match.confirmed_b;

  let headline: string;
  switch (match.status) {
    case 'pending':
      headline = iProposed ? 'Waiting for their answer' : 'They want to trade with you';
      break;
    case 'accepted_a':
    case 'accepted_b':
      headline = iAccepted ? 'Waiting for their answer' : 'Your turn to answer';
      break;
    case 'confirmed':
      headline = iConfirmed ? 'Waiting for them to confirm' : 'Confirm once you have swapped';
      break;
    case 'completed':
      headline = 'Trade complete';
      break;
    case 'cancelled':
      headline = 'Cancelled';
      break;
  }

  return {
    mine,
    theirs,
    iProposed,
    iAccepted,
    theyAccepted,
    canRespond,
    iConfirmed,
    canComplete: match.status === 'confirmed' && !iConfirmed,
    isOpen: match.status !== 'completed' && match.status !== 'cancelled',
    headline,
  };
}
