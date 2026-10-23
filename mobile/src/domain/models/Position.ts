import { Coordinate } from './Coordinate';
import { TraceContext } from './TraceContext';

/**
 * Position status:
 * - 'pending': Captured locally, not yet sent to backend
 * - 'sent': Backend responded with 202 Accepted
 * - 'error': Transient error (5xx, timeout, network) → will retry with exponential backoff
 * - 'validation_error': Permanent error (4xx) → will NOT retry
 */
export type PositionStatus = 'pending' | 'sent' | 'error' | 'validation_error';

/**
 * Clock interface for injectable time in Position backoff calculations.
 * Allows tests to control time and verify exponential backoff behavior.
 */
export interface Clock {
  now(): Date;
}

/**
 * Position is an immutable entity representing a captured GPS location.
 * Handles state transitions (pending → sent, pending → error → pending_retry).
 * Implements exponential backoff for transient errors: min(2s * 2^(attempts-1), 300s).
 */
export class Position {
  readonly eventId: string; // UUID
  readonly vehicleId: string;
  readonly coordinate: Coordinate;
  readonly capturedAt: Date;
  readonly traceContext: TraceContext;
  readonly status: PositionStatus;
  readonly errorReason?: string;
  readonly isValidationError: boolean;
  readonly updatedAt: Date;
  readonly lastSyncAttempt?: Date;
  readonly syncAttemptCount: number;
  readonly nextRetryAt?: Date;

  constructor(
    eventId: string,
    vehicleId: string,
    coordinate: Coordinate,
    capturedAt: Date,
    traceContext: TraceContext,
    status: PositionStatus = 'pending',
    errorReason?: string,
    updatedAt?: Date,
    isValidationError: boolean = false,
    lastSyncAttempt?: Date,
    syncAttemptCount: number = 0,
    nextRetryAt?: Date
  ) {
    this.eventId = eventId;
    this.vehicleId = vehicleId;
    this.coordinate = coordinate;
    this.capturedAt = capturedAt;
    this.traceContext = traceContext;
    this.status = status;
    this.errorReason = errorReason;
    this.updatedAt = updatedAt || new Date();
    this.isValidationError = isValidationError;
    this.lastSyncAttempt = lastSyncAttempt;
    this.syncAttemptCount = syncAttemptCount;
    this.nextRetryAt = nextRetryAt;
  }

  /**
   * Validate that all required fields are present and valid.
   */
  isValid(): boolean {
    return (
      !!this.eventId &&
      !!this.vehicleId &&
      !!this.coordinate &&
      !!this.capturedAt &&
      !!this.traceContext &&
      ['pending', 'sent', 'error', 'validation_error'].includes(this.status)
    );
  }

  /**
   * Transition from 'pending' to 'sent'.
   * Called when backend responds with 202 Accepted.
   * Clears error info and resets retry counters.
   * Returns a new Position instance (immutability).
   */
  markAsSent(): Position {
    return new Position(
      this.eventId,
      this.vehicleId,
      this.coordinate,
      this.capturedAt,
      this.traceContext,
      'sent',
      undefined, // Clear error reason
      new Date(), // Update timestamp
      false, // No validation error
      new Date(), // Update last sync attempt
      0, // Reset attempt counter
      undefined // Clear next retry time
    );
  }

  /**
   * Transition to 'error' or 'validation_error' status.
   * For transient errors (5xx, network, timeout): calculates next retry time with exponential backoff.
   * For validation errors (4xx): marks as 'validation_error' without retry schedule.
   *
   * Backoff formula: nextRetryAt = now + min(2s * 2^(attempts-1), 300s)
   * Examples:
   * - Attempt 1: 2s
   * - Attempt 2: 4s
   * - Attempt 3: 8s
   * - Attempt 10: 300s (capped)
   *
   * @param reason Error message for logging
   * @param isValidationError true if 4xx (no retry), false if 5xx/timeout (retry with backoff)
   * @param clock Clock provider for getting current time
   * @returns New Position instance with updated error state
   */
  markAsError(
    reason: string,
    isValidationError: boolean,
    clock: Clock
  ): Position {
    const newSyncAttemptCount = this.syncAttemptCount + 1;
    const now = clock.now();

    let nextRetryAt: Date | undefined;

    // Only calculate nextRetryAt if this is NOT a validation error
    if (!isValidationError) {
      // Exponential backoff: min(2s * 2^(attempts-1), 300s)
      const baseDelayMs = 2000; // 2 seconds
      const maxDelayMs = 300000; // 300 seconds = 5 minutes
      const exponentialDelayMs = baseDelayMs * Math.pow(2, newSyncAttemptCount - 1);
      const delayMs = Math.min(exponentialDelayMs, maxDelayMs);

      nextRetryAt = new Date(now.getTime() + delayMs);
    }

    return new Position(
      this.eventId,
      this.vehicleId,
      this.coordinate,
      this.capturedAt,
      this.traceContext,
      isValidationError ? 'validation_error' : 'error',
      reason,
      now,
      isValidationError,
      now,
      newSyncAttemptCount,
      nextRetryAt
    );
  }

  /**
   * Check if this position is ready to be synced NOW.
   * Returns true if:
   * - status='pending' (never sent)
   * - status='error' AND nextRetryAt <= now (time to retry)
   *
   * Returns false if:
   * - status='sent' (already sent successfully)
   * - status='validation_error' (permanent error, don't retry)
   * - status='error' but nextRetryAt > now (too early to retry)
   *
   * @param now Current time for comparison (defaults to now)
   */
  canBeSynced(now: Date = new Date()): boolean {
    if (this.status === 'pending') {
      return true;
    }

    if (this.status === 'error' && this.nextRetryAt !== undefined) {
      return now >= this.nextRetryAt;
    }

    return false;
  }

  /**
   * Check if this position has already been successfully sent.
   */
  isAlreadySent(): boolean {
    return this.status === 'sent';
  }

  /**
   * Check if this position has a permanent (validation) error.
   */
  hasValidationError(): boolean {
    return this.status === 'validation_error';
  }

  /**
   * Get human-readable status label for UI display.
   */
  getStatusLabel(): string {
    const labels: Record<PositionStatus, string> = {
      pending: 'Pendiente',
      sent: 'Enviado',
      error: 'Error',
      validation_error: 'Error de datos',
    };
    return labels[this.status];
  }

  /**
   * Get emoji icon for visual status indicator.
   */
  getStatusIcon(): string {
    const icons: Record<PositionStatus, string> = {
      pending: '🔵',
      sent: '✓',
      error: '⚠️',
      validation_error: '❌',
    };
    return icons[this.status];
  }

  /**
   * Get text describing when this position will retry (for UI display).
   * Only meaningful if status='error' and nextRetryAt is set.
   *
   * Examples:
   * - "Reintentar en 32s"
   * - "Listo para reintentar"
   * - null if not applicable
   */
  getNextRetryText(now: Date = new Date()): string | null {
    if (!this.nextRetryAt) {
      return null;
    }

    const diffMs = this.nextRetryAt.getTime() - now.getTime();
    if (diffMs <= 0) {
      return 'Listo para reintentar';
    }

    const diffSec = Math.ceil(diffMs / 1000);
    return `Reintentar en ${diffSec}s`;
  }

  /**
   * Convert Position to a payload for transmission to the backend API.
   * Backend expects: eventId, vehicleId, latitude, longitude, accuracy, capturedAt, traceId, idempotencyKey
   */
  toPayload(): PositionPayload {
    return {
      eventId: this.eventId,
      vehicleId: this.vehicleId,
      latitude: this.coordinate.latitude,
      longitude: this.coordinate.longitude,
      accuracy: this.coordinate.accuracy,
      capturedAt: this.capturedAt,
      traceId: this.traceContext.traceId,
      idempotencyKey: this.traceContext.idempotencyKey,
    };
  }

  /**
   * String representation for logging.
   */
  toString(): string {
    return `Position(eventId=${this.eventId}, vehicleId=${this.vehicleId}, status=${this.status}, coord=${this.coordinate.toString()})`;
  }
}

/**
 * Payload sent to backend API.
 */
export interface PositionPayload {
  eventId: string;
  vehicleId: string;
  latitude: number;
  longitude: number;
  accuracy?: number;
  capturedAt: Date;
  traceId: string;
  idempotencyKey: string;
}
