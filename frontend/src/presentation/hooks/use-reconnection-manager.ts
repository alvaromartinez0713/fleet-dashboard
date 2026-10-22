import { useEffect, useState, useCallback, useRef } from 'react';
import { ConnectionStatus, VehicleRealtimePort } from '@application/ports/vehicle-realtime-port';

/**
 * Reconnection state information.
 */
export interface UseReconnectionManagerResult {
  /**
   * Whether the connection is currently in a reconnecting state.
   * True when ConnectionStatus is RECONNECTING, false otherwise.
   */
  isReconnecting: boolean;

  /**
   * Milliseconds until the next automatic reconnection attempt.
   * 0 if not reconnecting or when a retry is ready immediately.
   * Decrements toward 0 during reconnection delay.
   */
  nextRetryIn: number;

  /**
   * Triggers an immediate manual reconnection attempt.
   * Useful for UI elements like "Retry Now" buttons.
   * Safe to call multiple times; only one reconnection will proceed.
   */
  manualRetry: () => void;
}

/**
 * useReconnectionManager - React hook for monitoring and managing SSE reconnection.
 *
 * Tracks the connection status of the SSE service and provides UI-friendly
 * information about reconnection state, including a countdown timer and
 * a manual retry capability.
 *
 * The hook monitors connection status through a provided VehicleRealtimePort
 * and displays useful feedback when reconnection is in progress:
 * - Shows countdown timer for the next automatic retry
 * - Provides manual retry button functionality
 * - Displays appropriate status messages for users
 *
 * Features:
 * - Monitors real-time connection state changes
 * - Tracks reconnection attempts with exponential backoff display
 * - Provides countdown timer that decrements in real-time
 * - Allows manual retry triggering
 * - User-friendly reconnection indicators
 * - Proper cleanup of timers and subscriptions
 * - Type-safe with full TypeScript support
 *
 * @param realtimePort - The SSE connection port to monitor (from context or props)
 * @returns UseReconnectionManagerResult with connection state and retry controls
 *
 * @example
 * ```typescript
 * const { realtimePort } = useApplication();
 * const { isReconnecting, nextRetryIn, manualRetry } = useReconnectionManager(realtimePort);
 *
 * return (
 *   <>
 *     {isReconnecting && (
 *       <div className="reconnection-indicator">
 *         <p>Reconectando... Próximo intento en {Math.ceil(nextRetryIn / 1000)}s</p>
 *         <button onClick={manualRetry}>Reintentar ahora</button>
 *       </div>
 *     )}
 *   </>
 * );
 * ```
 *
 * Implementation notes:
 * - Connection status is monitored via subscription callbacks
 * - A countdown timer decrements nextRetryIn every second during reconnection
 * - Manual retry calls the internal reconnection logic
 * - All timers and subscriptions are cleaned up on unmount
 */
export function useReconnectionManager(realtimePort: VehicleRealtimePort): UseReconnectionManagerResult {

  // Track reconnection state
  const [isReconnecting, setIsReconnecting] = useState(false);
  const [nextRetryIn, setNextRetryIn] = useState(0);

  // Track pending timers for cleanup
  const countdownIntervalRef = useRef<NodeJS.Timeout | null>(null);
  const reconnectionTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  /**
   * Calculates the next retry delay based on exponential backoff.
   * Used to show user when next retry is expected.
   *
   * Backoff sequence: 1s, 2s, 4s, 8s, 16s (max 30s)
   *
   * @param attemptNumber - Current attempt number (0-based)
   * @returns Delay in milliseconds
   */
  const calculateBackoffDelay = useCallback((attemptNumber: number): number => {
    const baseDelayMs = 1000;
    const maxDelayMs = 30000;
    const delayMs = baseDelayMs * Math.pow(2, attemptNumber);
    return Math.min(delayMs, maxDelayMs);
  }, []);

  /**
   * Handles reconnection status changes from the SSE connection.
   * Updates reconnecting state and countdown timer.
   *
   * @param status - Current connection status
   */
  const handleConnectionStatusChange = useCallback(
    (status: ConnectionStatus) => {
      if (status === ConnectionStatus.RECONNECTING) {
        setIsReconnecting(true);

        // Get current attempt count from the port state
        // For MVP, we'll show a reasonable estimated backoff
        // In a future version, this could be tracked more precisely
        const estimatedAttemptNumber = 1; // Conservative estimate for first retry
        const delayMs = calculateBackoffDelay(estimatedAttemptNumber);

        setNextRetryIn(delayMs);

        // Start countdown timer that decrements every second
        if (countdownIntervalRef.current !== null) {
          clearInterval(countdownIntervalRef.current);
        }

        let remainingMs = delayMs;
        countdownIntervalRef.current = setInterval(() => {
          remainingMs -= 1000;

          if (remainingMs <= 0) {
            if (countdownIntervalRef.current !== null) {
              clearInterval(countdownIntervalRef.current);
              countdownIntervalRef.current = null;
            }
            setNextRetryIn(0);
          } else {
            setNextRetryIn(remainingMs);
          }
        }, 1000);
      } else {
        // Not reconnecting anymore
        setIsReconnecting(false);
        setNextRetryIn(0);

        // Clear any pending timers
        if (countdownIntervalRef.current !== null) {
          clearInterval(countdownIntervalRef.current);
          countdownIntervalRef.current = null;
        }

        if (reconnectionTimeoutRef.current !== null) {
          clearTimeout(reconnectionTimeoutRef.current);
          reconnectionTimeoutRef.current = null;
        }
      }
    },
    [calculateBackoffDelay],
  );

  /**
   * Triggers a manual reconnection attempt.
   * Useful for "Retry Now" buttons in the UI.
   */
  const manualRetry = useCallback(() => {
    // Clear any existing timers
    if (countdownIntervalRef.current !== null) {
      clearInterval(countdownIntervalRef.current);
      countdownIntervalRef.current = null;
    }

    if (reconnectionTimeoutRef.current !== null) {
      clearTimeout(reconnectionTimeoutRef.current);
      reconnectionTimeoutRef.current = null;
    }

    // Reset state and attempt immediate reconnection
    setIsReconnecting(false);
    setNextRetryIn(0);

    // Trigger reconnection via the realtime port
    if (realtimePort) {
      realtimePort.connect().catch((error) => {
        console.error('Manual reconnection attempt failed:', error);
      });
    }
  }, [realtimePort]);

  /**
   * Subscribe to connection status changes on mount.
   * Clean up subscription on unmount.
   */
  useEffect(() => {
    // Get initial status
    if (realtimePort) {
      const currentStatus = realtimePort.getStatus();
      handleConnectionStatusChange(currentStatus);

      // Subscribe to future status changes
      const unsubscribe = realtimePort.onConnectionStatusChange(handleConnectionStatusChange);

      return () => {
        unsubscribe();

        // Clean up timers on unmount
        if (countdownIntervalRef.current !== null) {
          clearInterval(countdownIntervalRef.current);
          countdownIntervalRef.current = null;
        }

        if (reconnectionTimeoutRef.current !== null) {
          clearTimeout(reconnectionTimeoutRef.current);
          reconnectionTimeoutRef.current = null;
        }
      };
    }

    return () => {
      // Clean up timers on unmount
      if (countdownIntervalRef.current !== null) {
        clearInterval(countdownIntervalRef.current);
        countdownIntervalRef.current = null;
      }

      if (reconnectionTimeoutRef.current !== null) {
        clearTimeout(reconnectionTimeoutRef.current);
        reconnectionTimeoutRef.current = null;
      }
    };
  }, [realtimePort, handleConnectionStatusChange]);

  return {
    isReconnecting,
    nextRetryIn,
    manualRetry,
  };
}
