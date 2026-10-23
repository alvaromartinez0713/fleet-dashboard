import {
  IConnectivityProvider,
  ConnectivityStatus,
  ConnectivityChangedCallback,
  UnsubscribeFunction,
} from '../../application/ports/IConnectivityProvider';

/**
 * NetInfo interface (from @react-native-community/netinfo)
 * Abstracted for flexibility
 */
export interface NetInfoState {
  isConnected: boolean | null;
}

export interface NetInfo {
  fetch(): Promise<NetInfoState>;
  addEventListener(listener: (state: NetInfoState) => void): UnsubscribeFunction;
}

/**
 * RealConnectivityProvider
 *
 * Production implementation of IConnectivityProvider using react-native NetInfo.
 */
export class RealConnectivityProvider implements IConnectivityProvider {
  private netInfo: NetInfo;
  private listeners: ConnectivityChangedCallback[] = [];
  private unsubscribe: UnsubscribeFunction | null = null;

  constructor(netInfo: NetInfo) {
    this.netInfo = netInfo;
  }

  /**
   * Get current connectivity status
   */
  async getStatus(): Promise<ConnectivityStatus> {
    try {
      const state = await this.netInfo.fetch();
      return state.isConnected ? 'online' : 'offline';
    } catch (error) {
      console.error('[RealConnectivityProvider] Failed to get connectivity status:', error);
      // Default to offline if we can't determine status
      return 'offline';
    }
  }

  /**
   * Subscribe to connectivity changes
   */
  onConnectivityChanged(
    callback: ConnectivityChangedCallback
  ): UnsubscribeFunction {
    this.listeners.push(callback);

    // Start listening on first subscription
    if (this.unsubscribe === null) {
      this.unsubscribe = this.netInfo.addEventListener(state => {
        const newStatus: ConnectivityStatus = state.isConnected ? 'online' : 'offline';
        this.listeners.forEach(listener => listener(newStatus));
      });
    }

    // Return unsubscribe function for this listener
    return () => {
      this.listeners = this.listeners.filter(listener => listener !== callback);
      // Stop listening if no more listeners
      if (this.listeners.length === 0 && this.unsubscribe) {
        this.unsubscribe();
        this.unsubscribe = null;
      }
    };
  }

  /**
   * Cleanup all listeners
   */
  cleanup(): void {
    if (this.unsubscribe) {
      this.unsubscribe();
      this.unsubscribe = null;
    }
    this.listeners = [];
  }
}
