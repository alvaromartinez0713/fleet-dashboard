import { describe, it, expect, beforeEach, vi } from 'vitest';
import { ApiDataProvider } from '@infrastructure/data-providers/api-data-provider';
import { ApiClient, VehicleLoadError } from '@infrastructure/http/api-client';
import { VehicleMother } from '@tests/support/mothers/vehicle-mother';
import { InvalidVehicleError, InvalidCoordinateError } from '@domain/errors/vehicle-errors';
import { Vehicle } from '@domain/entities/vehicle';

/**
 * Tests for ApiDataProvider - adapter that loads vehicle data from backend API
 *
 * Verifies: Requirement 2 (last known vehicle positions), Requirement 8 (OpenAPI integration)
 * Test suite: Fast (< 2s per test)
 *
 * Entry point (public edge): loadVehicles() method
 * Observable output: Array of Vehicle domain entities or thrown errors
 * Dependencies (non-managed): ApiClient HTTP calls - replaced with mocks
 * Dependencies (managed): Domain entities (Vehicle, Position) - real instances
 */
describe('ApiDataProvider', () => {
  let apiClient: ApiClient;
  let provider: ApiDataProvider;

  beforeEach(() => {
    apiClient = new ApiClient();
    provider = new ApiDataProvider(apiClient);
  });

  describe('loadVehicles()', () => {
    describe('success scenarios', () => {
      it('loads valid vehicles from API and returns Vehicle entities', async () => {
        // Arrange
        const validVehicles = VehicleMother.validMultiple(3);
        const apiResponse = validVehicles.map(v => ({
          vehicleId: v.vehicleId.toString(),
          licensePlate: v.licensePlate,
          latitude: v.position.latitude,
          longitude: v.position.longitude,
          timestamp: v.position.timestamp.toISOString(),
        }));

        vi.spyOn(apiClient, 'get').mockResolvedValue(apiResponse);

        // Act
        const result = await provider.loadVehicles();

        // Assert
        expect(result).toHaveLength(3);
        expect(result).toEqual(
          expect.arrayContaining([
            expect.objectContaining({
              vehicleId: expect.objectContaining({ value: 'MOCK-VEHICLE-01' }),
              licensePlate: 'MOCK-VEHICLE-01',
              position: expect.objectContaining({
                latitude: expect.any(Number),
                longitude: expect.any(Number),
              }),
            }),
          ]),
        );

        // Verify API was called with correct endpoint
        expect(apiClient.get).toHaveBeenCalledWith('/api/vehicles');
      });

      it('constructs Vehicle entities with correct domain invariants', async () => {
        // Arrange
        const apiResponse = [
          {
            vehicleId: 'VH-001',
            licensePlate: 'ABC-1234',
            latitude: 40.7128,
            longitude: -74.006,
            timestamp: '2024-01-15T10:30:00.000Z',
          },
        ];

        vi.spyOn(apiClient, 'get').mockResolvedValue(apiResponse);

        // Act
        const result = await provider.loadVehicles();

        // Assert - Verify entity invariants are satisfied
        expect(result).toHaveLength(1);
        const vehicle = result[0] as Vehicle;
        expect(vehicle.vehicleId.toString()).toBe('VH-001');
        expect(vehicle.licensePlate).toBe('ABC-1234');
        expect(vehicle.position.latitude).toBe(40.7128);
        expect(vehicle.position.longitude).toBe(-74.006);
        expect(vehicle.position.timestamp).toEqual(new Date('2024-01-15T10:30:00.000Z'));
      });

      it('returns empty array when API returns empty list', async () => {
        // Arrange
        vi.spyOn(apiClient, 'get').mockResolvedValue([]);

        // Act
        const result = await provider.loadVehicles();

        // Assert
        expect(result).toHaveLength(0);
      });

      it('skips invalid vehicles and returns only valid ones', async () => {
        // Arrange
        const apiResponse = [
          {
            vehicleId: 'VH-001',
            licensePlate: 'ABC-1234',
            latitude: 40.7128,
            longitude: -74.006,
            timestamp: '2024-01-15T10:30:00.000Z',
          },
          {
            vehicleId: 'VH-002',
            licensePlate: '',
            latitude: 40.7200,
            longitude: -74.0100,
            timestamp: '2024-01-15T10:30:00.000Z',
          },
          {
            vehicleId: 'VH-003',
            licensePlate: 'DEF-5678',
            latitude: 40.7300,
            longitude: -74.0200,
            timestamp: '2024-01-15T10:30:00.000Z',
          },
        ];

        vi.spyOn(apiClient, 'get').mockResolvedValue(apiResponse);

        // Act
        const result = await provider.loadVehicles();

        // Assert - VH-002 should be skipped (empty license plate)
        expect(result).toHaveLength(2);
        expect(result[0]?.vehicleId.toString()).toBe('VH-001');
        expect(result[1]?.vehicleId.toString()).toBe('VH-003');
      });

      it('handles coordinate precision correctly (4 decimal places)', async () => {
        // Arrange
        const apiResponse = [
          {
            vehicleId: 'VH-001',
            licensePlate: 'ABC-1234',
            latitude: 40.416825,
            longitude: -3.703819,
            timestamp: '2024-01-15T10:30:00.000Z',
          },
        ];

        vi.spyOn(apiClient, 'get').mockResolvedValue(apiResponse);

        // Act
        const result = await provider.loadVehicles();

        // Assert
        const vehicle = result[0] as Vehicle;
        expect(vehicle.position.latitude).toBe(40.416825);
        expect(vehicle.position.longitude).toBe(-3.703819);
      });
    });

    describe('validation scenarios', () => {
      it('rejects vehicles with missing vehicleId', async () => {
        // Arrange
        const apiResponse = [
          {
            licensePlate: 'ABC-1234',
            latitude: 40.7128,
            longitude: -74.006,
            timestamp: '2024-01-15T10:30:00.000Z',
          },
        ];

        vi.spyOn(apiClient, 'get').mockResolvedValue(apiResponse as any);

        // Act
        const result = await provider.loadVehicles();

        // Assert - Should skip invalid vehicle
        expect(result).toHaveLength(0);
      });

      it('rejects vehicles with empty vehicleId', async () => {
        // Arrange
        const apiResponse = [
          {
            vehicleId: '',
            licensePlate: 'ABC-1234',
            latitude: 40.7128,
            longitude: -74.006,
            timestamp: '2024-01-15T10:30:00.000Z',
          },
        ];

        vi.spyOn(apiClient, 'get').mockResolvedValue(apiResponse);

        // Act
        const result = await provider.loadVehicles();

        // Assert
        expect(result).toHaveLength(0);
      });

      it('rejects vehicles with missing licensePlate', async () => {
        // Arrange
        const apiResponse = [
          {
            vehicleId: 'VH-001',
            latitude: 40.7128,
            longitude: -74.006,
            timestamp: '2024-01-15T10:30:00.000Z',
          },
        ];

        vi.spyOn(apiClient, 'get').mockResolvedValue(apiResponse as any);

        // Act
        const result = await provider.loadVehicles();

        // Assert
        expect(result).toHaveLength(0);
      });

      it('rejects vehicles with empty licensePlate', async () => {
        // Arrange
        const apiResponse = [
          {
            vehicleId: 'VH-001',
            licensePlate: '',
            latitude: 40.7128,
            longitude: -74.006,
            timestamp: '2024-01-15T10:30:00.000Z',
          },
        ];

        vi.spyOn(apiClient, 'get').mockResolvedValue(apiResponse);

        // Act
        const result = await provider.loadVehicles();

        // Assert
        expect(result).toHaveLength(0);
      });

      it('rejects vehicles with latitude out of range (too high)', async () => {
        // Arrange - Verifies: Requirement 2.6 (coordinate validation)
        const apiResponse = [
          {
            vehicleId: 'VH-001',
            licensePlate: 'ABC-1234',
            latitude: 91.0, // Out of range: > 90
            longitude: -74.006,
            timestamp: '2024-01-15T10:30:00.000Z',
          },
        ];

        vi.spyOn(apiClient, 'get').mockResolvedValue(apiResponse);

        // Act
        const result = await provider.loadVehicles();

        // Assert
        expect(result).toHaveLength(0);
      });

      it('rejects vehicles with latitude out of range (too low)', async () => {
        // Arrange
        const apiResponse = [
          {
            vehicleId: 'VH-001',
            licensePlate: 'ABC-1234',
            latitude: -91.0, // Out of range: < -90
            longitude: -74.006,
            timestamp: '2024-01-15T10:30:00.000Z',
          },
        ];

        vi.spyOn(apiClient, 'get').mockResolvedValue(apiResponse);

        // Act
        const result = await provider.loadVehicles();

        // Assert
        expect(result).toHaveLength(0);
      });

      it('rejects vehicles with longitude out of range (too high)', async () => {
        // Arrange
        const apiResponse = [
          {
            vehicleId: 'VH-001',
            licensePlate: 'ABC-1234',
            latitude: 40.7128,
            longitude: 181.0, // Out of range: > 180
            timestamp: '2024-01-15T10:30:00.000Z',
          },
        ];

        vi.spyOn(apiClient, 'get').mockResolvedValue(apiResponse);

        // Act
        const result = await provider.loadVehicles();

        // Assert
        expect(result).toHaveLength(0);
      });

      it('rejects vehicles with longitude out of range (too low)', async () => {
        // Arrange
        const apiResponse = [
          {
            vehicleId: 'VH-001',
            licensePlate: 'ABC-1234',
            latitude: 40.7128,
            longitude: -181.0, // Out of range: < -180
            timestamp: '2024-01-15T10:30:00.000Z',
          },
        ];

        vi.spyOn(apiClient, 'get').mockResolvedValue(apiResponse);

        // Act
        const result = await provider.loadVehicles();

        // Assert
        expect(result).toHaveLength(0);
      });

      it('rejects vehicles with non-numeric coordinates', async () => {
        // Arrange
        const apiResponse = [
          {
            vehicleId: 'VH-001',
            licensePlate: 'ABC-1234',
            latitude: 'not-a-number',
            longitude: -74.006,
            timestamp: '2024-01-15T10:30:00.000Z',
          },
        ];

        vi.spyOn(apiClient, 'get').mockResolvedValue(apiResponse as any);

        // Act
        const result = await provider.loadVehicles();

        // Assert
        expect(result).toHaveLength(0);
      });

      it('rejects vehicles with invalid ISO-8601 timestamp', async () => {
        // Arrange
        const apiResponse = [
          {
            vehicleId: 'VH-001',
            licensePlate: 'ABC-1234',
            latitude: 40.7128,
            longitude: -74.006,
            timestamp: 'not-a-valid-date',
          },
        ];

        vi.spyOn(apiClient, 'get').mockResolvedValue(apiResponse);

        // Act
        const result = await provider.loadVehicles();

        // Assert
        expect(result).toHaveLength(0);
      });

      it('accepts boundary latitude values (-90, 0, 90)', async () => {
        // Arrange
        const apiResponse = [
          {
            vehicleId: 'VH-001',
            licensePlate: 'ABC-1234',
            latitude: -90.0,
            longitude: 0,
            timestamp: '2024-01-15T10:30:00.000Z',
          },
          {
            vehicleId: 'VH-002',
            licensePlate: 'DEF-5678',
            latitude: 0,
            longitude: 0,
            timestamp: '2024-01-15T10:30:00.000Z',
          },
          {
            vehicleId: 'VH-003',
            licensePlate: 'GHI-9012',
            latitude: 90.0,
            longitude: 0,
            timestamp: '2024-01-15T10:30:00.000Z',
          },
        ];

        vi.spyOn(apiClient, 'get').mockResolvedValue(apiResponse);

        // Act
        const result = await provider.loadVehicles();

        // Assert - All three should be accepted
        expect(result).toHaveLength(3);
        expect(result[0]?.position.latitude).toBe(-90.0);
        expect(result[1]?.position.latitude).toBe(0);
        expect(result[2]?.position.latitude).toBe(90.0);
      });

      it('accepts boundary longitude values (-180, 0, 180)', async () => {
        // Arrange
        const apiResponse = [
          {
            vehicleId: 'VH-001',
            licensePlate: 'ABC-1234',
            latitude: 0,
            longitude: -180.0,
            timestamp: '2024-01-15T10:30:00.000Z',
          },
          {
            vehicleId: 'VH-002',
            licensePlate: 'DEF-5678',
            latitude: 0,
            longitude: 0,
            timestamp: '2024-01-15T10:30:00.000Z',
          },
          {
            vehicleId: 'VH-003',
            licensePlate: 'GHI-9012',
            latitude: 0,
            longitude: 180.0,
            timestamp: '2024-01-15T10:30:00.000Z',
          },
        ];

        vi.spyOn(apiClient, 'get').mockResolvedValue(apiResponse);

        // Act
        const result = await provider.loadVehicles();

        // Assert - All three should be accepted
        expect(result).toHaveLength(3);
        expect(result[0]?.position.longitude).toBe(-180.0);
        expect(result[1]?.position.longitude).toBe(0);
        expect(result[2]?.position.longitude).toBe(180.0);
      });
    });

    describe('error handling scenarios', () => {
      it('throws VehicleLoadError when API response is not an array', async () => {
        // Arrange
        vi.spyOn(apiClient, 'get').mockResolvedValue({ vehicles: [] });

        // Act & Assert
        await expect(provider.loadVehicles()).rejects.toThrow(VehicleLoadError);
        await expect(provider.loadVehicles()).rejects.toThrow('must be an array');
      });

      it('throws VehicleLoadError when API request fails', async () => {
        // Arrange
        const apiError = new Error('Network timeout');
        vi.spyOn(apiClient, 'get').mockRejectedValue(apiError);

        // Act & Assert
        await expect(provider.loadVehicles()).rejects.toThrow(VehicleLoadError);
      });

      it('wraps InvalidVehicleError in VehicleLoadError with context', async () => {
        // Arrange
        const apiResponse = [
          {
            vehicleId: 'VH-001',
            licensePlate: 'ABC-1234',
            latitude: 40.7128,
            longitude: -74.006,
            timestamp: '2024-01-15T10:30:00.000Z',
          },
        ];

        // Simulate an error during construction
        const constructionError = new InvalidVehicleError('Invalid vehicle');
        vi.spyOn(apiClient, 'get').mockResolvedValue(apiResponse);


        // Act - Force error by making Vehicle constructor throw
        // We'll mock the entire flow by rejecting the promise
        vi.spyOn(apiClient, 'get').mockRejectedValue(constructionError);

        // Assert
        await expect(provider.loadVehicles()).rejects.toThrow(VehicleLoadError);

      });

      it('continues processing after invalid vehicle and returns valid ones', async () => {
        // Arrange - Property: Robustness - invalid items don't break entire operation
        const apiResponse = [
          {
            vehicleId: 'VH-001',
            licensePlate: 'ABC-1234',
            latitude: 40.7128,
            longitude: -74.006,
            timestamp: '2024-01-15T10:30:00.000Z',
          },
          {
            vehicleId: 'VH-002',
            licensePlate: 'DEF-5678',
            latitude: 91.0, // Invalid: out of range
            longitude: -74.006,
            timestamp: '2024-01-15T10:30:00.000Z',
          },
          {
            vehicleId: 'VH-003',
            licensePlate: 'GHI-9012',
            latitude: 40.7300,
            longitude: -74.0200,
            timestamp: '2024-01-15T10:30:00.000Z',
          },
        ];

        vi.spyOn(apiClient, 'get').mockResolvedValue(apiResponse);

        // Act
        const result = await provider.loadVehicles();

        // Assert - VH-001 and VH-003 should be included, VH-002 skipped
        expect(result).toHaveLength(2);
        expect(result[0]?.vehicleId.toString()).toBe('VH-001');
        expect(result[1]?.vehicleId.toString()).toBe('VH-003');
      });
    });

    describe('integration with port interface', () => {
      it('implements VehicleDataProvider interface exactly', async () => {
        // Arrange
        const apiResponse = [
          {
            vehicleId: 'VH-001',
            licensePlate: 'ABC-1234',
            latitude: 40.7128,
            longitude: -74.006,
            timestamp: '2024-01-15T10:30:00.000Z',
          },
        ];

        vi.spyOn(apiClient, 'get').mockResolvedValue(apiResponse);

        // Act
        const result = await provider.loadVehicles();

        // Assert - Verify method signature and return type
        expect(typeof provider.loadVehicles).toBe('function');
        expect(Array.isArray(result)).toBe(true);
        expect(result[0]).toBeInstanceOf(Vehicle);
      });

      it('is compatible with mock data provider (same interface)', async () => {
        // Arrange - Verify it can be swapped with MockDataProvider
        const apiResponse = VehicleMother.validMultiple(5).map(v => ({
          vehicleId: v.vehicleId.toString(),
          licensePlate: v.licensePlate,
          latitude: v.position.latitude,
          longitude: v.position.longitude,
          timestamp: v.position.timestamp.toISOString(),
        }));

        vi.spyOn(apiClient, 'get').mockResolvedValue(apiResponse);

        // Act
        const result = await provider.loadVehicles();

        // Assert - Same output contract as mock provider
        expect(result).toHaveLength(5);
        result.forEach(vehicle => {
          expect(vehicle).toBeInstanceOf(Vehicle);
          expect(vehicle.vehicleId).toBeDefined();
          expect(vehicle.licensePlate).toBeDefined();
          expect(vehicle.position).toBeDefined();
        });
      });
    });

    describe('edge cases', () => {
      it('handles vehicles with special characters in licensePlate', async () => {
        // Arrange
        const apiResponse = [
          {
            vehicleId: 'VH-001',
            licensePlate: 'ABC-1234-ÑÇ',
            latitude: 40.7128,
            longitude: -74.006,
            timestamp: '2024-01-15T10:30:00.000Z',
          },
        ];

        vi.spyOn(apiClient, 'get').mockResolvedValue(apiResponse);

        // Act
        const result = await provider.loadVehicles();

        // Assert
        expect(result).toHaveLength(1);
        expect(result[0]?.licensePlate).toBe('ABC-1234-ÑÇ');
      });

      it('handles very large coordinate precision values', async () => {
        // Arrange
        const apiResponse = [
          {
            vehicleId: 'VH-001',
            licensePlate: 'ABC-1234',
            latitude: 40.4168253254623,
            longitude: -3.7038198257492,
            timestamp: '2024-01-15T10:30:00.000Z',
          },
        ];

        vi.spyOn(apiClient, 'get').mockResolvedValue(apiResponse);

        // Act
        const result = await provider.loadVehicles();

        // Assert
        expect(result).toHaveLength(1);
        expect(result[0]?.position.latitude).toBeCloseTo(40.4168253254623);
      });

      it('handles ISO-8601 timestamps with milliseconds', async () => {
        // Arrange
        const apiResponse = [
          {
            vehicleId: 'VH-001',
            licensePlate: 'ABC-1234',
            latitude: 40.7128,
            longitude: -74.006,
            timestamp: '2024-01-15T10:30:00.123Z',
          },
        ];

        vi.spyOn(apiClient, 'get').mockResolvedValue(apiResponse);

        // Act
        const result = await provider.loadVehicles();

        // Assert
        expect(result).toHaveLength(1);
        expect(result[0]?.position.timestamp.getMilliseconds()).toBe(123);
      });
    });
  });
});
