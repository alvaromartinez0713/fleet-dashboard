import { IClock } from '../../../src/application/ports/IClock';

export class FakeClock implements IClock {
  private currentTime: Date;

  constructor(time: Date = new Date('2024-01-01T00:00:00Z')) {
    this.currentTime = new Date(time);
  }

  now(): Date {
    return new Date(this.currentTime);
  }

  advance(ms: number): void {
    this.currentTime = new Date(this.currentTime.getTime() + ms);
  }

  setTime(time: Date): void {
    this.currentTime = new Date(time);
  }
}
