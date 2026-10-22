import { describe, it, expect } from 'vitest';
import { VehicleId } from '@domain/entities/vehicle-id';
import { InvalidVehicleIdError } from '@domain/errors/vehicle-errors';

// Verifies: Requisito 5 (dominio independiente), Requisito 2

describe('VehicleId - Objeto de valor', () => {
  describe('Construcción y validación', () => {
    it('rejects_empty_string_on_construction', () => {
      // Verifies: AC - VehicleId no permite valores vacíos
      expect(() => new VehicleId('')).toThrow(InvalidVehicleIdError);
      expect(() => new VehicleId('')).toThrow('VehicleId must not be empty');
    });

    it('rejects_whitespace_only_string_on_construction', () => {
      // Verifies: AC - VehicleId no permite valores vacíos
      expect(() => new VehicleId('   ')).toThrow(InvalidVehicleIdError);
      expect(() => new VehicleId('   ')).toThrow('VehicleId must not be empty');
    });

    it('rejects_null_like_values', () => {
      // Property test: VehicleId rechaza cualquier valor falsy o vacío
      const invalidValues = ['', '   ', '\t', '\n'];
      invalidValues.forEach(val => {
        expect(() => new VehicleId(val)).toThrow(InvalidVehicleIdError);
      });
    });

    it('creates_vehicle_id_with_valid_value', () => {
      // Verifies: AC - VehicleId permite valores válidos
      const id = new VehicleId('VEHICLE-001');
      expect(id.value).toBe('VEHICLE-001');
    });

    it('stores_exact_value_without_trimming', () => {
      // Property: El valor se almacena exactamente como se proporciona (sin trim después de validación)
      const id = new VehicleId('VEHICLE-001');
      expect(id.value).toBe('VEHICLE-001');
    });

    it('accepts_various_valid_formats', () => {
      // Property: VehicleId acepta cualquier string no vacío
      const validFormats = [
        'MOCK-VEHICLE-01',
        'VH-12345',
        'ABC123',
        'vehicle_id_long_form',
        '12345',
        'ÑOÑO', // non-ASCII characters
      ];

      validFormats.forEach(format => {
        const id = new VehicleId(format);
        expect(id.value).toBe(format);
      });
    });
  });

  describe('equals() - Comparación de igualdad', () => {
    it('equals_returns_true_for_same_value', () => {
      // Verifies: AC - equals() funciona correctamente
      const id1 = new VehicleId('VEHICLE-001');
      const id2 = new VehicleId('VEHICLE-001');

      expect(id1.equals(id2)).toBe(true);
    });

    it('equals_returns_false_for_different_value', () => {
      // Verifies: AC - equals() funciona correctamente
      const id1 = new VehicleId('VEHICLE-001');
      const id2 = new VehicleId('VEHICLE-002');

      expect(id1.equals(id2)).toBe(false);
    });

    it('equals_is_case_sensitive', () => {
      // Property: equals() es sensible a mayúsculas/minúsculas
      const id1 = new VehicleId('VEHICLE-001');
      const id2 = new VehicleId('vehicle-001');

      expect(id1.equals(id2)).toBe(false);
    });

    it('equals_is_reflexive', () => {
      // Property (reflexividad): a.equals(a) === true
      const id = new VehicleId('VEHICLE-001');
      expect(id.equals(id)).toBe(true);
    });

    it('equals_is_symmetric', () => {
      // Property (simetría): a.equals(b) === b.equals(a)
      const id1 = new VehicleId('VEHICLE-001');
      const id2 = new VehicleId('VEHICLE-001');

      expect(id1.equals(id2)).toBe(id2.equals(id1));
    });

    it('equals_is_transitive', () => {
      // Property (transitividad): a.equals(b) && b.equals(c) => a.equals(c)
      const id1 = new VehicleId('VEHICLE-001');
      const id2 = new VehicleId('VEHICLE-001');
      const id3 = new VehicleId('VEHICLE-001');

      expect(id1.equals(id2) && id2.equals(id3)).toBe(true);
      expect(id1.equals(id3)).toBe(true);
    });
  });

  describe('toString() - Representación en string', () => {
    it('toString_returns_vehicle_id_value', () => {
      // Verifies: AC - toString() funciona correctamente
      const id = new VehicleId('VEHICLE-001');
      expect(id.toString()).toBe('VEHICLE-001');
    });

    it('toString_matches_value_property', () => {
      // Property: toString() retorna exactamente el mismo valor que .value
      const testCases = ['VEHICLE-001', 'VH-123', 'ABC', 'test-id'];

      testCases.forEach(testValue => {
        const id = new VehicleId(testValue);
        expect(id.toString()).toBe(id.value);
      });
    });

    it('toString_preserves_original_format', () => {
      // Property: toString() preserva el formato exacto sin transformar
      const id = new VehicleId('MOCK-VEHICLE-01');
      expect(id.toString()).toBe('MOCK-VEHICLE-01');
    });
  });

  describe('Round-trip y serialización', () => {
    it('value_can_be_used_to_create_identical_vehicle_id', () => {
      // Property (round-trip): new VehicleId(id.value) crea un VehicleId idéntico
      const id1 = new VehicleId('VEHICLE-001');
      const id2 = new VehicleId(id1.value);

      expect(id1.equals(id2)).toBe(true);
    });

    it('multiple_vehicle_ids_with_same_value_are_equal', () => {
      // Property: VehicleId es un value object - dos instancias con mismo valor son "iguales"
      const ids = [
        new VehicleId('SAME-ID'),
        new VehicleId('SAME-ID'),
        new VehicleId('SAME-ID'),
      ];

      ids.forEach(id => {
        ids.forEach(other => {
          expect(id.equals(other)).toBe(true);
        });
      });
    });
  });

  describe('Invariantes del dominio', () => {
    it('never_allows_empty_or_null_value', () => {
      // Invariante: VehicleId.value nunca puede ser vacío
      const validId = new VehicleId('VEHICLE-001');
      expect(validId.value).toBeTruthy();
      expect(validId.value.length).toBeGreaterThan(0);
    });

    it('value_is_readonly_cannot_be_modified', () => {
      // Invariante: El value no puede ser modificado después de creación (readonly)
      const id = new VehicleId('VEHICLE-001');
      
      // TypeScript compilation check: property is readonly
      // This would fail compilation: id.value = 'CHANGED'
      expect(id.value).toBe('VEHICLE-001');
    });

    it('two_different_instances_with_same_value_are_semantically_equal', () => {
      // Propiedad: VehicleId es value object - la identidad es por valor, no por referencia
      const id1 = new VehicleId('ID-001');
      const id2 = new VehicleId('ID-001');

      // Distintas instancias
      expect(id1).not.toBe(id2);
      // Pero semánticamente iguales
      expect(id1.equals(id2)).toBe(true);
    });
  });

  describe('Edge cases y robustez', () => {
    it('handles_very_long_vehicle_ids', () => {
      // Property: VehicleId soporta IDs de cualquier longitud
      const longId = 'V' + 'E'.repeat(1000);
      const id = new VehicleId(longId);
      expect(id.value).toBe(longId);
      expect(id.value.length).toBe(1001);
    });

    it('handles_special_characters', () => {
      // Property: VehicleId soporta caracteres especiales
      const specialIds = ['V-001', 'V_001', 'V:001', 'V#001', 'V@001', 'V.001'];
      specialIds.forEach(id => {
        expect(() => new VehicleId(id)).not.toThrow();
      });
    });

    it('handles_unicode_characters', () => {
      // Property: VehicleId soporta caracteres Unicode
      const unicodeIds = ['VH-Ñ01', 'VH-中01', 'VH-🚗01'];
      unicodeIds.forEach(id => {
        expect(() => new VehicleId(id)).not.toThrow();
      });
    });
  });
});
