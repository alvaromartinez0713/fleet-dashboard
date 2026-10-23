import { Coordinate } from '../../../src/domain/models/Coordinate';
import { InvalidCoordinateError } from '../../../src/domain/errors/InvalidCoordinateError';

/**
 * Test: Coordinate validation rules
 * 
 * Protects: RF-01 (Captura móvil)
 * Tests business rule: coordinates must be within valid GPS bounds
 * 
 * Pillars:
 * - Regression: detects invalid lat/lon
 * - Refactoring: passes if behavior unchanged
 * - Speed: <100ms
 * - Maintenance: 3 lines per test
 */
describe('Coordinate Domain Model', () => {
  it('accepts valid coordinates', () => {
    const coord = new Coordinate(40.7128, -74.006);
    expect(coord.latitude).toBe(40.7128);
    expect(coord.longitude).toBe(-74.006);
  });

  it('rejects latitude > 90', () => {
    expect(() => new Coordinate(91, 0)).toThrow(InvalidCoordinateError);
  });

  it('rejects latitude < -90', () => {
    expect(() => new Coordinate(-91, 0)).toThrow(InvalidCoordinateError);
  });

  it('rejects longitude > 180', () => {
    expect(() => new Coordinate(0, 181)).toThrow(InvalidCoordinateError);
  });

  it('rejects longitude < -180', () => {
    expect(() => new Coordinate(0, -181)).toThrow(InvalidCoordinateError);
  });
});
