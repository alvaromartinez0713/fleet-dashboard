/**
 * TraceContext is an immutable value object containing trace and idempotency information.
 * Used to correlate related events and prevent duplicate processing in the backend.
 */
export class TraceContext {
  readonly traceId: string; // UUID
  readonly idempotencyKey: string; // UUID

  constructor(traceId: string, idempotencyKey: string) {
    if (!traceId || typeof traceId !== 'string') {
      throw new Error('traceId must be a non-empty string');
    }
    if (!idempotencyKey || typeof idempotencyKey !== 'string') {
      throw new Error('idempotencyKey must be a non-empty string');
    }

    this.traceId = traceId;
    this.idempotencyKey = idempotencyKey;
  }

  /**
   * Value object equality: two TraceContexts are equal if their IDs match.
   */
  equals(other: TraceContext): boolean {
    return (
      this.traceId === other.traceId &&
      this.idempotencyKey === other.idempotencyKey
    );
  }

  /**
   * String representation for logging.
   */
  toString(): string {
    return `TraceContext(traceId=${this.traceId}, idempotencyKey=${this.idempotencyKey})`;
  }
}
