import { useState, useEffect } from 'react';
import { Position, PositionStatus } from '../../domain/models/Position';
import { GetCapturedPositionsUseCase, IPersistenceProvider } from '../../application/usecases';

export interface UsePositionsListResult {
  positions: Position[];
  statusCounts: Record<PositionStatus, number>;
  isLoading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
}

/**
 * usePositionsList
 *
 * Fetches all captured positions and subscribes to updates.
 */
export function usePositionsList(
  getCapturedPositions: GetCapturedPositionsUseCase | null,
  persistenceProvider: IPersistenceProvider | null
): UsePositionsListResult {
  const [positions, setPositions] = useState<Position[]>([]);
  const [statusCounts, setStatusCounts] = useState<Record<PositionStatus, number>>({
    pending: 0,
    sent: 0,
    error: 0,
    validation_error: 0,
  });
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadPositions = async () => {
    if (!getCapturedPositions) return;

    try {
      setIsLoading(true);
      setError(null);
      const allPositions = await getCapturedPositions.execute();
      setPositions(allPositions);

      if (persistenceProvider) {
        const counts = await persistenceProvider.getPositionCountsByAllStatus();
        setStatusCounts(counts);
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Unknown error';
      setError(msg);
      console.error('[usePositionsList] Failed to load positions:', err);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    loadPositions();
  }, [getCapturedPositions]);

  useEffect(() => {
    if (!persistenceProvider) return;

    const unsubscribeCaptured = persistenceProvider.onPositionCaptured(() => {
      loadPositions();
    });

    const unsubscribeStatusChanged = persistenceProvider.onPositionStatusChanged(() => {
      loadPositions();
    });

    return () => {
      unsubscribeCaptured();
      unsubscribeStatusChanged();
    };
  }, [persistenceProvider]);

  return {
    positions,
    statusCounts,
    isLoading,
    error,
    refresh: loadPositions,
  };
}
