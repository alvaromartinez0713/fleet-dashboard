import { Position } from '../../domain/models/Position';
import { IPersistenceProvider } from '../ports/IPersistenceProvider';

/**
 * GetCapturedPositionsUseCase
 *
 * Retrieves all positions that have been captured and stored locally.
 * Results are returned in descending order by capturedAt (newest first).
 *
 * This use case bridges the domain (Position entity) with the presentation layer,
 * allowing UI to display the complete history of captured locations.
 *
 * Requirements covered:
 * - RF-02: Display captured positions to the user
 * - RNF-01: Clean Architecture - coordinates domain and infrastructure
 */
export class GetCapturedPositionsUseCase {
  constructor(private persistenceProvider: IPersistenceProvider) {}

  /**
   * Execute the use case.
   *
   * Queries all positions from SQLite and returns them ordered by capturedAt descending.
   * The persistence provider ensures proper ordering.
   *
   * @returns Promise that resolves to an array of Position entities
   *          Empty array if no positions have been captured yet
   * @throws Error if persistence operation fails
   */
  async execute(): Promise<Position[]> {
    return this.persistenceProvider.getAllPositions();
  }
}
