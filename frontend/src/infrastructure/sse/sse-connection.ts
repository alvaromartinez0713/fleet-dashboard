import {
  ConnectionStatus,
  PositionUpdateEvent,
  Unsubscribe,
  VehicleRealtimePort,
} from '@application/ports/vehicle-realtime-port';

/**
 * SSEConnection - Real-time vehicle position updates via Server-Sent Events (SSE).
 *
 * Implements the VehicleRealtimePort interface for connecting to backend SSE endpoint
 * and receiving position update events. Handles connection lifecycle, automatic
 * reconnection with exponential backoff, and event parsing.
 *
 * Features:
 * - Establishes EventSource connection to backend SSE endpoint
 * - Parses JSON events and validates structure
 * - Manages connection state transitions
 * - Automatic reconnection with exponential backoff on disconnect
 * - Event callback management for position updates and status changes
 * - Proper resource cleanup on disconnect
 *
 * @example
 * ```typescript
 * const port = new SSEConnection('http://api.example.com/events');
 * await port.connect();
 *
 * const unsubscribe = port.onPositionUpdate((event) => {
 *   console.log(`Vehicle ${event.vehicleId} at ${event.latitude}, ${event.longitude}`);
 * });
 *
 * // Cleanup
 * unsubscribe();
 * await port.disconnect();
 * ```
 */
export class SSEConnection implements VehicleRealtimePort {
  /** EventSource instance for SSE connection */
  private eventSource: EventSource | null = null;

  /** Current connection status */
  private status: ConnectionStatus = ConnectionStatus.DISCONNECTED;

  /** Registered callbacks for position update events */
  private updateCallbacks: ((event: PositionUpdateEvent) => void)[] = [];

  /** Registered callbacks for connection status changes */
  private statusCallbacks: ((status: ConnectionStatus) => void)[] = [];

  /** Current number of reconnection attempts */
  private reconnectAttempts: number = 0;

  /** Maximum number of reconnection attempts before giving up */
  private readonly maxReconnectAttempts: number = 5;

  /** Base delay for exponential backoff (in milliseconds) */
  private readonly baseBackoffMs: number = 1000;

  /** Maximum delay for exponential backoff (in milliseconds) */
  private readonly maxBackoffMs: number = 30000;

  /** Timer ID for pending reconnection, used for cleanup */
  private reconnectTimeoutId: NodeJS.Timeout | null = null;

  /**
   * Creates a new SSEConnection instance.
   *
   * @param sseUrl - URL of the SSE endpoint (defaults to env var VITE_SSE_ENDPOINT)
   */
  constructor(private sseUrl: string = import.meta.env.VITE_SSE_ENDPOINT || 'http://localhost:8080/api/vehicles/updates') {
    if (!this.sseUrl) {
      throw new Error(
        'SSE endpoint URL must be provided or VITE_SSE_ENDPOINT environment variable must be set',
      );
    }
  }

  /**
   * Establishes connection to the SSE endpoint.
   *
   * If already connected or connecting, this method returns without action.
   * Transititions through states: DISCONNECTED → CONNECTING → CONNECTED
   * On error, transitions to ERROR and attempts reconnection with backoff.
   *
   * @throws {Error} If SSE is not supported in the environment
   */
  async connect(): Promise<void> {
    // Already connected or connecting
    if (
      this.status === ConnectionStatus.CONNECTED ||
      this.status === ConnectionStatus.RECONNECTING
    ) {
      return;
    }

    // Clear any pending reconnection timeout
    if (this.reconnectTimeoutId !== null) {
      clearTimeout(this.reconnectTimeoutId);
      this.reconnectTimeoutId = null;
    }

    this.setStatus(ConnectionStatus.RECONNECTING);

    try {
      // Verify EventSource is available
      if (typeof EventSource === 'undefined') {
        throw new Error('EventSource is not supported in this environment');
      }

      // Create EventSource connection
      this.eventSource = new EventSource(this.sseUrl);

      // Register handler for position_update events
      this.eventSource.addEventListener('position_update', (event: Event) => {
        if (event instanceof MessageEvent) {
          this.handlePositionUpdateEvent(event);
        }
      });

      // Register handler for connection errors
      this.eventSource.addEventListener('error', () => {
        this.handleConnectionError();
      });

      // Transition to connected state
      this.setStatus(ConnectionStatus.CONNECTED);
      // Reset reconnection attempts on successful connection
      this.reconnectAttempts = 0;
    } catch (error) {
      this.setStatus(ConnectionStatus.ERROR);
      this.handleConnectionError();
    }
  }

  /**
   * Handles incoming position update event from SSE.
   *
   * Parses JSON event data and triggers all registered update callbacks.
   * Invalid JSON or missing fields are logged as errors but don't interrupt
   * the event stream.
   *
   * @param event - MessageEvent from EventSource with JSON data
   */
  private handlePositionUpdateEvent(event: MessageEvent): void {
    try {
      const data = JSON.parse(event.data) as PositionUpdateEvent;

      // Basic validation of required fields
      if (!data.vehicleId || data.latitude === undefined || data.longitude === undefined) {
        console.error('Invalid position update event: missing required fields', data);
        return;
      }

      // Trigger all registered callbacks
      this.updateCallbacks.forEach((callback) => {
        try {
          callback(data);
        } catch (err) {
          console.error('Error in position update callback:', err);
        }
      });
    } catch (error) {
      console.error('Failed to parse position update event:', error, event.data);
    }
  }

  /**
   * Handles SSE connection errors and attempts reconnection with exponential backoff.
   *
   * Closes current EventSource and schedules reconnection attempt using
   * exponential backoff: 1s, 2s, 4s, 8s, 16s (max 30s).
   *
   * After maxReconnectAttempts failures, transitions to DISCONNECTED state
   * and stops attempting reconnection.
   */
  private handleConnectionError(): void {
    // Close current EventSource
    if (this.eventSource !== null) {
      this.eventSource.close();
      this.eventSource = null;
    }

    // Check if we should attempt reconnection
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      // Calculate exponential backoff delay
      const backoffMs = Math.min(
        this.baseBackoffMs * Math.pow(2, this.reconnectAttempts),
        this.maxBackoffMs,
      );

      this.reconnectAttempts++;

      // Schedule reconnection attempt
      this.reconnectTimeoutId = setTimeout(() => {
        this.reconnectTimeoutId = null;
        this.connect().catch((error) => {
          console.error('Reconnection attempt failed:', error);
        });
      }, backoffMs);
    } else {
      // Max reconnection attempts exceeded
      this.setStatus(ConnectionStatus.DISCONNECTED);
    }
  }

  /**
   * Registers a callback for position update events.
   *
   * The callback is invoked whenever a position_update event is received
   * from the SSE endpoint. Multiple callbacks can be registered.
   *
   * @param callback - Function to call when position update is received
   * @returns Function to unsubscribe the callback
   *
   * @example
   * ```typescript
   * const unsubscribe = port.onPositionUpdate((event) => {
   *   console.log(`Updated: ${event.vehicleId}`);
   * });
   * unsubscribe(); // Remove callback
   * ```
   */
  onPositionUpdate(callback: (event: PositionUpdateEvent) => void): Unsubscribe {
    this.updateCallbacks.push(callback);

    // Return unsubscribe function
    return () => {
      this.updateCallbacks = this.updateCallbacks.filter((cb) => cb !== callback);
    };
  }

  /**
   * Registers a callback for connection status changes.
   *
   * The callback is invoked whenever the connection status transitions
   * (e.g., DISCONNECTED → RECONNECTING → CONNECTED).
   * Useful for UI feedback like connection indicators.
   *
   * @param callback - Function to call when status changes
   * @returns Function to unsubscribe the callback
   *
   * @example
   * ```typescript
   * const unsubscribe = port.onConnectionStatusChange((status) => {
   *   console.log(`Connection status: ${status}`);
   * });
   * unsubscribe(); // Remove callback
   * ```
   */
  onConnectionStatusChange(callback: (status: ConnectionStatus) => void): Unsubscribe {
    this.statusCallbacks.push(callback);

    // Return unsubscribe function
    return () => {
      this.statusCallbacks = this.statusCallbacks.filter((cb) => cb !== callback);
    };
  }

  /**
   * Disconnects from the SSE endpoint and releases all resources.
   *
   * - Closes EventSource connection
   * - Cancels any pending reconnection attempts
   * - Transitions to DISCONNECTED state
   * - Preserves registered callbacks but stops invoking them
   */
  async disconnect(): Promise<void> {
    // Cancel pending reconnection if any
    if (this.reconnectTimeoutId !== null) {
      clearTimeout(this.reconnectTimeoutId);
      this.reconnectTimeoutId = null;
    }

    // Close EventSource
    if (this.eventSource !== null) {
      this.eventSource.close();
      this.eventSource = null;
    }

    this.setStatus(ConnectionStatus.DISCONNECTED);
  }

  /**
   * Returns the current connection status.
   *
   * @returns Current ConnectionStatus value
   */
  getStatus(): ConnectionStatus {
    return this.status;
  }

  /**
   * Updates internal status and notifies all status callbacks.
   *
   * @param status - New connection status
   */
  private setStatus(status: ConnectionStatus): void {
    this.status = status;

    // Notify all registered status callbacks
    this.statusCallbacks.forEach((callback) => {
      try {
        callback(status);
      } catch (error) {
        console.error('Error in connection status callback:', error);
      }
    });
  }
}
