import { useEffect } from 'react';
import { IConnectivityProvider } from '../../application/ports/IConnectivityProvider';
import { SyncPendingPositionsUseCase } from '../../application/usecases';

/**
 * useSyncOnReconnect
 *
 * Triggers sync when device transitions from offline to online.
 */
export function useSyncOnReconnect(
  connectivityProvider: IConnectivityProvider | null,
  syncPending: SyncPendingPositionsUseCase | null,
  onSyncTriggered?: () => void
): void {
  useEffect(() => {
    if (!connectivityProvider || !syncPending) {
      return;
    }

    let previousStatus = '';

    const unsubscribe = connectivityProvider.onConnectivityChanged(status => {
      // Trigger sync when transitioning to 'online'
      if (previousStatus === 'offline' && status === 'online') {
        onSyncTriggered?.();
        syncPending.execute().catch(err => {
          console.error('[useSyncOnReconnect] Reconnect sync failed:', err);
        });
      }
      previousStatus = status;
    });

    return () => {
      unsubscribe();
    };
  }, [connectivityProvider, syncPending, onSyncTriggered]);
}
