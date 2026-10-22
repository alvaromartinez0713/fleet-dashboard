import { VehicleDataProvider } from '@application/ports/vehicle-data-provider';
import { Vehicle } from '@domain/entities/vehicle';
import { VehicleId } from '@domain/entities/vehicle-id';
import { Position } from '@domain/entities/position';
import { InvalidVehicleError, InvalidCoordinateError } from '@domain/errors/vehicle-errors';
import { ApiClient, VehicleLoadError } from '@infrastructure/http/api-client';

/**
 * Vehicle Data Transfer Object (DTO) - matches backend API response schema.
 * Represents the structure of vehicle data received from the API.
 */
interface VehicleDTO {
  vehicleId: string;
  licensePlate: string;
  latitude: number;
  longitude: number;
  timestamp: string; // ISO-8601 format
}

/**
 * ApiDataProvider - Adapter that fetches vehicle data from the backend API.
 *
 * Implements the VehicleDataProvider port.
 *
 * Responsibilities:
 * - Fetch vehicle data from the backend API (GET /api/vehicles)
 * - Validate response data structure and values
 * - Transform API DTOs into domain Vehicle entities
 * - Handle network errors and invalid responses gracefully
 *
 * The adapter validates all incoming data before constructing domain entities
 * to ensure invariants are maintained.
 *
 * Usage:
 * ```typescript
 * const provider = new ApiDataProvider();
 * const vehicles = await provider.loadVehicles();
 * ```
 *
 * Error handling:
 * - Network errors are caught and re-thrown as VehicleLoadError
 * - Invalid response data is caught during entity construction
 * - Individual vehicle construction failures are caught and reported
 */
export class ApiDataProvider implements VehicleDataProvider {
  private apiClient: ApiClient;

  /**
   * Creates a new ApiDataProvider instance.
   *
   * @param apiClient - HTTP client for making API requests. If not provided,
   *                    a new ApiClient with default configuration is created.
   */
  constructor(apiClient?: ApiClient) {
    this.apiClient = apiClient || new ApiClient();
  }

  /**
   * Loads all vehicles from the backend API.
   *
   * Fetches vehicle data from GET /api/vehicles, validates the response,
   * and constructs Vehicle domain entities from the API data.
   *
   * @returns Promise that resolves to an array of Vehicle entities
   * @throws VehicleLoadError if API request fails or response is invalid
   * @throws Error if API response violates the expected schema
   *
   * Process:
   * 1. Fetch vehicle data from the API
   * 2. Validate response is an array
   * 3. For each vehicle DTO:
   *    - Validate required fields and data types
   *    - Validate coordinate ranges (latitude: [-90, 90], longitude: [-180, 180])
   *    - Parse ISO-8601 timestamp
   *    - Construct Vehicle entity (which performs invariant validation)
   * 4. Return array of valid Vehicle entities
   * 5. If any step fails, throw appropriate error
   */
  async loadVehicles(): Promise<Vehicle[]> {
    try {
      // Fetch vehicle data from API
      const dtos = await this.apiClient.get<VehicleDTO[]>('/api/vehicles');

      // Validate response is an array
      if (!Array.isArray(dtos)) {
        throw new VehicleLoadError('API response must be an array of vehicles');
      }

      // Transform DTOs to Vehicle entities
      const vehicles: Vehicle[] = [];
      for (const dto of dtos) {
        try {
          const vehicle = this.dtoToEntity(dto);
          vehicles.push(vehicle);
        } catch (error) {
          // Silently skip invalid vehicles and continue processing others
          continue;
        }
      }

      return vehicles;
    } catch (error) {
      // If it's already a VehicleLoadError, re-throw it
      if (error instanceof VehicleLoadError) {
        throw error;
      }

      // If it's an error from domain validation, wrap it
      if (
        error instanceof InvalidVehicleError ||
        error instanceof InvalidCoordinateError
      ) {
        throw new VehicleLoadError(`Invalid vehicle data from API: ${error.message}`);
      }

      // Wrap any other error
      throw new VehicleLoadError(
        `Failed to load vehicles from API: ${error instanceof Error ? error.message : String(error)}`,
      );
    }
  }

  /**
   * Transforms a VehicleDTO into a Vehicle domain entity.
   *
   * Validates:
   * - vehicleId is a non-empty string
   * - licensePlate is a non-empty string
   * - latitude is a number within [-90, 90]
   * - longitude is a number within [-180, 180]
   * - timestamp is a valid ISO-8601 string that can be parsed
   *
   * @param dto - The vehicle data from the API
   * @returns Vehicle entity constructed from the DTO
   * @throws InvalidVehicleIdError if vehicleId is invalid
   * @throws InvalidCoordinateError if coordinates are out of range
   * @throws InvalidVehicleError if licensePlate is missing or any field is invalid
   * @throws Error if timestamp cannot be parsed
   *
   * The method validates data before passing to domain constructors.
   * This ensures that invalid data is caught early with meaningful error messages.
   */
  private dtoToEntity(dto: VehicleDTO): Vehicle {
    // Validate vehicleId
    if (!dto.vehicleId || typeof dto.vehicleId !== 'string') {
      throw new InvalidVehicleError('vehicleId must be a non-empty string');
    }

    // Validate licensePlate
    if (!dto.licensePlate || typeof dto.licensePlate !== 'string') {
      throw new InvalidVehicleError('licensePlate must be a non-empty string');
    }

    // Validate coordinates are numbers
    if (typeof dto.latitude !== 'number' || typeof dto.longitude !== 'number') {
      throw new InvalidCoordinateError(
        'latitude and longitude must be numbers',
      );
    }

    // Validate coordinate ranges
    if (dto.latitude < -90 || dto.latitude > 90) {
      throw new InvalidCoordinateError(
        `Latitude must be between -90 and 90, got ${dto.latitude}`,
      );
    }

    if (dto.longitude < -180 || dto.longitude > 180) {
      throw new InvalidCoordinateError(
        `Longitude must be between -180 and 180, got ${dto.longitude}`,
      );
    }

    // Parse timestamp
    if (!dto.timestamp || typeof dto.timestamp !== 'string') {
      throw new InvalidVehicleError('timestamp must be a non-empty ISO-8601 string');
    }

    const timestamp = new Date(dto.timestamp);
    if (isNaN(timestamp.getTime())) {
      throw new InvalidVehicleError(
        `timestamp must be a valid ISO-8601 date string, got ${dto.timestamp}`,
      );
    }

    // Construct domain entities - this will also validate invariants
    const vehicleId = new VehicleId(dto.vehicleId);
    const position = new Position(dto.latitude, dto.longitude, timestamp);

    return new Vehicle(vehicleId, dto.licensePlate, position);
  }
}
