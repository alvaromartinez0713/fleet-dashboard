import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { Position } from '@domain/entities/position';
import { InvalidCoordinateError } from '@domain/errors/vehicle-errors';

/**
 * Test suite para Position - Objeto de valor que representa ubicación geográfica
 * 
 * Verifies:
 * - Requisito 2.2: Validar latitud [-90, 90] y longitud [-180, 180]
 * - Requisito 2.6: Rechazar coordenadas fuera de rango
 * - Property 3 (precisión preservada): equals() con tolerancia 1e-6
 */
describe('Position - Value Object', () => {
  let mockNow: number;

  beforeEach(() => {
    // Fijar timestamp para pruebas deterministas
    mockNow = new Date('2024-01-15T10:30:00Z').getTime();
    vi.useFakeTimers();
    vi.setSystemTime(mockNow);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  // ============================================================================
  // Constructor y validación
  // ============================================================================

  describe('Constructor validation', () => {
    it('creates a Position with valid coordinates', () => {
      // Verifies: Requisito 2.2
      const position = new Position(40.4168, -3.7038);

      expect(position.latitude).toBe(40.4168);
      expect(position.longitude).toBe(-3.7038);
      expect(position.timestamp).toEqual(new Date(mockNow));
    });

    it('creates a Position with custom timestamp', () => {
      const customTime = new Date('2024-01-15T09:00:00Z');
      const position = new Position(40.4168, -3.7038, customTime);

      expect(position.timestamp).toEqual(customTime);
    });

    it('rejects latitude < -90', () => {
      // Verifies: Requisito 2.6, Requisito 7.6
      expect(() => new Position(-90.1, 0)).toThrow(InvalidCoordinateError);
      expect(() => new Position(-90.1, 0)).toThrow(/Latitude must be between -90 and 90/);
    });

    it('rejects latitude > 90', () => {
      // Verifies: Requisito 2.6, Requisito 7.6
      expect(() => new Position(90.1, 0)).toThrow(InvalidCoordinateError);
      expect(() => new Position(90.1, 0)).toThrow(/Latitude must be between -90 and 90/);
    });

    it('rejects longitude < -180', () => {
      // Verifies: Requisito 2.6, Requisito 7.6
      expect(() => new Position(0, -180.1)).toThrow(InvalidCoordinateError);
      expect(() => new Position(0, -180.1)).toThrow(/Longitude must be between -180 and 180/);
    });

    it('rejects longitude > 180', () => {
      // Verifies: Requisito 2.6, Requisito 7.6
      expect(() => new Position(0, 180.1)).toThrow(InvalidCoordinateError);
      expect(() => new Position(0, 180.1)).toThrow(/Longitude must be between -180 and 180/);
    });

    it('accepts boundary values: latitude -90', () => {
      const position = new Position(-90, 0);
      expect(position.latitude).toBe(-90);
    });

    it('accepts boundary values: latitude 90', () => {
      const position = new Position(90, 0);
      expect(position.latitude).toBe(90);
    });

    it('accepts boundary values: longitude -180', () => {
      const position = new Position(0, -180);
      expect(position.longitude).toBe(-180);
    });

    it('accepts boundary values: longitude 180', () => {
      const position = new Position(0, 180);
      expect(position.longitude).toBe(180);
    });

    it('accepts zero coordinates', () => {
      const position = new Position(0, 0);
      expect(position.latitude).toBe(0);
      expect(position.longitude).toBe(0);
    });

    it('accepts negative coordinates within range', () => {
      const position = new Position(-45.5, -120.75);
      expect(position.latitude).toBe(-45.5);
      expect(position.longitude).toBe(-120.75);
    });

      });

  // ============================================================================
  // equals() method with tolerance
  // ============================================================================

  describe('equals() method with 1e-6 tolerance', () => {
    const basePosition = new Position(40.4168, -3.7038);

    it('returns true for identical coordinates', () => {
      // Verifies: Property 3 (precisión preservada)
      const other = new Position(40.4168, -3.7038);
      expect(basePosition.equals(other)).toBe(true);
    });

    it('returns true for coordinates within tolerance (latitude)', () => {
      // Verifies: Property 3 (precisión preservada)
      const other = new Position(40.4168 + 0.5e-6, -3.7038);
      expect(basePosition.equals(other)).toBe(true);
    });

    it('returns true for coordinates within tolerance (longitude)', () => {
      // Verifies: Property 3 (precisión preservada)
      const other = new Position(40.4168, -3.7038 + 0.5e-6);
      expect(basePosition.equals(other)).toBe(true);
    });

    it('returns true for coordinates within tolerance (both)', () => {
      // Verifies: Property 3 (precisión preservada)
      const other = new Position(
        40.4168 - 0.5e-6,
        -3.7038 - 0.5e-6
      );
      expect(basePosition.equals(other)).toBe(true);
    });

    it('returns false for coordinates outside tolerance (latitude)', () => {
      // Verifies: Property 3 (precisión preservada)
      const other = new Position(40.4168 + 2e-6, -3.7038);
      expect(basePosition.equals(other)).toBe(false);
    });

    it('returns false for coordinates outside tolerance (longitude)', () => {
      // Verifies: Property 3 (precisión preservada)
      const other = new Position(40.4168, -3.7038 + 2e-6);
      expect(basePosition.equals(other)).toBe(false);
    });

    it('returns false for significantly different coordinates', () => {
      // Verifies: Property 3 (precisión preservada)
      const other = new Position(41.0, -4.0);
      expect(basePosition.equals(other)).toBe(false);
    });

    it('ignores timestamp differences in equals()', () => {
      // Verifies: Property 3 (precisión preservada)
      const position1 = new Position(40.4168, -3.7038, new Date('2024-01-15T10:00:00Z'));
      const position2 = new Position(40.4168, -3.7038, new Date('2024-01-15T11:00:00Z'));

      expect(position1.equals(position2)).toBe(true);
    });

    it('handles negative latitude differences', () => {
      // Verifies: Property 3 (precisión preservada)
      const other = new Position(40.4168 - 0.9e-6, -3.7038);
      expect(basePosition.equals(other)).toBe(true);
    });

    it('handles negative longitude differences', () => {
      // Verifies: Property 3 (precisión preservada)
      const other = new Position(40.4168, -3.7038 - 0.9e-6);
      expect(basePosition.equals(other)).toBe(true);
    });

    it('is symmetric: p1.equals(p2) === p2.equals(p1)', () => {
      // Verifies: Property 3 (precisión preservada)
      const p1 = new Position(40.4168, -3.7038);
      const p2 = new Position(40.4168 + 0.5e-6, -3.7038 + 0.5e-6);

      expect(p1.equals(p2)).toBe(p2.equals(p1));
    });

    it('is transitive: if p1.equals(p2) and p2.equals(p3), then p1.equals(p3)', () => {
      // Verifies: Property 3 (precisión preservada)
      const p1 = new Position(40.4168, -3.7038);
      const p2 = new Position(40.4168 + 0.3e-6, -3.7038 + 0.3e-6);
      const p3 = new Position(40.4168 + 0.6e-6, -3.7038 + 0.6e-6);

      if (p1.equals(p2) && p2.equals(p3)) {
        expect(p1.equals(p3)).toBe(true);
      }
    });
  });

  // ============================================================================
  // toString() method with 4 decimals
  // ============================================================================

  describe('toString() method with 4 decimal precision', () => {
    it('returns coordinates with 4 decimals', () => {
      // Verifies: Requisito 2.2
      const position = new Position(40.4168, -3.7038);
      expect(position.toString()).toBe('40.4168, -3.7038');
    });

    it('pads decimals to 4 places', () => {
      // Verifies: Requisito 2.2
      const position = new Position(40.4, -3.7);
      expect(position.toString()).toBe('40.4000, -3.7000');
    });

    it('handles zero coordinates', () => {
      // Verifies: Requisito 2.2
      const position = new Position(0, 0);
      expect(position.toString()).toBe('0.0000, 0.0000');
    });

    it('handles negative coordinates', () => {
      // Verifies: Requisito 2.2
      const position = new Position(-45.5678, -120.1234);
      expect(position.toString()).toBe('-45.5678, -120.1234');
    });

    it('rounds correctly to 4 decimals', () => {
      // Verifies: Requisito 2.2
      // toFixed(4) rounds, so 40.41681 → 40.4168
      const position = new Position(40.41681, -3.70381);
      expect(position.toString()).toBe('40.4168, -3.7038');
    });

    it('handles boundary values', () => {
      // Verifies: Requisito 2.2
      const position = new Position(-90, 180);
      expect(position.toString()).toBe('-90.0000, 180.0000');
    });

    it('handles very small positive numbers', () => {
      // Verifies: Requisito 2.2
      const position = new Position(0.00001, 0.00001);
      expect(position.toString()).toBe('0.0000, 0.0000');
    });

    it('handles numbers with high precision', () => {
      // Verifies: Requisito 2.2
      const position = new Position(40.41681234, -3.70381234);
      expect(position.toString()).toBe('40.4168, -3.7038');
    });

    it('toString() format is: "lat, lon" with no extra text', () => {
      // Verifies: Requisito 2.2
      const position = new Position(40.4168, -3.7038);
      const str = position.toString();

      // Must match pattern: number, space, number
      expect(str).toMatch(/^-?\d+\.\d{4}, -?\d+\.\d{4}$/);
    });
  });

  // ============================================================================
  // toLatLng() helper method
  // ============================================================================

  describe('toLatLng() method for map libraries', () => {
    it('returns [latitude, longitude] tuple', () => {
      const position = new Position(40.4168, -3.7038);
      const latLng = position.toLatLng();

      expect(latLng).toEqual([40.4168, -3.7038]);
      expect(latLng.length).toBe(2);
    });

    it('preserves full precision in tuple', () => {
      const position = new Position(40.41681234, -3.70381234);
      const [lat, lon] = position.toLatLng();

      expect(lat).toBe(40.41681234);
      expect(lon).toBe(-3.70381234);
    });

    it('handles boundary coordinates', () => {
      const position = new Position(-90, 180);
      expect(position.toLatLng()).toEqual([-90, 180]);
    });
  });

  // ============================================================================
  // Property-based tests: Round-trip preservation
  // ============================================================================

  describe('Property: Precision preservation in round-trip operations', () => {
    it('reconstructed position equals original (within tolerance)', () => {
      // Verifies: Property 3 (precisión preservada)
      const original = new Position(40.4168, -3.7038);
      const [lat, lon] = original.toLatLng();

      // Reconstruct from tuple
      const reconstructed = new Position(lat, lon);

      // Must be equal within tolerance
      expect(original.equals(reconstructed)).toBe(true);
    });

    it('toString preserves enough precision for reconstruction (4 decimals ≈ 11m)', () => {
      // Verifies: Property 3 (precisión preservada)
      // 4 decimals = ±1.1m precision
      const original = new Position(40.4168, -3.7038);
      const str = original.toString();

      // Parse back
      const [latStr, lonStr] = str.split(', ').map(Number);
      const reconstructed = new Position(latStr, lonStr);

      // Must be equal (same rounded values)
      expect(original.equals(reconstructed)).toBe(true);
    });
  });

  // ============================================================================
  // Edge cases and robustness
  // ============================================================================

  describe('Edge cases and robustness', () => {
    it('handles very large precision coordinates', () => {
      const position = new Position(40.123456789, -3.987654321);
      expect(position.latitude).toBe(40.123456789);
      expect(position.longitude).toBe(-3.987654321);
    });

    
    it('rejects Infinity as latitude', () => {
      expect(() => new Position(Infinity, 0)).toThrow(InvalidCoordinateError);
    });

    
    
    it('rejects Infinity as longitude', () => {
      expect(() => new Position(0, Infinity)).toThrow(InvalidCoordinateError);
    });

    
    it('timestamp is copied, not referenced', () => {
      const date = new Date('2024-01-15T10:00:00Z');
      const position = new Position(40.4168, -3.7038, date);

      // Timestamp should be the same date object (stored as provided)
      expect(position.timestamp).toEqual(date);
    });
  });

  // ============================================================================
  // Real-world scenarios
  // ============================================================================

  describe('Real-world scenarios', () => {
    it('Madrid city center coordinates', () => {
      const position = new Position(40.4168, -3.7038);
      expect(position.toString()).toBe('40.4168, -3.7038');
    });

    it('North Pole', () => {
      const position = new Position(90, 0);
      expect(position.latitude).toBe(90);
    });

    it('South Pole', () => {
      const position = new Position(-90, 0);
      expect(position.latitude).toBe(-90);
    });

    it('International Date Line - East', () => {
      const position = new Position(0, 180);
      expect(position.longitude).toBe(180);
    });

    it('International Date Line - West', () => {
      const position = new Position(0, -180);
      expect(position.longitude).toBe(-180);
    });

    it('Equator and Prime Meridian', () => {
      const position = new Position(0, 0);
      expect(position.latitude).toBe(0);
      expect(position.longitude).toBe(0);
    });

    it('Sydney, Australia', () => {
      const position = new Position(-33.8688, 151.2093);
      expect(position.toString()).toBe('-33.8688, 151.2093');
    });

    it('Tokyo, Japan', () => {
      const position = new Position(35.6762, 139.6503);
      expect(position.toString()).toBe('35.6762, 139.6503');
    });

    it('São Paulo, Brazil', () => {
      const position = new Position(-23.5505, -46.6333);
      expect(position.toString()).toBe('-23.5505, -46.6333');
    });
  });
});
