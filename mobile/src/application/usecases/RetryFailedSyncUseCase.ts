import { SyncPendingPositionsUseCase, SyncPendingResult } from './SyncPendingPositionsUseCase';

/**
 * RetryFailedSyncUseCase
 *
 * Wrapper around SyncPendingPositionsUseCase with explicit naming for UI button actions.
 * This use case is triggered by the "Reintentar todo (N)" button to manually trigger
 * retry of all failed and pending positions.
 *
 * The implementation is essentially the same as SyncPendingPositionsUseCase:
 * - Gets pending and retryable positions
 * - Syncs up to MAX_CONCURRENT positions in parallel
 * - Returns synced/failed counts and updated positions
 *
 * The naming distinction helps document the UI intent: this is a manual retry action,
 * not an automatic background sync.
 */
export class RetryFailedSyncUseCase {
  constructor(private syncPending: SyncPendingPositionsUseCase) {}

  /**
   * Execute manual retry of all pending and failed positions.
   *
   * @returns Promise that resolves to SyncPendingResult
   */
  async execute(): Promise<SyncPendingResult> {
    return this.syncPending.execute();
  }
}
