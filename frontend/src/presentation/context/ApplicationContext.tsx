import { createContext, ReactNode, useContext, useEffect, useMemo } from 'react';
import { VehicleStateManager } from '@application/state-manager/vehicle-state-manager';
import { LoadVehiclesUseCase } from '@application/usecases/load-vehicles';
import { SubscribeToPositionUpdatesUseCase } from '@application/usecases/subscribe-to-position-updates';
import { HandlePositionUpdateUseCase } from '@application/usecases/handle-position-update';
import { VehicleDataProvider } from '@application/ports/vehicle-data-provider';
import { VehicleRealtimePort } from '@application/ports/vehicle-realtime-port';
import { VehicleStorage } from '@application/ports/vehicle-storage';

/**
 * ApplicationContext - React Context for providing application-level dependencies.
 *
 * Contains instances of all application services and coordinators that should be
 * available throughout the frontend. This ensures:
 * - Single instance of VehicleStateManager (shared state)
 * - Clean dependency injection
 * - Easy access from any component via useApplicationContext() hook
 *
 * Verifies: Requisito 5 (inyección de dependencias), Requisito 13.1
 */
export interface ApplicationContextType {
  /**
   * The centralized state manager for all vehicle-related operations.
   * Coordinates loading, realtime updates, and state propagation.
   */
  vehicleStateManager: VehicleStateManager;
}

/**
 * React Context for application-level services.
 *
 * Must be provided by wrapping components with ApplicationProvider.
 * Attempting to use context consumers outside of ApplicationProvider
 * will throw an error.
 */
export const ApplicationContext = createContext<ApplicationContextType | undefined>(
  undefined,
);

ApplicationContext.displayName = 'ApplicationContext';

/**
 * Props for ApplicationProvider component.
 */
export interface ApplicationProviderProps {
  /** React components to render as children */
  children: ReactNode;
  /** Vehicle data provider (can be mock or API-based) */
  dataProvider: VehicleDataProvider;
  /** Real-time updates provider (SSE connection) */
  realtimePort: VehicleRealtimePort;
  /** Vehicle storage (persistence layer) */
  storage: VehicleStorage;
}

/**
 * ApplicationProvider - React component that provides application-level dependencies.
 *
 * Responsibilities:
 * - Initialize VehicleStateManager as a singleton
 * - Create all required use cases and inject dependencies
 * - Provide context to all descendant components
 * - Handle initialization and cleanup
 * - Prevent memory leaks by proper resource cleanup
 *
 * Verifies: Requisito 5 (inyección de dependencias), Requisito 13.1
 *
 * @param props - ApplicationProviderProps with dataProvider, realtimePort, storage, and children
 * @returns React component that wraps children with ApplicationContext provider
 */
export const ApplicationProvider: React.FC<ApplicationProviderProps> = ({
  children,
  dataProvider,
  realtimePort,
  storage,
}) => {
  /**
   * Initialize context value with VehicleStateManager singleton.
   *
   * Uses useMemo to ensure:
   * - Single instance of VehicleStateManager per provider mount
   * - Dependencies (dataProvider, realtimePort, storage) are stable
   * - Proper initialization of all use cases with injected dependencies
   *
   * Verifies: Singleton pattern, proper dependency injection
   */
  const contextValue = useMemo<ApplicationContextType>(() => {
    // Create use cases
    const loadVehiclesUseCase = new LoadVehiclesUseCase(dataProvider, storage);
    const subscribeToPositionUpdatesUseCase = new SubscribeToPositionUpdatesUseCase(
      realtimePort,
    );
    const handlePositionUpdateUseCase = new HandlePositionUpdateUseCase(storage);

    // Create state manager with injected use cases
    const vehicleStateManager = new VehicleStateManager(
      loadVehiclesUseCase,
      subscribeToPositionUpdatesUseCase,
      handlePositionUpdateUseCase,
    );

    return {
      vehicleStateManager,
    };
  }, [dataProvider, realtimePort, storage]);

  /**
   * Cleanup on unmount: Stop realtime updates to prevent memory leaks.
   *
   * Ensures:
   * - SSE connection is properly closed when provider unmounts
   * - All subscriptions are cleaned up
   * - No orphaned event listeners or timers
   *
   * Verifies: No memory leaks, proper resource cleanup
   */
  useEffect(() => {
    return () => {
      // Stop realtime updates when provider unmounts
      contextValue.vehicleStateManager.stopRealtimeUpdates().catch((error) => {
        console.error('Error stopping realtime updates on cleanup:', error);
      });
    };
  }, [contextValue.vehicleStateManager]);

  return (
    <ApplicationContext.Provider value={contextValue}>{children}</ApplicationContext.Provider>
  );
};

/**
 * useApplicationContext - Custom hook to access ApplicationContext.
 *
 * Responsibilities:
 * - Get context using useContext
 * - Throw error if used outside ApplicationProvider
 * - Return fully typed ApplicationContextType
 *
 * Verifies: Error handling, proper hook usage
 *
 * @returns ApplicationContextType containing vehicleStateManager and other app-wide state
 * @throws Error if used outside ApplicationProvider
 */
export const useApplicationContext = (): ApplicationContextType => {
  const context = useContext(ApplicationContext);

  if (context === undefined) {
    throw new Error(
      'useApplicationContext must be used within an ApplicationProvider. ' +
        'Ensure your component is wrapped with <ApplicationProvider>.',
    );
  }

  return context;
};
