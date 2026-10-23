/**
 * FakeIDGenerator
 *
 * Generates deterministic UUIDs for testing.
 * Each call returns a predictable UUID in the format: 00000000-0000-0000-0000-000000000000 + counter
 * Allows tests to verify correct UUID generation and tracking.
 */
export class FakeIDGenerator {
  private counter = 0;

  generateUUID(): string {
    this.counter++;
    const hex = this.counter.toString(16).padStart(32, '0');
    return `${hex.substring(0, 8)}-${hex.substring(8, 12)}-${hex.substring(
      12,
      16
    )}-${hex.substring(16, 20)}-${hex.substring(20, 32)}`;
  }

  /**
   * Reset counter for test isolation.
   */
  reset(): void {
    this.counter = 0;
  }

  /**
   * Get current counter value (useful for debugging tests).
   */
  getCounter(): number {
    return this.counter;
  }
}
