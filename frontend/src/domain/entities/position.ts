import { InvalidCoordinateError } from '../errors/vehicle-errors';

/**
 * Position - Objeto de valor que representa una posición geográfica con timestamp.
 * Valida que las coordenadas estén dentro de los rangos válidos:
 * - Latitud: [-90, 90]
 * - Longitud: [-180, 180]
 *
 * Proporciona comparación con tolerancia de 1e-6 decimales y representación
 * con 4 decimales de precisión (~11 metros).
 */
export class Position {
  readonly latitude: number;
  readonly longitude: number;
  readonly timestamp: Date;

  constructor(latitude: number, longitude: number, timestamp?: Date) {
    this.validate(latitude, longitude);
    this.latitude = latitude;
    this.longitude = longitude;
    this.timestamp = timestamp ?? new Date();
  }

  /**
   * Valida que las coordenadas estén dentro de los rangos permitidos.
   * Lanza InvalidCoordinateError si las coordenadas son inválidas.
   */
  private validate(lat: number, lon: number): void {
    if (lat < -90 || lat > 90) {
      throw new InvalidCoordinateError(
        `Latitude must be between -90 and 90, got ${lat}`
      );
    }
    if (lon < -180 || lon > 180) {
      throw new InvalidCoordinateError(
        `Longitude must be between -180 and 180, got ${lon}`
      );
    }
  }

  /**
   * Compara esta posición con otra usando tolerancia de 1e-6 en decimales.
   * Dos posiciones son iguales si sus coordenadas están dentro de la tolerancia.
   * El timestamp NO se incluye en la comparación (solo coordenadas).
   */
  equals(other: Position): boolean {
    return (
      Math.abs(this.latitude - other.latitude) < 1e-6 &&
      Math.abs(this.longitude - other.longitude) < 1e-6
    );
  }

  /**
   * Representación en string con 4 decimales de precisión para latitud y longitud.
   * Formato: "lat, lon" (ej: "40.4168, -3.7038")
   */
  toString(): string {
    return `${this.latitude.toFixed(4)}, ${this.longitude.toFixed(4)}`;
  }

  /**
   * Retorna un array [latitude, longitude] para uso en librerías de mapas.
   */
  toLatLng(): [number, number] {
    return [this.latitude, this.longitude];
  }
}
