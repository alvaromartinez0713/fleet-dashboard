import { IConnectivityProvider, ConnectivityStatus } from '../ports/IConnectivityProvider';

/**
 * GetConnectivityStatusUseCase
 *
 * Retrieves the current connectivity status of the device.
 *
 * This use case supports RNF-07 (Operación offline-first) by allowing the application
 * to check whether the device currently has network connectivity before attempting
 * to send data.
 *
 * The use case delegates all connectivity detection to the IConnectivityProvider,
 * which abstracts platform-specific network state detection (iOS, Android).
 *
 * @example
 * const usecase = new GetConnectivityStatusUseCase(connectivityProvider);
 * const status = await usecase.execute();
 * if (status === 'offline') {
 *   console.log('Device is offline, will retry later');
 * }
 */
export class GetConnectivityStatusUseCase {
  constructor(private readonly connectivityProvider: IConnectivityProvider) {}

  /**
   * Execute the use case: retrieve current connectivity status
   *
   * @returns Promise that resolves to 'online' or 'offline'
   * @throws May throw if unable to determine connectivity status
   *
   * Implementation steps:
   * 1. Query the connectivity provider for the current status
   * 2. Return the status to the caller
   */
  async execute(): Promise<ConnectivityStatus> {
    return this.connectivityProvider.getStatus();
  }
}
