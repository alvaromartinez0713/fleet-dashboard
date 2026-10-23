import { CapturePositionUseCase } from './CapturePositionUseCase';

/**
 * StartCaptureUseCase
 *
 * Starts periodic GPS capture every 30 seconds.
 * Each capture is performed via CapturePositionUseCase.
 * Errors during capture are logged but don't break the interval.
 *
 * Flow:
 * 1. Validate vehicleId is not empty
 * 2. Check if capture is already running (prevent duplicate intervals)
 * 3. Set up interval timer (default 30000ms = 30 seconds)
 * 4. Store interval ID for later cancellation via StopCaptureUseCase
 * 5. Handle errors gracefully - log but don't crash the interval
 *
 * Requirements covered:
 * - RF-01: Automatic capture of coordinates every 30 seconds
 * - RNF-01: Clean Architecture
 * - RNF-06: Resiliencia en operaciones
 */
export class StartCaptureUseCase {
  private intervalId: NodeJS.Timeout | undefined;

  constructor(private capturePositionUseCase: CapturePositionUseCase) {}

  /**
   * Start periodic GPS capture.
   *
   * @param vehicleId The vehicle ID to associate with captures
   * @param intervalMs Optional interval in milliseconds (default: 30000ms = 30 seconds)
   * @returns Promise that resolves when interval is set up
   * @throws Error if vehicleId is empty
   */
  async execute(vehicleId: string, intervalMs: number = 30000): Promise<void> {
    // 1. Validate vehicleId
    if (!vehicleId || vehicleId.trim().length === 0) {
      throw new Error('Vehicle ID is required and must not be empty');
    }

    // 2. Validate interval
    if (intervalMs <= 0) {
      throw new Error('Interval must be greater than 0');
    }

    // 3. Check if already running
    if (this.intervalId !== undefined) {
      console.warn('GPS capture already running. Call stop() first to restart.');
      return;
    }

    // 4. Set up interval timer
    this.intervalId = setInterval(async () => {
      try {
        // 5. Call CapturePositionUseCase for each interval tick
        await this.capturePositionUseCase.execute(vehicleId);
      } catch (error) {
        // 6. Handle errors gracefully - log but don't crash interval
        console.error(
          'Error capturing position:',
          error instanceof Error ? error.message : String(error)
        );
        // The interval continues running, allowing recovery from transient errors
      }
    }, intervalMs);
  }

  /**
   * Stop periodic GPS capture.
   * Clears the interval timer and releases resources.
   */
  stop(): void {
    if (this.intervalId !== undefined) {
      clearInterval(this.intervalId);
      this.intervalId = undefined;
    }
  }

  /**
   * Check if capture is currently running.
   *
   * @returns true if interval is active, false otherwise
   */
  isRunning(): boolean {
    return this.intervalId !== undefined;
  }
}
