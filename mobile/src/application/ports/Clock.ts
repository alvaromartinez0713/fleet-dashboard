/**
 * Clock
 *
 * Abstracts time access to enable deterministic testing.
 *
 * Instead of directly calling Date.now() or new Date(), the application
 * layer receives a Clock instance through dependency injection. This allows
 * tests to provide a fake clock with controlled time, making test data
 * reproducible and eliminating timing-based test flakiness.
 *
 * This interface supports the Testing standard (section 6: Aislar el acoplamiento
 * inevitable) by:
 * 1. Providing a single, injectable source of time
 * 2. Enabling fake implementations with fixed or controlled timestamps
 * 3. Allowing tests to verify time-dependent behavior without relying on real time passage
 *
 * Concrete implementations include:
 * - RealClock: Returns current system time (Date.now())
 * - FakeClock: Returns controlled/fixed time for deterministic testing
 *
 * Usage pattern:
 * - Infrastructure layer composes RealClock in production, FakeClock in tests
 * - Application layer receives clock through constructor injection
 * - Use cases call clock.now() instead of Date.now() or new Date()
 * - Domain models and tests use injected clock to create reproducible timestamps
 */

/**
 * Port interface for time access
 *
 * Provides the current date and time through a controlled injection point,
 * enabling both real time queries in production and deterministic time
 * control in tests.
 *
 * @example
 * // In production: use RealClock
 * const clock = new RealClock();
 * const now = clock.now(); // Returns current time
 *
 * @example
 * // In tests: use FakeClock
 * const fixedDate = new Date('2024-01-15T10:30:00Z');
 * const clock = new FakeClock(fixedDate);
 * const now = clock.now(); // Always returns 2024-01-15T10:30:00Z
 */
export interface Clock {
  /**
   * Get the current date and time
   *
   * Returns a Date object representing the current moment in time.
   * The exact behavior depends on the implementation:
   * - RealClock: Returns the current system time
   * - FakeClock: Returns a fixed or controlled time (for testing)
   *
   * @returns Date object representing the current date and time
   *
   * @example
   * const now = clock.now();
   * const timestamp = now.getTime(); // Milliseconds since epoch
   * const iso = now.toISOString(); // ISO 8601 string
   */
  now(): Date;
}
