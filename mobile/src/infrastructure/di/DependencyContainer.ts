import { ServiceFactory } from './ServiceFactory';
import { ILocationProvider } from '../../application/ports/LocationProvider';
import { IPersistenceProvider } from '../../application/ports/IPersistenceProvider';
import { IConnectivityProvider } from '../../application/ports/IConnectivityProvider';
import { IApiClient } from '../../application/ports/IApiClient';
import { Clock } from '../ports/Clock';
import { IdentifierGenerator } from '../ports/IdentifierGenerator';

/**
 * DependencyContainer
 *
 * Registers providers and manages dependency injection.
 * Supports different implementations (real vs fake) based on environment.
 */
export class DependencyContainer {
  private serviceFactory: ServiceFactory;

  constructor(
    private locationProvider: ILocationProvider,
    private persistenceProvider: IPersistenceProvider,
    private connectivityProvider: IConnectivityProvider,
    private apiClient: IApiClient,
    private clock: Clock,
    private idGenerator: IdentifierGenerator
  ) {
    this.serviceFactory = new ServiceFactory(
      locationProvider,
      persistenceProvider,
      connectivityProvider,
      apiClient,
      clock,
      idGenerator
    );
  }

  // Public accessor for service factory
  getServiceFactory(): ServiceFactory {
    return this.serviceFactory;
  }

  // Provider accessors
  getLocationProvider(): ILocationProvider {
    return this.locationProvider;
  }

  getPersistenceProvider(): IPersistenceProvider {
    return this.persistenceProvider;
  }

  getConnectivityProvider(): IConnectivityProvider {
    return this.connectivityProvider;
  }

  getApiClient(): IApiClient {
    return this.apiClient;
  }

  getClock(): Clock {
    return this.clock;
  }

  getIDGenerator(): IdentifierGenerator {
    return this.idGenerator;
  }

  /**
   * Override a provider for testing
   */
  overrideLocationProvider(provider: ILocationProvider): void {
    this.locationProvider = provider;
    this.recreateServiceFactory();
  }

  overridePersistenceProvider(provider: IPersistenceProvider): void {
    this.persistenceProvider = provider;
    this.recreateServiceFactory();
  }

  overrideConnectivityProvider(provider: IConnectivityProvider): void {
    this.connectivityProvider = provider;
    this.recreateServiceFactory();
  }

  overrideApiClient(client: IApiClient): void {
    this.apiClient = client;
    this.recreateServiceFactory();
  }

  overrideClock(clock: Clock): void {
    this.clock = clock;
    this.recreateServiceFactory();
  }

  overrideIDGenerator(generator: IdentifierGenerator): void {
    this.idGenerator = generator;
    this.recreateServiceFactory();
  }

  /**
   * Recreate service factory after provider override
   */
  private recreateServiceFactory(): void {
    this.serviceFactory = new ServiceFactory(
      this.locationProvider,
      this.persistenceProvider,
      this.connectivityProvider,
      this.apiClient,
      this.clock,
      this.idGenerator
    );
  }
}
