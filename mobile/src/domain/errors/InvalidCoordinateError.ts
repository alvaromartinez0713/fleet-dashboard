import { DomainError } from './DomainError';

/**
 * Raised when a Coordinate has invalid latitude, longitude, or accuracy values.
 */
export class InvalidCoordinateError extends DomainError {
  constructor(message: string) {
    super('INVALID_COORDINATE', message);
    this.name = 'InvalidCoordinateError';
    Object.setPrototypeOf(this, InvalidCoordinateError.prototype);
  }
}
