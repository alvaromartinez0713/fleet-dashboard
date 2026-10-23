import { IPersistenceProvider } from '../../../src/application/ports/IPersistenceProvider';
import { Position, PositionStatus } from '../../../src/domain/models/Position';

/**
 * FakePersistenceProvider
 *
 * In-memory implementation of IPersistenceProvider for testing.
 * Stores positions in a Map and supports all query operations.
 *
 * Characteristics:
 * - No persistence to disk
 * - Fully deterministic behavior
 * - Supports event listeners for reactive testing
 * - Can be reset for test isolation
 */
export class FakePersistenceProvider implements IPersistenceProvider {
  private positions: Map<string, Position> = new Map();
  private positionCapturedListeners: Array<(position: Position) => void> = [];
  private positionStatusChangedListeners: Array<(position: Position) => void> = [];

  // ============================================================================
  // CRUD Operations
  // ============================================================================

  async savePosition(position: Position): Promise<void> {
    if (!position.eventId) {
      throw new Error('Cannot save position without eventId');
    }
    this.positions.set(position.eventId, position);
    // Fire event for captured position
    this.positionCapturedListeners.forEach(listener => listener(position));
  }

  async updatePosition(position: Position): Promise<void> {
    if (!position.eventId) {
      throw new Error('Cannot update position without eventId');
    }
    if (!this.positions.has(position.eventId)) {
      throw new Error(`Position with eventId ${position.eventId} not found`);
    }
    this.positions.set(position.eventId, position);
    // Fire event for status changed
    this.positionStatusChangedListeners.forEach(listener => listener(position));
  }

  async getAllPositions(): Promise<Position[]> {
    const positionsArray = Array.from(this.positions.values());
    // Sort by capturedAt descending (newest first)
    return positionsArray.sort(
      (a, b) => b.capturedAt.getTime() - a.capturedAt.getTime()
    );
  }

  async getPositionById(eventId: string): Promise<Position | undefined> {
    return this.positions.get(eventId);
  }

  async deletePosition(eventId: string): Promise<void> {
    this.positions.delete(eventId);
  }

  async deleteAllPositions(): Promise<void> {
    this.positions.clear();
  }

  // ============================================================================
  // Filter Operations
  // ============================================================================

  async getPositionsByVehicle(vehicleId: string): Promise<Position[]> {
    const filtered = Array.from(this.positions.values()).filter(
      p => p.vehicleId === vehicleId
    );
    return filtered.sort(
      (a, b) => b.capturedAt.getTime() - a.capturedAt.getTime()
    );
  }

  async getPositionsByStatus(status: PositionStatus): Promise<Position[]> {
    const filtered = Array.from(this.positions.values()).filter(
      p => p.status === status
    );
    return filtered.sort(
      (a, b) => b.capturedAt.getTime() - a.capturedAt.getTime()
    );
  }

  // ============================================================================
  // Backoff-Aware Queries
  // ============================================================================

  async getPositionsPendingRetry(now?: Date): Promise<Position[]> {
    const currentTime = now || new Date();

    const pending = Array.from(this.positions.values()).filter(position => {
      // Include 'pending' status
      if (position.status === 'pending') {
        return true;
      }

      // Include 'error' status if nextRetryAt <= now
      if (position.status === 'error' && position.nextRetryAt) {
        return currentTime >= position.nextRetryAt;
      }

      return false;
    });

    // Sort by nextRetryAt ascending (retry soonest first)
    return pending.sort((a, b) => {
      const aRetry = a.nextRetryAt?.getTime() ?? 0;
      const bRetry = b.nextRetryAt?.getTime() ?? 0;
      return aRetry - bRetry;
    });
  }

  // ============================================================================
  // Aggregation Queries
  // ============================================================================

  async getPositionCount(): Promise<number> {
    return this.positions.size;
  }

  async getPositionCountByStatus(status: PositionStatus): Promise<number> {
    return Array.from(this.positions.values()).filter(
      p => p.status === status
    ).length;
  }

  async getPositionCountsByAllStatus(): Promise<Record<PositionStatus, number>> {
    const counts: Record<PositionStatus, number> = {
      pending: 0,
      sent: 0,
      error: 0,
      validation_error: 0,
    };

    for (const position of this.positions.values()) {
      counts[position.status]++;
    }

    return counts;
  }

  // ============================================================================
  // Event Listeners
  // ============================================================================

  onPositionCaptured(
    callback: (position: Position) => void
  ): () => void {
    this.positionCapturedListeners.push(callback);
    // Return unsubscribe function
    return () => {
      this.positionCapturedListeners = this.positionCapturedListeners.filter(
        listener => listener !== callback
      );
    };
  }

  onPositionStatusChanged(
    callback: (position: Position) => void
  ): () => void {
    this.positionStatusChangedListeners.push(callback);
    // Return unsubscribe function
    return () => {
      this.positionStatusChangedListeners = this.positionStatusChangedListeners.filter(
        listener => listener !== callback
      );
    };
  }

  // ============================================================================
  // Lifecycle
  // ============================================================================

  async initialize(): Promise<void> {
    // No-op for fake implementation
  }

  async close(): Promise<void> {
    this.positions.clear();
    this.positionCapturedListeners = [];
    this.positionStatusChangedListeners = [];
  }

  // ============================================================================
  // Test Helpers
  // ============================================================================

  /**
   * Reset the fake provider for test isolation.
   */
  reset(): void {
    this.positions.clear();
    this.positionCapturedListeners = [];
    this.positionStatusChangedListeners = [];
  }
}
