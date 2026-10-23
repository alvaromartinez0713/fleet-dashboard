import { IdentifierGenerator } from '../ports/IdentifierGenerator';

/**
 * RealIDGenerator
 *
 * Production implementation of IdentifierGenerator using Math.random() to generate UUIDs.
 * UUID format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx (v4)
 */
export class RealIDGenerator implements IdentifierGenerator {
  generateUUID(): string {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function (c) {
      const r = (Math.random() * 16) | 0;
      const v = c === 'x' ? r : (r & 0x3) | 0x8;
      return v.toString(16);
    });
  }
}
