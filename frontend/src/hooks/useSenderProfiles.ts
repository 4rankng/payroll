import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useUsersByIds } from '@/hooks/api/useUsers';
import { userService } from '@/services/api/user.service';
import type { UserListResponse } from '@/services/api/user.service';

type UserSummary = UserListResponse['data'][0];

interface UseSenderProfilesOptions {
  enabled?: boolean;
}

interface UseSenderProfilesResult {
  data: Map<number, UserSummary>;
  isLoading: boolean;
  isFetching: boolean;
  hasMissingProfiles: boolean;
  refetch: () => Promise<void>;
}

/**
 * Ensure all sender profiles are available by combining cached batch lookups with individual fetches.
 */
export const useSenderProfiles = (
  ids: number[],
  options: UseSenderProfilesOptions = {}
): UseSenderProfilesResult => {
  const resolvedEnabled = options.enabled ?? true;

  const {
    data: cachedProfiles,
    isLoading: isBatchLoading,
    isFetching: isBatchFetching,
    refetch: refetchBatch,
  } = useUsersByIds(ids, {
    enabled: resolvedEnabled && ids.length > 0,
  });

  const [extraProfiles, setExtraProfiles] = useState<Map<number, UserSummary>>(new Map());
  const [isFetchingMissing, setIsFetchingMissing] = useState(false);
  const pendingIdsRef = useRef<Set<number>>(new Set());

  const combinedProfiles = useMemo(() => {
    const merged = new Map<number, UserSummary>();
    if (cachedProfiles) {
      Object.entries(cachedProfiles).forEach(([key, value]) => {
        merged.set(Number(key), value);
      });
    }
    extraProfiles.forEach((value, key) => {
      merged.set(key, value);
    });
    return merged;
  }, [cachedProfiles, extraProfiles]);

  const missingIds = useMemo(() => {
    if (!resolvedEnabled || ids.length === 0) {
      return [];
    }
    return ids.filter((id) => !combinedProfiles.has(id));
  }, [combinedProfiles, ids, resolvedEnabled]);

  useEffect(() => {
    if (!resolvedEnabled || missingIds.length === 0 || isBatchLoading || isBatchFetching) {
      return;
    }

    const newMissing = missingIds.filter((id) => !pendingIdsRef.current.has(id));
    if (newMissing.length === 0) {
      return;
    }

    newMissing.forEach((id) => {
      pendingIdsRef.current.add(id);
    });

    let isCancelled = false;
    setIsFetchingMissing(true);
    void (async () => {
      try {
        const results = await Promise.all(
          newMissing.map(async (id) => {
            const profile = await userService.getUserById(id);
            return { id, profile };
          })
        );

        if (isCancelled) {
          return;
        }

        setExtraProfiles((previous) => {
          const next = new Map(previous);
          results.forEach(({ id, profile }) => {
            next.set(id, profile);
          });
          return next;
        });
      } finally {
        if (!isCancelled) {
          newMissing.forEach((id) => {
            pendingIdsRef.current.delete(id);
          });
          setIsFetchingMissing(false);
        }
      }
    })();

    return () => {
      isCancelled = true;
    };
  }, [missingIds, resolvedEnabled, isBatchLoading, isBatchFetching]);

  const handleRefetch = useCallback(async () => {
    if (!resolvedEnabled) {
      return;
    }
    pendingIdsRef.current.clear();
    setExtraProfiles(new Map());
    await refetchBatch();
  }, [refetchBatch, resolvedEnabled]);

  const isLoading = resolvedEnabled && (isBatchLoading || (isFetchingMissing && combinedProfiles.size === 0));
  const isFetching = resolvedEnabled && (isBatchFetching || isFetchingMissing);

  return {
    data: combinedProfiles,
    isLoading,
    isFetching,
    hasMissingProfiles: missingIds.length > 0,
    refetch: handleRefetch,
  };
};
