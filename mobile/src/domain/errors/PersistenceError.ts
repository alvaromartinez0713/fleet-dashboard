import { DomainError } from './DomainError';

/**
 * Raised when persistence operations (SQLite) fail.
 */
export class PersistenceError extends DomainError {
  constructor(message: string) {
    super('PERSISTENCE_ERROR', message);
    this.name = 'PersistenceError';
    Object.setPrototypeOf(this, PersistenceError.prototype);
  }
}
