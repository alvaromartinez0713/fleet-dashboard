import * as SQLite from 'expo-sqlite';
import { Position, PositionStatus } from '../../domain/models/Position';
import { IPersistenceProvider } from '../../application/ports/IPersistenceProvider';

/**
 * SQLitePersistenceProvider
 *
 * Real SQLite database using expo-sqlite.
 * Stores positions in a relational database.
 */
export class SQLitePersistenceProvider implements IPersistenceProvider {
  private db: SQLite.SQLiteDatabase | null = null;
  private positionCapturedListeners: Array<(position: Position) => void> = [];
  private positionStatusChangedListeners: Array<(position: Position) => void> = [];

  /**
   * Initialize the database
   */
  async initialize(): Promise<void> {
    try {
      console.log('[SQLitePersistenceProvider] Initializing SQLite database...');
      
      this.db = await SQLite.openDatabaseAsync('fleet_positions.db');
      
      // Create the positions table
      await this.db.execAsync(`
        CREATE TABLE IF NOT EXISTS positions (
          event_id TEXT PRIMARY KEY,
          vehicle_id TEXT NOT NULL,
          latitude REAL NOT NULL,
          longitude REAL NOT NULL,
          captured_at TEXT NOT NULL,
          status TEXT NOT NULL,
          updated_at TEXT NOT NULL,
          error_message TEXT
        );
        
        CREATE INDEX IF NOT EXISTS idx_positions_status ON positions(status);
        CREATE INDEX IF NOT EXISTS idx_positions_captured_at ON positions(captured_at DESC);
      `);
      
      console.log('[SQLitePersistenceProvider] SQLite database initialized');
    } catch (error) {
      console.error('[SQLitePersistenceProvider] Initialization failed:', error);
      throw new Error(`Failed to initialize SQLite: ${error}`);
    }
  }

  /**
   * Close the database connection
   */
  async close(): Promise<void> {
    if (this.db) {
      await this.db.closeAsync();
      this.db = null;
      console.log('[SQLitePersistenceProvider] Database closed');
    }
  }

  /**
   * Save a position
   */
  async savePosition(position: Position): Promise<void> {
    if (!this.db) throw new Error('Database not initialized');
    
    try {
      await this.db.runAsync(
        `INSERT INTO positions (event_id, vehicle_id, latitude, longitude, captured_at, status, updated_at, error_message)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
        [
          position.eventId,
          position.vehicleId,
          position.latitude,
          position.longitude,
          position.capturedAt.toISOString(),
          position.status,
          position.updatedAt.toISOString(),
          position.errorMessage || null
        ]
      );
      
      console.log(`[SQLitePersistenceProvider] Saved position ${position.eventId}`);
      this.positionCapturedListeners.forEach(listener => listener(position));
    } catch (error) {
      console.error('[SQLitePersistenceProvider] Save failed:', error);
      throw new Error(`Failed to save position: ${error}`);
    }
  }

  /**
   * Get all positions sorted by captured_at descending
   */
  async getAllPositions(): Promise<Position[]> {
    if (!this.db) throw new Error('Database not initialized');
    
    try {
      const rows = await this.db.getAllAsync(
        `SELECT * FROM positions ORDER BY captured_at DESC`
      );
      
      return (rows as any[]).map(row => this.mapRowToPosition(row));
    } catch (error) {
      console.error('[SQLitePersistenceProvider] Get all failed:', error);
      return [];
    }
  }

  /**
   * Get positions by status
   */
  async getPositionsByStatus(status: PositionStatus): Promise<Position[]> {
    if (!this.db) throw new Error('Database not initialized');
    
    try {
      const rows = await this.db.getAllAsync(
        `SELECT * FROM positions WHERE status = ? ORDER BY captured_at DESC`,
        [status]
      );
      
      return (rows as any[]).map(row => this.mapRowToPosition(row));
    } catch (error) {
      console.error('[SQLitePersistenceProvider] Get by status failed:', error);
      return [];
    }
  }

  /**
   * Get count of positions by all statuses
   */
  async getPositionCountsByAllStatus(): Promise<Record<PositionStatus, number>> {
    if (!this.db) throw new Error('Database not initialized');
    
    try {
      const result = await this.db.getAllAsync(
        `SELECT status, COUNT(*) as count FROM positions GROUP BY status`
      );
      
      const counts: Record<PositionStatus, number> = {
        pending: 0,
        sent: 0,
        error: 0,
        validation_error: 0,
      };

      (result as any[]).forEach(row => {
        counts[row.status as PositionStatus] = row.count;
      });

      return counts;
    } catch (error) {
      console.error('[SQLitePersistenceProvider] Get counts failed:', error);
      return { pending: 0, sent: 0, error: 0, validation_error: 0 };
    }
  }

  /**
   * Update position status
   */
  async updatePositionStatus(eventId: string, status: PositionStatus): Promise<void> {
    if (!this.db) throw new Error('Database not initialized');
    
    try {
      const updatedAt = new Date().toISOString();
      
      await this.db.runAsync(
        `UPDATE positions SET status = ?, updated_at = ? WHERE event_id = ?`,
        [status, updatedAt, eventId]
      );
      
      // Fetch the updated position to notify listeners
      const row = await this.db.getFirstAsync(
        `SELECT * FROM positions WHERE event_id = ?`,
        [eventId]
      );
      
      if (row) {
        const position = this.mapRowToPosition(row as any);
        console.log(`[SQLitePersistenceProvider] Updated position ${eventId} status to ${status}`);
        this.positionStatusChangedListeners.forEach(listener => listener(position));
      }
    } catch (error) {
      console.error('[SQLitePersistenceProvider] Update status failed:', error);
      throw new Error(`Failed to update position status: ${error}`);
    }
  }

  /**
   * Get a single position by event ID
   */
  async getPositionByEventId(eventId: string): Promise<Position | null> {
    if (!this.db) throw new Error('Database not initialized');
    
    try {
      const row = await this.db.getFirstAsync(
        `SELECT * FROM positions WHERE event_id = ?`,
        [eventId]
      );
      
      return row ? this.mapRowToPosition(row as any) : null;
    } catch (error) {
      console.error('[SQLitePersistenceProvider] Get by event ID failed:', error);
      return null;
    }
  }

  /**
   * Delete all positions
   */
  async deleteAllPositions(): Promise<void> {
    if (!this.db) throw new Error('Database not initialized');
    
    try {
      await this.db.runAsync(`DELETE FROM positions`);
      console.log('[SQLitePersistenceProvider] Deleted all positions');
    } catch (error) {
      console.error('[SQLitePersistenceProvider] Delete all failed:', error);
      throw new Error(`Failed to delete positions: ${error}`);
    }
  }

  /**
   * Subscribe to position captured events
   */
  onPositionCaptured(listener: (position: Position) => void): () => void {
    this.positionCapturedListeners.push(listener);
    return () => {
      this.positionCapturedListeners = this.positionCapturedListeners.filter(l => l !== listener);
    };
  }

  /**
   * Subscribe to position status changed events
   */
  onPositionStatusChanged(listener: (position: Position) => void): () => void {
    this.positionStatusChangedListeners.push(listener);
    return () => {
      this.positionStatusChangedListeners = this.positionStatusChangedListeners.filter(l => l !== listener);
    };
  }

  /**
   * Map a database row to a Position object
   */
  private mapRowToPosition(row: any): Position {
    return new Position(
      row.event_id,
      row.vehicle_id,
      row.latitude,
      row.longitude,
      new Date(row.captured_at),
      row.status as PositionStatus,
      new Date(row.updated_at),
      row.error_message
    );
  }
}
