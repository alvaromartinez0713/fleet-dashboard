import { useEffect, useRef } from 'react';
import { SyncPendingPositionsUseCase } from '../../application/usecases';

/**
 * useAutoSync
 *
 * Automatically syncs pending positions every 10 seconds when online.
 * Cleans up interval on unmount.
 */
export function useAutoSync(
  syncPending: SyncPendingPositionsUseCase | null,
  isOnline: boolean,
  onSyncComplete?: (synced: number, failed: number) => void
): void {
  const intervalRef = useRef<NodeJS.Timeout | null>(null);

  useEffect(() => {
    if (!syncPending || !isOnline) {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
      return;
    }

    // Run sync immediately on mount/reconnect
    syncPending.execute().then(result => {
      onSyncComplete?.(result.synced, result.failed);
    }).catch(err => {
      console.error('[useAutoSync] Immediate sync failed:', err);
    });

    // Then set up interval for every 10 seconds
    intervalRef.current = setInterval(() => {
      syncPending.execute().then(result => {
        onSyncComplete?.(result.synced, result.failed);
      }).catch(err => {
        console.error('[useAutoSync] Interval sync failed:', err);
      });
    }, 10000); // 10 seconds

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
    };
  }, [syncPending, isOnline, onSyncComplete]);
}
