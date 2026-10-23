import { useState, useEffect } from 'react';
import { ILocationProvider } from '../../application/ports/LocationProvider';

export interface UseLocationPermissionsResult {
  hasPermission: boolean;
  isLoading: boolean;
  requestPermission: () => Promise<boolean>;
}

/**
 * useLocationPermissions
 *
 * Checks and requests GPS location permissions.
 */
export function useLocationPermissions(
  locationProvider: ILocationProvider | null
): UseLocationPermissionsResult {
  const [hasPermission, setHasPermission] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  const checkPermission = async () => {
    if (!locationProvider) {
      setIsLoading(false);
      return;
    }

    try {
      const granted = locationProvider.isPermissionGranted();
      setHasPermission(granted);
    } catch (err) {
      console.error('[useLocationPermissions] Failed to check permission:', err);
      setHasPermission(false);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    checkPermission();
  }, [locationProvider]);

  const requestPermission = async (): Promise<boolean> => {
    if (!locationProvider) return false;

    try {
      const result = await locationProvider.requestPermissions();
      const granted = result === 'granted';
      setHasPermission(granted);
      return granted;
    } catch (err) {
      console.error('[useLocationPermissions] Failed to request permission:', err);
      return false;
    }
  };

  return { hasPermission, isLoading, requestPermission };
}
