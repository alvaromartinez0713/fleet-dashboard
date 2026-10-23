/**
 * Base error class for all domain-level errors.
 * Ensures domain errors are distinguishable from infrastructure/framework errors.
 */
export class DomainError extends Error {
  readonly code: string;

  constructor(code: string, message: string) {
    super(message);
    this.code = code;
    this.name = 'DomainError';
    Object.setPrototypeOf(this, DomainError.prototype);
  }

  toString(): string {
    return `${this.name} [${this.code}]: ${this.message}`;
  }
}
