/**
 * ConnectivityProvider
 *
 * Abstracts platform-specific network state detection for the mobile application.
 * Allows the application layer to check current connectivity status and react to changes
 * without depending on specific platform implementations (iOS, Android, etc.).
 *
 * This interface supports RNF-07 (Operación offline-first) by enabling the application to:
 * 1. Check current connectivity status synchronously or asynchronously
 * 2. Subscribe to connectivity changes (online → offline → online)
 * 3. Unsubscribe from connectivity updates when no longer needed
 *
 * Implementation details:
 * - The actual network detection is delegated to platform-specific adapters
 * - Changes should be detected and reported in a timely manner
 * - The callback should be invoked whenever the status transitions between online/offline
 */

/**
 * Connectivity status type
 * - 'online': Device has network connectivity
 * - 'offline': Device has no network connectivity
 */
export type ConnectivityStatus = 'online' | 'offline';

/**
 * Callback function type for connectivity changes
 */
export type ConnectivityChangedCallback = (status: ConnectivityStatus) => void;

/**
 * Unsubscribe function type returned by onConnectivityChanged
 */
export type UnsubscribeFunction = () => void;

/**
 * Port interface for connectivity detection
 *
 * Implementations of this interface should handle:
 * - Querying the current network connectivity status
 * - Registering listeners for connectivity state changes
 * - Proper cleanup when listeners are unsubscribed
 */
export interface ConnectivityProvider {
  /**
   * Get the current connectivity status
   *
   * @returns Promise that resolves to the current connectivity status
   * @throws May throw if unable to determine status, implementation-dependent
   *
   * @example
   * const status = await provider.getStatus();
   * if (status === 'offline') {
   *   // Handle offline mode
   * }
   */
  getStatus(): Promise<ConnectivityStatus>;

  /**
   * Register a callback to be invoked when connectivity status changes
   *
   * The callback will be called whenever the connectivity status transitions
   * between online and offline states. The callback receives the new status
   * as its parameter.
   *
   * @param callback - Function to be called on connectivity changes
   * @returns Function to unsubscribe the listener. Calling this function removes
   *          the callback from future updates and cleans up resources.
   *
   * @example
   * const unsubscribe = provider.onConnectivityChanged((status) => {
   *   console.log('Connectivity changed to:', status);
   * });
   *
   * // Later, when you no longer need updates
   * unsubscribe();
   */
  onConnectivityChanged(
    callback: ConnectivityChangedCallback
  ): UnsubscribeFunction;
}
