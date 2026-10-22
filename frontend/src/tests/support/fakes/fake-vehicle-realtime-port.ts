import {
  VehicleRealtimePort,
  PositionUpdateEvent,
  ConnectionStatus,
  Unsubscribe,
} from '@application/ports/vehicle-realtime-port';

/**
 * FakeVehicleRealtimePort - Implementación fake de VehicleRealtimePort para testing
 *
 * Permite:
 * - Simular conexiones SSE
 * - Emitir eventos de actualización de posición bajo control manual
 * - Cambiar estado de conexión para probar reconexión
 * - Verificar que los listeners se registran correctamente
 *
 * Verifica: testing.md Sección 5 (Fakes con estado)
 */
export class FakeVehicleRealtimePort implements VehicleRealtimePort {
  private isConnected: boolean = false;
  private status: ConnectionStatus = ConnectionStatus.DISCONNECTED;
  private positionUpdateCallbacks: ((event: PositionUpdateEvent) => void)[] = [];
  private statusChangeCallbacks: ((status: ConnectionStatus) => void)[] = [];
  private connectError: Error | null = null;
  private disconnectCalled: boolean = false;

  /**
   * Configura que la conexión lance error
   */
  withConnectError(error: Error): this {
    this.connectError = error;
    return this;
  }

  /**
   * Simula una conexión SSE exitosa
   */
  async connect(): Promise<void> {
    if (this.connectError) {
      throw this.connectError;
    }

    this.isConnected = true;
    this.status = ConnectionStatus.CONNECTED;
    this.notifyStatusChange(ConnectionStatus.CONNECTED);
  }

  /**
   * Registra callback para actualizaciones de posición
   */
  onPositionUpdate(callback: (event: PositionUpdateEvent) => void): Unsubscribe {
    this.positionUpdateCallbacks.push(callback);

    // Retorna función para desuscribirse
    return () => {
      const index = this.positionUpdateCallbacks.indexOf(callback);
      if (index > -1) {
        this.positionUpdateCallbacks.splice(index, 1);
      }
    };
  }

  /**
   * Registra callback para cambios de estado
   */
  onConnectionStatusChange(callback: (status: ConnectionStatus) => void): Unsubscribe {
    this.statusChangeCallbacks.push(callback);

    // Retorna función para desuscribirse
    return () => {
      const index = this.statusChangeCallbacks.indexOf(callback);
      if (index > -1) {
        this.statusChangeCallbacks.splice(index, 1);
      }
    };
  }

  /**
   * Desconecta y limpia listeners
   */
  async disconnect(): Promise<void> {
    this.disconnectCalled = true;
    this.isConnected = false;
    this.status = ConnectionStatus.DISCONNECTED;
    this.notifyStatusChange(ConnectionStatus.DISCONNECTED);
  }

  /**
   * Retorna el estado actual de conexión
   */
  getStatus(): ConnectionStatus {
    return this.status;
  }

  // === Métodos auxiliares para testing ===

  /**
   * Simula la recepción de un evento de actualización de posición
   */
  emitPositionUpdate(event: PositionUpdateEvent): void {
    this.positionUpdateCallbacks.forEach((callback) => {
      callback(event);
    });
  }

  /**
   * Simula un cambio de estado de conexión
   */
  emitStatusChange(newStatus: ConnectionStatus): void {
    this.status = newStatus;
    this.notifyStatusChange(newStatus);
  }

  /**
   * Notifica a todos los listeners del cambio de estado
   */
  private notifyStatusChange(newStatus: ConnectionStatus): void {
    this.statusChangeCallbacks.forEach((callback) => {
      callback(newStatus);
    });
  }

  /**
   * Retorna el número de listeners registrados para actualizaciones
   */
  getPositionUpdateListenerCount(): number {
    return this.positionUpdateCallbacks.length;
  }

  /**
   * Retorna el número de listeners registrados para cambios de estado
   */
  getStatusChangeListenerCount(): number {
    return this.statusChangeCallbacks.length;
  }

  /**
   * Verifica si se llamó a disconnect
   */
  wasDisconnectCalled(): boolean {
    return this.disconnectCalled;
  }

  /**
   * Reinicia el estado para otra prueba
   */
  reset(): void {
    this.isConnected = false;
    this.status = ConnectionStatus.DISCONNECTED;
    this.positionUpdateCallbacks = [];
    this.statusChangeCallbacks = [];
    this.connectError = null;
    this.disconnectCalled = false;
  }
}
