import { VehicleRealtimePort } from '@application/ports/vehicle-realtime-port';
import { SSEConnection } from './sse-connection';
import { SSESimulator } from './sse-simulator';
import { Vehicle } from '@domain/entities/vehicle';
import { VehicleId } from '@domain/entities/vehicle-id';
import { Position } from '@domain/entities/position';

/**
 * SSEConnectionFactory - Factory for creating real or simulated SSE connections.
 *
 * This factory encapsulates the decision logic for selecting between real SSE
 * connections (SSEConnection) and simulated connections (SSESimulator) based
 * on environment configuration or explicit mode selection.
 *
 * The factory provides:
 * - Type-safe creation of VehicleRealtimePort implementations
 * - Environment-based mode detection
 * - Helper functions to query current mode
 * - Convenient access to mock vehicle data for simulators
 *
 * Environment variables:
 * - VITE_DATA_MODE: 'mock' or 'api' (defaults to 'mock')
 * - When VITE_DATA_MODE='mock', SSESimulator is used
 * - When VITE_DATA_MODE='api', SSEConnection is used
 *
 * @example
 * ```typescript
 * // Create with environment-based selection
 * const port = createRealtimePort();
 *
 * // Or explicitly select mode
 * const realPort = createRealtimePort('real');
 * const simulatorPort = createRealtimePort('simulator');
 *
 * // Check current mode
 * if (isSimulatorMode()) {
 *   console.log('Running in simulator mode');
 * }
 * ```
 */

/**
 * Mode type for explicit factory selection.
 */
type RealtimeMode = 'real' | 'simulator';

/**
 * Generates mock vehicle data for use in the SSESimulator.
 *
 * Creates a deterministic array of mock vehicles centered around Madrid (40.4168, -3.7038)
 * with random positions within ±0.2 degrees. Each vehicle is generated with a fixed seed
 * to ensure reproducibility across runs.
 *
 * @returns Array of 20 mock Vehicle instances
 * @internal
 */
function createMockVehicles(): Vehicle[] {
  const vehicles: Vehicle[] = [];
  const baseLat = 40.4168; // Madrid latitude
  const baseLon = -3.7038; // Madrid longitude
  const offset = 0.2; // ~22km

  // Deterministic seeded random for reproducibility
  const seed = 12345;
  let state = seed;

  const seededRandom = (): number => {
    state = (state * 9301 + 49297) % 233280;
    return state / 233280;
  };

  for (let i = 1; i <= 20; i++) {
    const vehicleId = new VehicleId(`MOCK-VEHICLE-${String(i).padStart(2, '0')}`);
    const licensePlate = vehicleId.value; // MVP: licensePlate = vehicleId

    // Generate deterministic coordinates within offset range
    const lat = baseLat + (seededRandom() - 0.5) * offset * 2;
    const lon = baseLon + (seededRandom() - 0.5) * offset * 2;
    const position = new Position(lat, lon, new Date());

    const vehicle = new Vehicle(vehicleId, licensePlate, position);
    vehicles.push(vehicle);
  }

  return vehicles;
}

/**
 * Determines the current mode based on environment configuration.
 *
 * Uses VITE_DATA_MODE environment variable:
 * - 'mock' → 'simulator'
 * - 'api' or undefined → 'real'
 *
 * @returns Current mode ('real' or 'simulator')
 */
function getCurrentMode(): RealtimeMode {
  const dataMode = import.meta.env.VITE_DATA_MODE;

  if (dataMode === 'mock') {
    return 'simulator';
  }

  // Default to real SSE for 'api' mode or production
  return 'real';
}

/**
 * Checks if the current mode is simulator (mock data) mode.
 *
 * Useful for conditionally rendering UI elements like demo mode banners.
 *
 * @returns true if running in simulator mode, false otherwise
 * @example
 * ```typescript
 * {isSimulatorMode() && <DemoModeBanner />}
 * ```
 */
export function isSimulatorMode(): boolean {
  return getCurrentMode() === 'simulator';
}

/**
 * Checks if the current mode is real SSE mode.
 *
 * @returns true if running in real SSE mode, false otherwise
 */
export function isRealMode(): boolean {
  return getCurrentMode() === 'real';
}

/**
 * Gets the current realtime mode.
 *
 * @returns Current mode ('real' or 'simulator')
 */
export function getCurrentRealtimeMode(): RealtimeMode {
  return getCurrentMode();
}

/**
 * Creates a VehicleRealtimePort instance based on environment or explicit mode.
 *
 * This is the main factory function that determines which implementation
 * (SSEConnection or SSESimulator) to instantiate.
 *
 * Selection logic:
 * 1. If mode is explicitly provided, use that
 * 2. Otherwise, use VITE_DATA_MODE environment variable
 * 3. Default to 'real' if environment variable is not set
 *
 * @param mode - Optional explicit mode selection ('real' or 'simulator')
 * @returns VehicleRealtimePort instance (either SSEConnection or SSESimulator)
 *
 * @example
 * ```typescript
 * // Environment-based selection
 * const port = createRealtimePort();
 *
 * // Explicit real connection
 * const realPort = createRealtimePort('real');
 *
 * // Explicit simulator
 * const simPort = createRealtimePort('simulator');
 * ```
 */
export function createRealtimePort(mode?: RealtimeMode): VehicleRealtimePort {
  // Determine mode: explicit parameter takes precedence over environment
  const selectedMode = mode ?? getCurrentMode();

  if (selectedMode === 'real') {
    // Real SSE connection to backend
    return new SSEConnection();
  }

  // Simulator mode: create with mock vehicle data
  const mockVehicles = createMockVehicles();
  return new SSESimulator(mockVehicles);
}

/**
 * Exports type for explicit mode selection in component props.
 * Useful for components that need to override factory mode selection.
 */
export type { RealtimeMode };

/**
 * Exports utility for testing or debugging current factory state.
 * @internal
 */
export { getCurrentMode as __getCurrentMode, createMockVehicles as __createMockVehicles };
