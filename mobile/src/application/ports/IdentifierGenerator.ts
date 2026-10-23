/**
 * Port interface for UUID generation.
 *
 * IDGenerator abstracts UUID generation to support deterministic testing.
 * By injecting a configurable ID generator, tests can provide predictable UUIDs
 * instead of relying on random generation, enabling:
 *
 * 1. Reproducible test data: Tests can verify exact event IDs, trace IDs, and idempotency keys
 * 2. Correlation across test layers: Trace IDs remain consistent from UI hooks through use cases to domain entities
 * 3. Predictable test assertions: Test assertions don't need to use regex or partial matching
 * 4. Deterministic snapshots: Test data remains stable across runs for easier debugging
 *
 * Concrete implementations include:
 * - RealIDGenerator: Uses the uuid library to generate random UUIDs (v4)
 * - FakeIDGenerator: Returns predictable UUIDs in sequence (for testing)
 *
 * This interface supports:
 * - RF-01 (Captura móvil) by ensuring events can be uniquely identified for tracking
 * - RNF-04 (Idempotencia) by providing stable idempotency keys in tests
 * - Testing infrastructure by enabling deterministic test data generation
 *
 * Implementation details:
 * - Implementations should return valid UUID v4 format strings (36 characters including hyphens)
 * - Implementations should be thread-safe if called from concurrent contexts
 * - Test implementations may return sequential IDs (e.g., "00000000-0000-0000-0000-000000000001")
 */
export interface IDGenerator {
  /**
   * Generate a new UUID string.
   *
   * Produces a unique identifier that can be used for:
   * - Event IDs (identify a specific position capture)
   * - Trace IDs (correlate a request across system boundaries)
   * - Idempotency keys (prevent duplicate processing of retried requests)
   * - Vehicle IDs or other domain entities
   *
   * @returns A UUID string in standard format (e.g., "550e8400-e29b-41d4-a716-446655440000")
   *
   * @example
   * // In production: generates random UUIDs
   * const eventId = idGenerator.generateUUID();  // "a1b2c3d4-e5f6-47g8-h9i0-j1k2l3m4n5o6"
   *
   * @example
   * // In tests: generates predictable UUIDs
   * const eventId = idGenerator.generateUUID();  // "00000000-0000-0000-0000-000000000001"
   * const traceId = idGenerator.generateUUID();  // "00000000-0000-0000-0000-000000000002"
   *
   * @example
   * // Used in domain entities
   * const event = PositionReceivedEvent.create({
   *   eventId: EventId.of(idGenerator.generateUUID()),
   *   vehicleId: VehicleId.of("VEHICLE-001"),
   *   coordinate: coordinates,
   *   capturedAt: clock.now(),
   *   traceContext: TraceContext.create(
   *     TraceId.of(idGenerator.generateUUID()),
   *     IdempotencyKey.of(idGenerator.generateUUID())
   *   ),
   * });
   */
  generateUUID(): string;
}
