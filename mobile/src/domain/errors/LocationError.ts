import { DomainError } from './DomainError';

/**
 * Raised when GPS capture fails (permission denied, location unavailable, etc.)
 */
export class LocationError extends DomainError {
  constructor(message: string) {
    super('LOCATION_ERROR', message);
    this.name = 'LocationError';
    Object.setPrototypeOf(this, LocationError.prototype);
  }
}
