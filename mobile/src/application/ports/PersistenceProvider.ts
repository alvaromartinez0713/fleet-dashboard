import { Position, PositionStatus } from '../../domain/models/Position';

/**
 * PersistenceProvider
 *
 * Abstraction for local persistence layer (SQLite on mobile).
 *
 * Key responsibilities:
 * - Store and retrieve Position entities with state management
 * - Track synchronization state and retry schedules
 * - Notify listeners when positions change (for reactive UI)
 * - Support offline-first operation with backoff-aware queries
 *
 * Requirements covered:
 * - RF-02: Local persistent storage with synchronization states
 * - RNF-07: Offline-first operation with pending position tracking
 * - RNF-04: Idempotency and retry management via nextRetryAt
 */
export interface PersistenceProvider {
  // ============================================================================
  // CRUD Operations
  // ============================================================================

  /**
   * Save a new position to local storage.
   *
   * Position is always saved with initial status='pending' (unless overridden).
   * Generates and persists event_id and idempotency_key.
   *
   * @param position The Position entity to save
   * @throws PersistenceError if save operation fails
   */
  savePosition(position: Position): Promise<void>;

  /**
   * Update an existing position in local storage.
   *
   * Typically called by SyncPositionUseCase to transition status:
   * - 'pending' → 'sent' (after 202 Accepted)
   * - 'pending' → 'error' or 'validation_error' (after sync failure)
   * - 'error' → 'pending' (manual retry)
   *
   * @param position The Position entity with updated fields
   * @throws PersistenceError if position not found or update fails
   */
  updatePosition(position: Position): Promise<void>;

  /**
   * Retrieve all positions stored locally.
   *
   * Results are ordered by capturedAt descending (newest first).
   * Includes all statuses: 'pending', 'sent', 'error', 'validation_error'.
   *
   * @returns Array of all Position entities (empty array if none exist)
   * @throws PersistenceError if retrieval fails
   */
  getAllPositions(): Promise<Position[]>;

  /**
   * Get position by unique event_id.
   *
   * @param eventId UUID of the position event
   * @returns Position entity or undefined if not found
   * @throws PersistenceError if retrieval fails
   */
  getPositionById(eventId: string): Promise<Position | undefined>;

  /**
   * Delete a specific position by event_id.
   *
   * Note: Currently not used in MVP (positions are retained for audit).
   * Future: May be used when backend confirms final persistence.
   *
   * @param eventId UUID of the position event
   * @throws PersistenceError if deletion fails
   */
  deletePosition(eventId: string): Promise<void>;

  /**
   * Delete all positions from local storage.
   *
   * WARNING: Destructive operation. Used only in testing/reset scenarios.
   *
   * @throws PersistenceError if deletion fails
   */
  deleteAllPositions(): Promise<void>;

  // ============================================================================
  // Filter Operations
  // ============================================================================

  /**
   * Get all positions for a specific vehicle.
   *
   * @param vehicleId Vehicle identifier
   * @returns Array of Position entities for that vehicle (empty if none)
   * @throws PersistenceError if retrieval fails
   */
  getPositionsByVehicle(vehicleId: string): Promise<Position[]>;

  /**
   * Get all positions with a specific status.
   *
   * Returns ALL positions with the given status without time filtering.
   * Used by higher-level logic to filter by nextRetryAt.
   *
   * @param status One of: 'pending', 'sent', 'error', 'validation_error'
   * @returns Array of positions matching that status (empty if none)
   * @throws PersistenceError if retrieval fails
   */
  getPositionsByStatus(status: PositionStatus): Promise<Position[]>;

  // ============================================================================
  // Backoff-Aware Queries (CRITICAL for retry logic)
  // ============================================================================

  /**
   * Get positions ready for synchronization NOW.
   *
   * This is the CRITICAL method for SyncPendingPositionsUseCase.
   *
   * Returns positions where:
   * - status='pending' (never sent)
   * - OR status='error' AND nextRetryAt <= now (backoff time has passed)
   *
   * Does NOT return:
   * - status='sent' (already successful)
   * - status='validation_error' (permanent error, no retry)
   * - status='error' with nextRetryAt > now (too early to retry)
   *
   * Optimization: This query should be efficient with indexes on (status, nextRetryAt).
   *
   * @param now Current time for comparison (defaults to Date.now())
   * @returns Array of Position entities ready to sync
   * @throws PersistenceError if retrieval fails
   */
  getPositionsPendingRetry(now?: Date): Promise<Position[]>;

  // ============================================================================
  // Aggregation Queries (for UI display)
  // ============================================================================

  /**
   * Get total count of all positions stored.
   *
   * @returns Number of positions
   * @throws PersistenceError if query fails
   */
  getPositionCount(): Promise<number>;

  /**
   * Get count of positions with a specific status.
   *
   * Example: getPositionCountByStatus('pending') returns "8"
   * Used by UI to show badge like "Pendientes (8)".
   *
   * @param status One of: 'pending', 'sent', 'error', 'validation_error'
   * @returns Count of positions with that status
   * @throws PersistenceError if query fails
   */
  getPositionCountByStatus(status: PositionStatus): Promise<number>;

  /**
   * Get counts for all statuses in a single query.
   *
   * This is an optimization: instead of 4 separate queries, get all counts at once.
   * Used by UI header to show summary like "12 total: 8 Pendiente | 2 Enviado | 2 Error | 0 Datos inválidos"
   *
   * @returns Object with counts per status
   * @example
   * {
   *   pending: 8,
   *   sent: 2,
   *   error: 2,
   *   validation_error: 0
   * }
   * @throws PersistenceError if query fails
   */
  getPositionCountsByAllStatus(): Promise<Record<PositionStatus, number>>;

  // ============================================================================
  // Event Listeners (for reactive UI updates)
  // ============================================================================

  /**
   * Register a listener that fires when a new position is captured and saved.
   *
   * Used by presentation layer (hooks) to update UI listview when new GPS data arrives.
   *
   * @param callback Function invoked with the newly captured Position
   * @returns Unsubscribe function to remove the listener
   *
   * @example
   * const unsubscribe = persistenceProvider.onPositionCaptured((position) => {
   *   console.log(`New position captured: ${position.eventId}`);
   *   // Update React state to add item to list
   * });
   *
   * // Later, cleanup:
   * unsubscribe();
   */
  onPositionCaptured(
    callback: (position: Position) => void
  ): () => void;

  /**
   * Register a listener that fires when a position's status changes.
   *
   * Used by presentation layer to reactively update UI when position status changes
   * (e.g., 'pending' → 'sent' after successful sync, or 'pending' → 'error' after sync failure).
   *
   * Firing events:
   * - markAsSent() called → fires with position status='sent'
   * - markAsError() called → fires with position status='error' or 'validation_error'
   * - Manual sync button clicked → may fire multiple times
   *
   * @param callback Function invoked with the Position whose status changed
   * @returns Unsubscribe function to remove the listener
   *
   * @example
   * const unsubscribe = persistenceProvider.onPositionStatusChanged((position) => {
   *   console.log(`Position ${position.eventId} status changed to ${position.status}`);
   *   // Update React state to change item's status badge
   * });
   *
   * // Later, cleanup:
   * unsubscribe();
   */
  onPositionStatusChanged(
    callback: (position: Position) => void
  ): () => void;

  // ============================================================================
  // Lifecycle
  // ============================================================================

  /**
   * Initialize the persistence provider.
   *
   * For SQLite: creates database file, runs migrations, creates tables.
   * For in-memory (testing): prepares in-memory database.
   *
   * Must be called before any other operations.
   *
   * @throws PersistenceError if initialization fails
   */
  initialize(): Promise<void>;

  /**
   * Close the persistence provider and clean up resources.
   *
   * For SQLite: closes database connection.
   * For in-memory: releases memory.
   *
   * Must be called when shutting down the app or switching providers.
   *
   * @throws PersistenceError if close fails
   */
  close(): Promise<void>;
}
