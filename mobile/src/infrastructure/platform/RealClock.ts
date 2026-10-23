import { Clock } from '../ports/Clock';

/**
 * RealClock
 *
 * Production implementation of Clock using system time.
 */
export class RealClock implements Clock {
  now(): Date {
    return new Date();
  }
}
