/**
 * ConnectionStatus - Estados posibles de la conexión SSE.
 */
export enum ConnectionStatus {
  CONNECTED = 'connected',
  DISCONNECTED = 'disconnected',
  RECONNECTING = 'reconnecting',
  ERROR = 'error',
}

/**
 * PositionUpdateEvent - Evento de actualización de posición recibido por SSE.
 */
export interface PositionUpdateEvent {
  vehicleId: string;
  latitude: number;
  longitude: number;
  timestamp: string; // ISO 8601
}

/**
 * Unsubscribe - Función para desuscribirse de eventos.
 */
export type Unsubscribe = () => void;

/**
 * VehicleRealtimePort - Puerto (interfaz) para conexión en tiempo real vía SSE.
 * Las implementaciones pueden ser SSE real o simulador.
 */
export interface VehicleRealtimePort {
  /**
   * Conecta al servidor SSE.
   */
  connect(): Promise<void>;

  /**
   * Se suscribe a actualizaciones de posición.
   */
  onPositionUpdate(callback: (event: PositionUpdateEvent) => void): Unsubscribe;

  /**
   * Se suscribe a cambios de estado de conexión.
   */
  onConnectionStatusChange(callback: (status: ConnectionStatus) => void): Unsubscribe;

  /**
   * Desconecta del servidor SSE.
   */
  disconnect(): Promise<void>;

  /**
   * Obtiene el estado actual de la conexión.
   */
  getStatus(): ConnectionStatus;
}
