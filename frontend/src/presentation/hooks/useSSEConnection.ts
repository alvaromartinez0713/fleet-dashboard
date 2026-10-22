import { useEffect, useState } from 'react';
import { ConnectionStatus } from '@application/ports/vehicle-realtime-port';
import { useVehicleStateManager } from './useVehicleStateManager';

/**
 * UseSSEConnectionResult - Return type for useSSEConnection hook.
 *
 * @property status - Current connection status (connected, disconnected, reconnecting, error)
 * @property error - Error object if connection failed, null otherwise
 * @property connected - Boolean convenience property (true if status === 'connected')
 */
export interface UseSSEConnectionResult {
  status: ConnectionStatus;
  error: Error | null;
  connected: boolean;
}

/**
 * useSSEConnection - React hook to manage SSE connection lifecycle.
 *
 * This hook handles:
 * - Establishing SSE connection on component mount
 * - Tracking connection status transitions (connecting, connected, disconnected, error)
 * - Capturing and tracking errors during connection
 * - Automatic cleanup and disconnection on component unmount
 * - Re-establishes connection if status changes while mounted
 *
 * The hook integrates with VehicleStateManager to handle the actual SSE connection
 * via startRealtimeUpdates() and stopRealtimeUpdates() methods.
 *
 * Connection Flow:
 * 1. Component mounts → useEffect runs
 * 2. Calls vehicleStateManager.startRealtimeUpdates(onStatusChange)
 * 3. SSE connection begins, status changes emitted via callback
 * 4. Component unmounts → cleanup function runs
 * 5. Calls vehicleStateManager.stopRealtimeUpdates()
 * 6. SSE connection closed, resources cleaned up
 *
 * Error Handling:
 * - If startRealtimeUpdates throws an error, it's captured and stored in state
 * - Connection status is set to ERROR when connection fails
 * - Errors are cleared when a new connection attempt succeeds
 *
 * Verifies: Requisito 6 (hooks reutilizables), Requisito 3.7 (ciclo de vida), RNF-08 (actualización reactiva)
 *
 * @returns UseSSEConnectionResult with status, error, and connected properties
 *
 * @example
 * ```typescript
 * const MyComponent = () => {
 *   const { status, error, connected } = useSSEConnection();
 *
 *   if (error) {
 *     return <div>Error: {error.message}</div>;
 *   }
 *
 *   return (
 *     <div>
 *       Status: {status}
 *       {connected ? '🟢 Connected' : '🔴 Disconnected'}
 *     </div>
 *   );
 * };
 * ```
 */
export const useSSEConnection = (): UseSSEConnectionResult => {
  // State management for connection status and errors
  const [status, setStatus] = useState<ConnectionStatus>(ConnectionStatus.DISCONNECTED);
  const [error, setError] = useState<Error | null>(null);

  // Access the centralized state manager
  const vehicleStateManager = useVehicleStateManager();

  // Effect: Manage SSE connection lifecycle
  useEffect(() => {
    let unsubscribe: (() => void) | null = null;

    /**
     * Setup SSE connection and handle status changes.
     *
     * This function:
     * 1. Initiates the connection via startRealtimeUpdates
     * 2. Registers a callback to track status changes
     * 3. Stores the unsubscribe function for cleanup
     */
    const setupConnection = async () => {
      try {
        // Clear any previous error when attempting new connection
        setError(null);

        // Start realtime updates and get unsubscribe function
        // The callback receives status changes from the SSE connection
        unsubscribe = await vehicleStateManager.startRealtimeUpdates(
          (connectionStatus: ConnectionStatus) => {
            // Update local state when connection status changes
            setStatus(connectionStatus);

            // If connection enters ERROR state, create an error object
            if (connectionStatus === ConnectionStatus.ERROR) {
              setError(
                new Error(
                  'SSE connection error. Attempting to reconnect with exponential backoff.',
                ),
              );
            }
          },
        );
      } catch (err) {
        // Handle synchronous errors from startRealtimeUpdates
        const errorObj = err instanceof Error ? err : new Error(String(err));
        setError(errorObj);
        setStatus(ConnectionStatus.ERROR);
      }
    };

    // Initiate connection setup
    setupConnection();

    // Cleanup function: Called when component unmounts or dependencies change
    return () => {
      // If we have an unsubscribe function, call it to stop listening to SSE events
      if (unsubscribe) {
        unsubscribe();
      }

      // Disconnect from SSE server
      const stopPromise = vehicleStateManager.stopRealtimeUpdates();
      if (stopPromise && typeof stopPromise.catch === 'function') {
        stopPromise.catch((err) => {
          console.error('Error stopping realtime updates:', err);
        });
      }
    };
  }, [vehicleStateManager]);

  // Compute convenience property: connected = true only if status is exactly CONNECTED
  const connected = status === ConnectionStatus.CONNECTED;

  // Return the connection status interface
  return {
    status,
    error,
    connected,
  };
};
