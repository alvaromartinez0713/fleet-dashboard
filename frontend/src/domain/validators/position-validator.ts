/**
 * PositionValidator - Valida coordenadas de posición geográfica.
 *
 * Reglas de validación:
 * - Latitud: entre -90 y 90 grados
 * - Longitud: entre -180 y 180 grados
 * - Ambas deben ser números válidos
 */
export class PositionValidator {
  /**
   * Valida una par de coordenadas geográficas.
   *
   * @param latitude - Latitud (-90 a 90)
   * @param longitude - Longitud (-180 a 180)
   * @throws Error si las coordenadas son inválidas
   */
  static validate(latitude: number, longitude: number): void {
    if (!Number.isFinite(latitude)) {
      throw new Error(`Latitude must be a valid number, got ${latitude}`);
    }
    if (!Number.isFinite(longitude)) {
      throw new Error(`Longitude must be a valid number, got ${longitude}`);
    }
    if (latitude < -90 || latitude > 90) {
      throw new Error(`Latitude must be between -90 and 90, got ${latitude}`);
    }
    if (longitude < -180 || longitude > 180) {
      throw new Error(`Longitude must be between -180 and 180, got ${longitude}`);
    }
  }
}
