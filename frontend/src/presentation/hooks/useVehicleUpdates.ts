import { useEffect, useState } from 'react';
import { Vehicle } from '@domain/entities/vehicle';
import { useVehicleStateManager } from './useVehicleStateManager';

/**
 * useVehicleUpdates - React hook for granular vehicle position subscriptions.
 *
 * Subscribes to specific vehicle position updates, enabling fine-grained updates
 * for individual vehicles without causing unnecessary re-renders of sibling components.
 *
 * Verifies: Requisito 6 (hooks reutilizables), Requisito 6.3 (suscripción granular)
 *
 * @param vehicleId - The unique identifier of the vehicle to subscribe to
 * @returns Object containing current vehicle position state and loading status
 *
 * @example
 * ```typescript
 * const VehicleRow: React.FC<{ vehicleId: string }> = ({ vehicleId }) => {
 *   const { vehicle, loading } = useVehicleUpdates(vehicleId);
 *
 *   if (loading) return <div>Loading...</div>;
 *   if (!vehicle) return <div>Vehicle not found</div>;
 *
 *   return (
 *     <div>
 *       {vehicle.licensePlate}: {vehicle.position.latitude}, {vehicle.position.longitude}
 *     </div>
 *   );
 * };
 * ```
 */
export interface UseVehicleUpdatesResult {
  /** Current vehicle data; null if not found or still loading */
  vehicle: Vehicle | null;
  /** True while initial vehicle fetch is in progress */
  loading: boolean;
}

/**
 * Hook for subscribing to position updates of a specific vehicle.
 *
 * Behavior:
 * - On mount: Loads the initial vehicle data for the given vehicleId
 * - During lifecycle: Subscribes to granular updates for only this vehicle
 * - On unmount: Unsubscribes to prevent memory leaks
 * - Missing vehicle: Gracefully returns null without throwing
 * - Efficient renders: Component only re-renders when this specific vehicle updates
 *
 * @param vehicleId - The unique identifier of the vehicle to subscribe to
 * @returns UseVehicleUpdatesResult with vehicle data and loading state
 *
 * @throws No exceptions; handles missing vehicle gracefully
 */
export const useVehicleUpdates = (vehicleId: string): UseVehicleUpdatesResult => {
  const [vehicle, setVehicle] = useState<Vehicle | null>(null);
  const [loading, setLoading] = useState(true);
  const vehicleStateManager = useVehicleStateManager();

  useEffect(() => {
    setLoading(true);

    // Load initial vehicle data
    const loadInitialVehicle = async () => {
      try {
        const found = await vehicleStateManager.getVehicleById(vehicleId);
        setVehicle(found);
      } catch (error) {
        console.error(`Error loading vehicle ${vehicleId}:`, error);
        setVehicle(null);
      } finally {
        setLoading(false);
      }
    };

    // Execute initial load
    loadInitialVehicle();

    // Subscribe to granular updates for this vehicle
    const unsubscribe = vehicleStateManager.subscribeToVehicleUpdates(
      vehicleId,
      (updated: Vehicle) => {
        setVehicle(updated);
      },
    );

    // Cleanup: unsubscribe when component unmounts or vehicleId changes
    return unsubscribe;
  }, [vehicleId, vehicleStateManager]);

  return { vehicle, loading };
};
