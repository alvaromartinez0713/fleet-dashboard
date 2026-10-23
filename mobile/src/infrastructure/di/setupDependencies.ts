import { DependencyContainer } from './DependencyContainer';
import { RealLocationProvider } from '../platform/RealLocationProvider';
import { RealConnectivityProvider } from '../platform/RealConnectivityProvider';
import { RealClock } from '../platform/RealClock';
import { RealIDGenerator } from '../platform/RealIDGenerator';
import { RealApiClient } from '../api/RealApiClient';
import { SQLitePersistenceProvider } from '../persistence/SQLitePersistenceProvider';

/**
 * setupDependencies
 *
 * Factory function to create and initialize the DependencyContainer
 * with all real (production) implementations.
 */
export async function setupDependencies(): Promise<DependencyContainer> {
  try {
    console.log('[setupDependencies] Starting initialization...');

    // Initialize persistence provider (handles SQLite setup and migrations)
    const persistenceProvider = new SQLitePersistenceProvider();
    await persistenceProvider.initialize();
    console.log('[setupDependencies] SQLite initialized');

    // Create other providers
    const locationProvider = new RealLocationProvider();
    const connectivityProvider = new RealConnectivityProvider();
    const clock = new RealClock();
    const idGenerator = new RealIDGenerator();
    const apiClient = new RealApiClient();

    // Create and return the container
    const container = new DependencyContainer(
      locationProvider,
      persistenceProvider,
      connectivityProvider,
      apiClient,
      clock,
      idGenerator
    );

    console.log('[setupDependencies] Initialization complete');
    return container;
  } catch (error) {
    console.error('[setupDependencies] Initialization failed:', error);
    throw error;
  }
}
