import { IPersistenceProvider } from '../../../src/application/ports/IPersistenceProvider';
import { Position, PositionStatus } from '../../../src/domain/models/Position';

/**
 * Fake in-memory persistence for testing.
 * No database, pure memory storage.
 */
export class FakePersistenceProvider implements IPersistenceProvider {
  private positions: Map<string, Position> = new Map();
  private capturedListeners: Array<(position: Position) => void> = [];
  private statusChangedListeners: Array<(position: Position) => void> = [];

  async initialize(): Promise<void> {}
  async close(): Promise<void> {}

  async savePosition(position: Position): Promise<void> {
    this.positions.set(position.eventId, position);
    this.capturedListeners.forEach(l => l(position));
  }

  async getAllPositions(): Promise<Position[]> {
    return Array.from(this.positions.values()).sort(
      (a, b) => b.capturedAt.getTime() - a.capturedAt.getTime()
    );
  }

  async getPositionsByStatus(status: PositionStatus): Promise<Position[]> {
    return Array.from(this.positions.values())
      .filter(p => p.status === status)
      .sort((a, b) => b.capturedAt.getTime() - a.capturedAt.getTime());
  }

  async getPositionCountsByAllStatus(): Promise<Record<PositionStatus, number>> {
    const counts: Record<PositionStatus, number> = {
      pending: 0,
      sent: 0,
      error: 0,
      validation_error: 0,
    };
    this.positions.forEach(p => counts[p.status]++);
    return counts;
  }

  async updatePositionStatus(eventId: string, status: PositionStatus): Promise<void> {
    const pos = this.positions.get(eventId);
    if (pos) {
      pos.status = status;
      pos.updatedAt = new Date();
      this.statusChangedListeners.forEach(l => l(pos));
    }
  }

  async getPositionByEventId(eventId: string): Promise<Position | null> {
    return this.positions.get(eventId) || null;
  }

  async deleteAllPositions(): Promise<void> {
    this.positions.clear();
  }

  onPositionCaptured(listener: (position: Position) => void): () => void {
    this.capturedListeners.push(listener);
    return () => {
      this.capturedListeners = this.capturedListeners.filter(l => l !== listener);
    };
  }

  onPositionStatusChanged(listener: (position: Position) => void): () => void {
    this.statusChangedListeners.push(listener);
    return () => {
      this.statusChangedListeners = this.statusChangedListeners.filter(l => l !== listener);
    };
  }
}
