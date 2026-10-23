import { Position, PositionStatus } from '../../../src/domain/models';
import { v4 as uuidv4 } from 'uuid';
import { CoordinateMother } from './CoordinateMother';
import { TraceContextMother } from './TraceContextMother';

/**
 * Object Mother for creating Position test data.
 * Provides named scenarios with valid default values.
 */
export class PositionMother {
  /**
   * Valid position with status='pending' (just captured).
   */
  static valid(): PositionBuilder {
    const now = new Date();
    return new PositionBuilder({
      eventId: uuidv4(),
      vehicleId: 'VEHICLE-001',
      coordinate: CoordinateMother.valid().build(),
      capturedAt: now,
      traceContext: TraceContextMother.valid().build(),
      status: 'pending',
      updatedAt: now,
    });
  }

  /**
   * Position with status='sent' (successfully synced).
   */
  static sent(): PositionBuilder {
    const now = new Date();
    return new PositionBuilder({
      eventId: uuidv4(),
      vehicleId: 'VEHICLE-001',
      coordinate: CoordinateMother.valid().build(),
      capturedAt: now,
      traceContext: TraceContextMother.valid().build(),
      status: 'sent',
      updatedAt: now,
      lastSyncAttempt: now,
      syncAttemptCount: 1,
    });
  }

  /**
   * Position with status='error' (transient error, will retry).
   */
  static withError(): PositionBuilder {
    const now = new Date();
    const nextRetry = new Date(now.getTime() + 2000); // 2 seconds
    return new PositionBuilder({
      eventId: uuidv4(),
      vehicleId: 'VEHICLE-001',
      coordinate: CoordinateMother.valid().build(),
      capturedAt: now,
      traceContext: TraceContextMother.valid().build(),
      status: 'error',
      errorReason: 'Network timeout',
      updatedAt: now,
      isValidationError: false,
      lastSyncAttempt: now,
      syncAttemptCount: 1,
      nextRetryAt: nextRetry,
    });
  }

  /**
   * Position with status='validation_error' (permanent error, no retry).
   */
  static withValidationError(): PositionBuilder {
    const now = new Date();
    return new PositionBuilder({
      eventId: uuidv4(),
      vehicleId: 'VEHICLE-001',
      coordinate: CoordinateMother.valid().build(),
      capturedAt: now,
      traceContext: TraceContextMother.valid().build(),
      status: 'validation_error',
      errorReason: 'Invalid vehicle ID',
      updatedAt: now,
      isValidationError: true,
      lastSyncAttempt: now,
      syncAttemptCount: 1,
    });
  }

  /**
   * Deterministic position for predictable testing.
   */
  static deterministic(): PositionBuilder {
    const now = new Date('2024-01-01T00:00:00Z');
    return new PositionBuilder({
      eventId: '00000000-0000-0000-0000-000000000001',
      vehicleId: 'TEST-VEHICLE',
      coordinate: CoordinateMother.deterministic().build(),
      capturedAt: now,
      traceContext: TraceContextMother.deterministic().build(),
      status: 'pending',
      updatedAt: now,
    });
  }
}

/**
 * Test Data Builder for Position.
 * Allows overriding specific fields while maintaining defaults.
 */
export class PositionBuilder {
  private eventId: string;
  private vehicleId: string;
  private coordinate: ReturnType<typeof CoordinateMother.valid>['build'];
  private capturedAt: Date;
  private traceContext: ReturnType<typeof TraceContextMother.valid>['build'];
  private status: PositionStatus;
  private errorReason?: string;
  private updatedAt: Date;
  private isValidationError: boolean;
  private lastSyncAttempt?: Date;
  private syncAttemptCount: number;
  private nextRetryAt?: Date;

  constructor(data: {
    eventId: string;
    vehicleId: string;
    coordinate: ReturnType<typeof CoordinateMother.valid>['build'];
    capturedAt: Date;
    traceContext: ReturnType<typeof TraceContextMother.valid>['build'];
    status: PositionStatus;
    errorReason?: string;
    updatedAt?: Date;
    isValidationError?: boolean;
    lastSyncAttempt?: Date;
    syncAttemptCount?: number;
    nextRetryAt?: Date;
  }) {
    this.eventId = data.eventId;
    this.vehicleId = data.vehicleId;
    this.coordinate = data.coordinate;
    this.capturedAt = data.capturedAt;
    this.traceContext = data.traceContext;
    this.status = data.status;
    this.errorReason = data.errorReason;
    this.updatedAt = data.updatedAt || new Date();
    this.isValidationError = data.isValidationError ?? false;
    this.lastSyncAttempt = data.lastSyncAttempt;
    this.syncAttemptCount = data.syncAttemptCount ?? 0;
    this.nextRetryAt = data.nextRetryAt;
  }

  withEventId(eventId: string): PositionBuilder {
    this.eventId = eventId;
    return this;
  }

  withVehicleId(vehicleId: string): PositionBuilder {
    this.vehicleId = vehicleId;
    return this;
  }

  withStatus(status: PositionStatus): PositionBuilder {
    this.status = status;
    return this;
  }

  withErrorReason(reason: string | undefined): PositionBuilder {
    this.errorReason = reason;
    return this;
  }

  withSyncAttemptCount(count: number): PositionBuilder {
    this.syncAttemptCount = count;
    return this;
  }

  withNextRetryAt(date: Date | undefined): PositionBuilder {
    this.nextRetryAt = date;
    return this;
  }

  build(): Position {
    return new Position(
      this.eventId,
      this.vehicleId,
      this.coordinate,
      this.capturedAt,
      this.traceContext,
      this.status,
      this.errorReason,
      this.updatedAt,
      this.isValidationError,
      this.lastSyncAttempt,
      this.syncAttemptCount,
      this.nextRetryAt
    );
  }
}
