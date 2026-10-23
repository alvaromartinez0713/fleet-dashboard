import { IClock } from '../../../src/application/ports/IClock';

/**
 * FakeClock
 *
 * Injectable clock for deterministic testing.
 * Allows manual control of time for testing backoff and retry logic.
 */
export class FakeClock implements IClock {
  private currentTime: Date;

  constructor(initialTime: Date = new Date('2024-01-01T00:00:00Z')) {
    this.currentTime = new Date(initialTime);
  }

  now(): Date {
    return new Date(this.currentTime);
  }

  /**
   * Advance time by specified milliseconds.
   * Useful for testing retry backoff and timeout scenarios.
   */
  advance(ms: number): void {
    this.currentTime = new Date(this.currentTime.getTime() + ms);
  }

  /**
   * Set time to a specific instant.
   */
  setTime(time: Date): void {
    this.currentTime = new Date(time);
  }
}
