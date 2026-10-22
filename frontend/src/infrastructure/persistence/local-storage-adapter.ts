import { Vehicle } from '@domain/entities/vehicle';
import { VehicleId } from '@domain/entities/vehicle-id';
import { Position } from '@domain/entities/position';
import { VehicleStorage } from '@application/ports/vehicle-storage';
import {
  InvalidVehicleError,
  InvalidCoordinateError,
  InvalidVehicleIdError,
} from '@domain/errors/vehicle-errors';

/**
 * LocalStorageAdapter - Implementación de VehicleStorage usando browser localStorage.
 *
 * Responsabilidades:
 * - Serializar entidades Vehicle a JSON para almacenamiento
 * - Deserializar JSON y reconstruir entidades Vehicle
 * - Validar datos cargados antes de reconstrucción
 * - Manejar errores de serialización gracefully
 * - Almacenar vehículos con VehicleId único como clave
 * - Proporcionar operaciones CRUD en localStorage del navegador
 *
 * Verifica: Requisito 1.6 (cache local offline), Requisito 5 (puertos desacoplados)
 *
 * @class LocalStorageAdapter
 * @implements {VehicleStorage}
 */
export class LocalStorageAdapter implements VehicleStorage {
  private static readonly STORAGE_KEY = 'fleet_vehicles';
  private static readonly VEHICLES_LIST_KEY = 'fleet_vehicles_list';

  /**
   * Serializa un vehículo a formato JSON para almacenamiento.
   *
   * @private
   * @param vehicle - Vehículo a serializar
   * @returns Objeto plano serializable a JSON
   */
  private serializeVehicle(vehicle: Vehicle): Record<string, unknown> {
    return {
      vehicleId: vehicle.vehicleId.value,
      licensePlate: vehicle.licensePlate,
      position: {
        latitude: vehicle.position.latitude,
        longitude: vehicle.position.longitude,
        timestamp: vehicle.position.timestamp.toISOString(),
      },
    };
  }

  /**
   * Deserializa un objeto JSON a una entidad Vehicle.
   * Valida los datos antes de reconstrucción y lanza errores si son inválidos.
   *
   * @private
   * @param data - Objeto serializado desde localStorage
   * @returns Entidad Vehicle reconstruida
   * @throws InvalidVehicleIdError si vehicleId es inválido
   * @throws InvalidCoordinateError si las coordenadas están fuera de rango
   * @throws InvalidVehicleError si hay campos faltantes
   */
  private deserializeVehicle(data: unknown): Vehicle {
    if (!data || typeof data !== 'object') {
      throw new InvalidVehicleError('Vehicle data must be an object');
    }

    const obj = data as Record<string, unknown>;

    // Validar presencia de campos requeridos
    if (typeof obj.vehicleId !== 'string') {
      throw new InvalidVehicleIdError('vehicleId must be a non-empty string');
    }

    if (typeof obj.licensePlate !== 'string' || !obj.licensePlate.trim()) {
      throw new InvalidVehicleError('licensePlate must be a non-empty string');
    }

    if (!obj.position || typeof obj.position !== 'object') {
      throw new InvalidVehicleError('position must be an object');
    }

    const position = obj.position as Record<string, unknown>;

    // Validar coordenadas
    if (typeof position.latitude !== 'number' || typeof position.longitude !== 'number') {
      throw new InvalidCoordinateError('latitude and longitude must be numbers');
    }

    // Validar rango de coordenadas
    if (position.latitude < -90 || position.latitude > 90) {
      throw new InvalidCoordinateError(
        `Latitude must be between -90 and 90, got ${position.latitude}`
      );
    }

    if (position.longitude < -180 || position.longitude > 180) {
      throw new InvalidCoordinateError(
        `Longitude must be between -180 and 180, got ${position.longitude}`
      );
    }

    // Validar timestamp
    let timestamp: Date;
    if (typeof position.timestamp === 'string') {
      timestamp = new Date(position.timestamp);
      if (isNaN(timestamp.getTime())) {
        throw new InvalidVehicleError('Invalid timestamp format');
      }
    } else {
      throw new InvalidVehicleError('timestamp must be an ISO string');
    }

    // Reconstruir entidades de dominio
    const vehicleId = new VehicleId(obj.vehicleId);
    const positionEntity = new Position(
      position.latitude as number,
      position.longitude as number,
      timestamp
    );

    return new Vehicle(vehicleId, obj.licensePlate, positionEntity);
  }

  /**
   * Guarda múltiples vehículos en localStorage.
   * Reemplaza completamente el contenido anterior.
   *
   * @async
   * @param vehicles - Array de vehículos a guardar
   * @returns Promise que se resuelve cuando la operación completa
   * @throws Error si localStorage no está disponible o lleno
   */
  async saveVehicles(vehicles: Vehicle[]): Promise<void> {
    try {
      const vehiclesList: Record<string, Record<string, unknown>> = {};
      const vehicleIds: string[] = [];

      for (const vehicle of vehicles) {
        const vehicleId = vehicle.vehicleId.value;
        vehiclesList[vehicleId] = this.serializeVehicle(vehicle);
        vehicleIds.push(vehicleId);
      }

      // Almacenar el mapa completo de vehículos
      localStorage.setItem(
        LocalStorageAdapter.STORAGE_KEY,
        JSON.stringify(vehiclesList)
      );

      // Almacenar lista de IDs para recuperación rápida
      localStorage.setItem(
        LocalStorageAdapter.VEHICLES_LIST_KEY,
        JSON.stringify(vehicleIds)
      );
    } catch (error) {
      if (error instanceof Error) {
        // Capturar QuotaExceededError y otros errores de localStorage
        throw new Error(`Failed to save vehicles to localStorage: ${error.message}`);
      }
      throw error;
    }
  }

  /**
   * Carga todos los vehículos desde localStorage.
   *
   * @async
   * @returns Promise que se resuelve con array de vehículos cargados.
   *          Si no hay vehículos guardados, retorna array vacío.
   *          Si hay datos inválidos, los omite y continúa con datos válidos.
   */
  async loadVehicles(): Promise<Vehicle[]> {
    try {
      const stored = localStorage.getItem(LocalStorageAdapter.STORAGE_KEY);

      if (!stored) {
        return [];
      }

      const vehiclesList = JSON.parse(stored) as Record<string, unknown>;

      if (!vehiclesList || typeof vehiclesList !== 'object') {
        return [];
      }

      const vehicles: Vehicle[] = [];

      // Iterar sobre cada vehículo almacenado
      for (const [vehicleId, vehicleData] of Object.entries(vehiclesList)) {
        try {
          const vehicle = this.deserializeVehicle(vehicleData);
          vehicles.push(vehicle);
        } catch (error) {
          // Log del error pero continúa cargando otros vehículos válidos
          console.warn(
            `Failed to deserialize vehicle ${vehicleId}: ${error instanceof Error ? error.message : 'Unknown error'}`
          );
        }
      }

      return vehicles;
    } catch (error) {
      // Si hay error al parsear JSON o acceder a localStorage, retornar array vacío
      if (error instanceof SyntaxError) {
        console.warn('Failed to parse vehicles from localStorage: invalid JSON');
        return [];
      }

      if (error instanceof Error) {
        console.warn(`Failed to load vehicles from localStorage: ${error.message}`);
      }

      return [];
    }
  }

  /**
   * Limpia completamente el almacenamiento local de vehículos.
   *
   * @async
   * @returns Promise que se resuelve cuando la operación completa
   */
  async clear(): Promise<void> {
    try {
      localStorage.removeItem(LocalStorageAdapter.STORAGE_KEY);
      localStorage.removeItem(LocalStorageAdapter.VEHICLES_LIST_KEY);
    } catch (error) {
      // En algunos navegadores, localStorage.removeItem puede lanzar errores
      if (error instanceof Error) {
        throw new Error(`Failed to clear localStorage: ${error.message}`);
      }
      throw error;
    }
  }
}
