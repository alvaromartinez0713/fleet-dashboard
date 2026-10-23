import { TraceContext } from '../../../src/domain/models';
import { v4 as uuidv4 } from 'uuid';

/**
 * Object Mother for creating TraceContext test data.
 */
export class TraceContextMother {
  /**
   * Valid trace context with random UUIDs.
   */
  static valid(): TraceContextBuilder {
    return new TraceContextBuilder(uuidv4(), uuidv4());
  }

  /**
   * Trace context with deterministic UUIDs for predictable testing.
   */
  static deterministic(): TraceContextBuilder {
    return new TraceContextBuilder(
      '00000000-0000-0000-0000-000000000001',
      '00000000-0000-0000-0000-000000000002'
    );
  }
}

/**
 * Test Data Builder for TraceContext.
 */
export class TraceContextBuilder {
  private traceId: string;
  private idempotencyKey: string;

  constructor(traceId: string, idempotencyKey: string) {
    this.traceId = traceId;
    this.idempotencyKey = idempotencyKey;
  }

  withTraceId(traceId: string): TraceContextBuilder {
    this.traceId = traceId;
    return this;
  }

  withIdempotencyKey(idempotencyKey: string): TraceContextBuilder {
    this.idempotencyKey = idempotencyKey;
    return this;
  }

  build(): TraceContext {
    return new TraceContext(this.traceId, this.idempotencyKey);
  }
}
