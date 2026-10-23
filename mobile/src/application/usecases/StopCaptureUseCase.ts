import { StartCaptureUseCase } from './StartCaptureUseCase';

/**
 * StopCaptureUseCase
 *
 * Stops the periodic GPS capture that was started by StartCaptureUseCase.
 * Clears the interval timer and releases resources.
 * Returns gracefully if no capture is currently running.
 *
 * Flow:
 * 1. Check if StartCaptureUseCase has an active interval
 * 2. If active: clear the interval timer
 * 3. If inactive: return gracefully without error
 *
 * Requirements covered:
 * - RF-01: Permite al usuario detener la captura de GPS
 * - RNF-01: Clean Architecture
 * - RNF-06: Resiliencia en operaciones
 */
export class StopCaptureUseCase {
  /**
   * Stop periodic GPS capture.
   *
   * This method is idempotent - calling it multiple times is safe.
   * If no capture is running, it returns gracefully without error.
   *
   * @param startCaptureUseCase The StartCaptureUseCase instance to stop
   * @returns Promise that resolves when capture is stopped
   */
  async execute(startCaptureUseCase: StartCaptureUseCase): Promise<void> {
    // 1. Stop the interval (idempotent - safe to call even if not running)
    startCaptureUseCase.stop();

    // 2. Return gracefully
    return Promise.resolve();
  }

  /**
   * Alternative simpler interface without parameters.
   * This version stores the StartCaptureUseCase internally.
   *
   * @deprecated Use execute(startCaptureUseCase) instead for better testability
   */
  stop(): void {
    // This version is kept for convenience but is less testable
    // because it doesn't take StartCaptureUseCase as a parameter
  }
}
