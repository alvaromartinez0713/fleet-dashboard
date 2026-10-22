import { Vehicle } from '@domain/entities/vehicle';

/**
 * VehicleStorage - Puerto para almacenamiento local persistente de vehículos.
 * Define el contrato que deben cumplir las implementaciones de persistencia local
 * (LocalStorage, IndexedDB, SQLite en móvil, etc.)
 *
 * Verifica: Requisito 1.6 (cache local offline), Requisito 5 (puertos desacoplados)
 */
export interface VehicleStorage {
  /**
   * Guarda múltiples vehículos en almacenamiento local.
   * Reemplaza completamente el contenido anterior.
   *
   * @param vehicles - Array de vehículos a guardar
   * @returns Promise que se resuelve cuando la operación completa
   */
  saveVehicles(vehicles: Vehicle[]): Promise<void>;

  /**
   * Carga todos los vehículos desde almacenamiento local.
   *
   * @returns Promise que se resuelve con el array de vehículos guardados.
   *          Si no hay vehículos guardados, retorna un array vacío.
   */
  loadVehicles(): Promise<Vehicle[]>;

  /**
   * Limpia completamente el almacenamiento local de vehículos.
   *
   * @returns Promise que se resuelve cuando la operación completa
   */
  clear(): Promise<void>;
}
