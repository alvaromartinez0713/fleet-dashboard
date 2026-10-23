import {
  IConnectivityProvider,
  ConnectivityStatus,
  ConnectivityChangedCallback,
  UnsubscribeFunction,
} from '../../../src/application/ports/IConnectivityProvider';

/**
 * FakeConnectivityProvider
 *
 * Test implementation of IConnectivityProvider for testing connectivity-aware use cases.
 * Allows tests to simulate online/offline states and manual connectivity changes.
 */
export class FakeConnectivityProvider implements IConnectivityProvider {
  private status: ConnectivityStatus = 'online';
  private listeners: ConnectivityChangedCallback[] = [];

  constructor(initialStatus: ConnectivityStatus = 'online') {
    this.status = initialStatus;
  }

  /**
   * Get current connectivity status
   */
  async getStatus(): Promise<ConnectivityStatus> {
    return this.status;
  }

  /**
   * Subscribe to connectivity changes
   */
  onConnectivityChanged(
    callback: ConnectivityChangedCallback
  ): UnsubscribeFunction {
    this.listeners.push(callback);

    // Return unsubscribe function
    return () => {
      const index = this.listeners.indexOf(callback);
      if (index >= 0) {
        this.listeners.splice(index, 1);
      }
    };
  }

  /**
   * Manually trigger connectivity status change (for testing)
   */
  setStatus(newStatus: ConnectivityStatus): void {
    if (this.status !== newStatus) {
      this.status = newStatus;
      // Notify listeners
      this.listeners.forEach(listener => listener(newStatus));
    }
  }

  /**
   * Set to online
   */
  goOnline(): void {
    this.setStatus('online');
  }

  /**
   * Set to offline
   */
  goOffline(): void {
    this.setStatus('offline');
  }

  /**
   * Reset for test isolation
   */
  reset(): void {
    this.status = 'online';
    this.listeners = [];
  }
}
