import { useContext } from 'react';
import { ApplicationContext } from '@presentation/context/ApplicationContext';
import { VehicleStateManager } from '@application/state-manager/vehicle-state-manager';

/**
 * useVehicleStateManager - React hook to access the application's VehicleStateManager.
 *
 * Provides access to the centralized state manager from any component within the
 * ApplicationProvider context. Should be called only from components rendered
 * within the ApplicationProvider wrapper.
 *
 * @returns The VehicleStateManager instance from context
 * @throws Error if used outside of ApplicationProvider
 *
 * @example
 * ```typescript
 * const MyComponent = () => {
 *   const stateManager = useVehicleStateManager();
 *   const vehicles = await stateManager.loadVehicles();
 * };
 * ```
 */
export const useVehicleStateManager = (): VehicleStateManager => {
  const context = useContext(ApplicationContext);

  if (!context) {
    throw new Error(
      'useVehicleStateManager must be used within an ApplicationProvider',
    );
  }

  return context.vehicleStateManager;
};
