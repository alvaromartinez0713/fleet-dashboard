/**
 * InvalidVehicleIdError - Se lanza cuando el VehicleId es inválido.
 */
export class InvalidVehicleIdError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'InvalidVehicleIdError';
  }
}

/**
 * InvalidCoordinateError - Se lanza cuando las coordenadas son inválidas.
 */
export class InvalidCoordinateError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'InvalidCoordinateError';
  }
}

/**
 * InvalidVehicleError - Se lanza cuando el vehículo es inválido.
 */
export class InvalidVehicleError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'InvalidVehicleError';
  }
}
