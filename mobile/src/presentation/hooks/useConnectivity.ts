import { useState, useEffect } from 'react';
import { IConnectivityProvider } from '../../application/ports/IConnectivityProvider';

export interface UseConnectivityResult {
  isOnline: boolean;
  isLoading: boolean;
}

/**
 * useConnectivity
 *
 * Gets current connectivity status and subscribes to changes.
 */
export function useConnectivity(
  connectivityProvider: IConnectivityProvider | null
): UseConnectivityResult {
  const [isOnline, setIsOnline] = useState(true);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    if (!connectivityProvider) {
      setIsLoading(false);
      return;
    }

    // Get initial status
    connectivityProvider.getStatus().then(status => {
      setIsOnline(status === 'online');
      setIsLoading(false);
    }).catch(err => {
      console.error('[useConnectivity] Failed to get initial status:', err);
      setIsLoading(false);
    });

    // Subscribe to changes
    const unsubscribe = connectivityProvider.onConnectivityChanged(status => {
      setIsOnline(status === 'online');
    });

    return () => {
      unsubscribe();
    };
  }, [connectivityProvider]);

  return { isOnline, isLoading };
}
