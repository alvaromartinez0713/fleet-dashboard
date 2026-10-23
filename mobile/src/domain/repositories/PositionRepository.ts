import { Position, PositionStatus } from '../models';

/**
 * PositionRepository is an abstraction for persisting and retrieving Position entities.
 * No implementation details here - just the contract.
 * Implementation can be SQLite, in-memory, or any other storage mechanism.
 */
export interface PositionRepository {
  /**
   * Save a new position to persistence.
   * The position will have status='pending' when first saved.
   */
  save(position: Position): Promise<void>;

  /**
   * Update an existing position (typically to change its status after sync attempt).
   */
  update(position: Position): Promise<void>;

  /**
   * Retrieve all positions stored in the repository.
   */
  getAll(): Promise<Position[]>;

  /**
   * Retrieve positions for a specific vehicle.
   */
  getByVehicleId(vehicleId: string): Promise<Position[]>;

  /**
   * Retrieve positions filtered by status.
   */
  getByStatus(status: PositionStatus): Promise<Position[]>;

  /**
   * Get a single position by its event ID.
   */
  getById(eventId: string): Promise<Position | undefined>;

  /**
   * Delete a position by its event ID.
   */
  delete(eventId: string): Promise<void>;

  /**
   * Delete all positions (for testing/cleanup).
   */
  deleteAll(): Promise<void>;

  /**
   * Get total count of positions.
   */
  getCount(): Promise<number>;

  /**
   * Get count of positions with a specific status.
   */
  getCountByStatus(status: PositionStatus): Promise<number>;

  /**
   * Initialize or verify the repository (create tables, etc.)
   */
  initialize(): Promise<void>;

  /**
   * Close the repository (cleanup resources).
   */
  close(): Promise<void>;
}
