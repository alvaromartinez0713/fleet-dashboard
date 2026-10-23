import {
  CapturePositionUseCase,
  GetCapturedPositionsUseCase,
  StartCaptureUseCase,
  StopCaptureUseCase,
  GetConnectivityStatusUseCase,
  SyncPositionUseCase,
  SyncPendingPositionsUseCase,
  RetryFailedSyncUseCase,
  CaptureAndSyncPositionUseCase,
} from '../../application/usecases';
import { ILocationProvider } from '../../application/ports/LocationProvider';
import { IPersistenceProvider } from '../../application/ports/IPersistenceProvider';
import { IConnectivityProvider } from '../../application/ports/IConnectivityProvider';
import { IApiClient } from '../../application/ports/IApiClient';
import { Clock } from '../ports/Clock';
import { IdentifierGenerator } from '../ports/IdentifierGenerator';

/**
 * ServiceFactory
 *
 * Composes all use cases with their dependencies.
 */
export class ServiceFactory {
  constructor(
    private locationProvider: ILocationProvider,
    private persistenceProvider: IPersistenceProvider,
    private connectivityProvider: IConnectivityProvider,
    private apiClient: IApiClient,
    private clock: Clock,
    private idGenerator: IdentifierGenerator
  ) {}

  // Core capture use cases
  createCapturePositionUseCase(): CapturePositionUseCase {
    return new CapturePositionUseCase(
      this.locationProvider,
      this.persistenceProvider,
      this.clock,
      this.idGenerator
    );
  }

  createGetCapturedPositionsUseCase(): GetCapturedPositionsUseCase {
    return new GetCapturedPositionsUseCase(this.persistenceProvider);
  }

  createStartCaptureUseCase(): StartCaptureUseCase {
    return new StartCaptureUseCase(this.createCapturePositionUseCase());
  }

  createStopCaptureUseCase(): StopCaptureUseCase {
    return new StopCaptureUseCase();
  }

  createGetConnectivityStatusUseCase(): GetConnectivityStatusUseCase {
    return new GetConnectivityStatusUseCase(this.connectivityProvider);
  }

  // Sync use cases
  createSyncPositionUseCase(): SyncPositionUseCase {
    return new SyncPositionUseCase(
      this.apiClient,
      this.persistenceProvider,
      this.connectivityProvider,
      this.clock
    );
  }

  createSyncPendingPositionsUseCase(): SyncPendingPositionsUseCase {
    return new SyncPendingPositionsUseCase(
      this.createSyncPositionUseCase(),
      this.persistenceProvider,
      this.clock
    );
  }

  createRetryFailedSyncUseCase(): RetryFailedSyncUseCase {
    return new RetryFailedSyncUseCase(this.createSyncPendingPositionsUseCase());
  }

  createCaptureAndSyncPositionUseCase(): CaptureAndSyncPositionUseCase {
    return new CaptureAndSyncPositionUseCase(
      this.createCapturePositionUseCase(),
      this.createSyncPositionUseCase(),
      this.connectivityProvider
    );
  }
}
