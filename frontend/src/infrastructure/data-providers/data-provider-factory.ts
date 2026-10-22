import { VehicleDataProvider } from '@application/ports/vehicle-data-provider';
import { MockDataProvider } from './mock-data-provider';
import { ApiDataProvider } from './api-data-provider';
import { ApiClient } from '@infrastructure/http/api-client';

/**
 * DataProviderFactory - Factory pattern para crear instancias de VehicleDataProvider
 *
 * Selecciona la implementación apropiada (Mock o API real) basada en configuración
 * de entorno. Permite cambiar entre modo demostración y modo producción sin cambiar
 * código de la aplicación.
 *
 * Verifica: Requisito 4.6, Requisito 10 (Separación de preocupaciones)
 */

/**
 * Modo de proveedor de datos
 * - 'mock': Usa datos ficticios hardcoded para demostración y testing
 * - 'api': Consume datos desde la API real del backend
 */
export type DataProviderMode = 'mock' | 'api';

/**
 * Crea una instancia de VehicleDataProvider basada en configuración
 *
 * La selección del proveedor se determina en el siguiente orden:
 * 1. Parámetro `mode` si se proporciona explícitamente
 * 2. Variable de entorno VITE_DATA_MODE (establecida en .env.local, .env.production, etc.)
 * 3. Por defecto: 'api' en producción, 'mock' en desarrollo
 *
 * Responsabilidades:
 * - Decidir qué implementación usar basada en configuración
 * - Crear instancias con dependencias inyectadas
 * - Proporcionar valores por defecto sensatos
 * - Mantener la aplicación desacoplada de la selección del proveedor
 *
 * @param mode - Modo de proveedor (opcional): 'mock' o 'api'
 * @returns Instancia de VehicleDataProvider configurada
 *
 * @example
 * // Usar variable de entorno
 * const provider = createDataProvider();
 * const vehicles = await provider.loadVehicles();
 *
 * @example
 * // Forzar modo mock para testing
 * const mockProvider = createDataProvider('mock');
 *
 * @example
 * // Forzar modo API
 * const apiProvider = createDataProvider('api');
 *
 * Verifica: testing.md Sección 5 (Dobles de prueba - Fakes vs APIs reales)
 */
export function createDataProvider(mode?: DataProviderMode): VehicleDataProvider {
  // Determinar modo: explícito, de variable de entorno, o por defecto
  const selectedMode = mode || getEnvironmentMode();

  if (selectedMode === 'mock') {
    // Modo demostración: proveer datos mock
    return new MockDataProvider();
  }

  // Modo producción: consumir API real
  return new ApiDataProvider(new ApiClient());
}

/**
 * Lee la configuración de modo desde variables de entorno
 *
 * Busca la variable VITE_DATA_MODE (estándar de Vite para variables públicas)
 * Si no está definida, utiliza heurística basada en el entorno de compilación
 *
 * @returns 'mock' o 'api' según la configuración
 *
 * @private
 */
function getEnvironmentMode(): DataProviderMode {
  // Variable de entorno: VITE_DATA_MODE
  // Se puede establecer en:
  // - .env.local (desarrollo)
  // - .env.production (producción)
  // - Variables de entorno del sistema/CI
  const envMode = import.meta.env.VITE_DATA_MODE;

  if (envMode === 'mock' || envMode === 'api') {
    return envMode;
  }

  // Heurística: si el modo no está explícitamente configurado
  // usar 'mock' en desarrollo y 'api' en producción
  const isDevelopment = import.meta.env.DEV;
  return isDevelopment ? 'mock' : 'api';
}

/**
 * Obtiene el modo actual de proveedor de datos
 *
 * Útil para debugging, logging y decisiones condicionales en la presentación
 * (ej: mostrar banner de "modo demo")
 *
 * @returns El modo actualmente configurado: 'mock' o 'api'
 *
 * @example
 * if (getDataProviderMode() === 'mock') {
 *   console.log('Running in demo mode with mock data');
 * }
 */
export function getDataProviderMode(): DataProviderMode {
  return getEnvironmentMode();
}

/**
 * Valida que el modo proporcionado es válido
 *
 * @param mode - Modo a validar
 * @returns true si el modo es válido ('mock' o 'api'), false en caso contrario
 *
 * @example
 * const userInput = process.env.DATA_MODE;
 * if (!isValidDataProviderMode(userInput)) {
 *   throw new Error('Invalid data provider mode');
 * }
 */
export function isValidDataProviderMode(mode: unknown): mode is DataProviderMode {
  return mode === 'mock' || mode === 'api';
}
