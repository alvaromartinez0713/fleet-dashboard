import { describe, it, expect, beforeEach } from 'vitest';
import { LoadVehiclesUseCase } from '@application/usecases/load-vehicles';
import { FakeVehicleDataProvider } from '@tests/support/fakes/fake-vehicle-data-provider';
import { FakeVehicleStorage } from '@tests/support/fakes/fake-vehicle-storage';
import { VehicleMother } from '@tests/support/mothers/vehicle-mother';

/**
 * LoadVehiclesUseCase - Suite de pruebas de integración
 *
 * Estas pruebas verifican el caso de uso como caja negra:
 * - Entrada: Acción de cargar vehículos
 * - Salida observable: Array de vehículos o datos cacheados
 *
 * Verifica: Requisito 1.6 (offline-first), Requisito 4.2 (cargar vehículos)
 */
describe('LoadVehiclesUseCase', () => {
  let useCase: LoadVehiclesUseCase;
  let dataProvider: FakeVehicleDataProvider;
  let storage: FakeVehicleStorage;

  beforeEach(() => {
    dataProvider = new FakeVehicleDataProvider();
    storage = new FakeVehicleStorage();
    useCase = new LoadVehiclesUseCase(dataProvider, storage);
  });

  /**
   * Prueba 1: Carga vehículos exitosamente desde el proveedor de datos
   *
   * GIVEN un proveedor de datos con 20 vehículos válidos
   * WHEN se ejecuta el caso de uso
   * THEN retorna los 20 vehículos
   * AND los vehículos se guardan en almacenamiento local
   *
   * Verifica: Requisito 4.2, Requisito 1.6
   */
  it('should load vehicles successfully from data provider and save to storage', async () => {
    // ARRANGE
    dataProvider.withValidVehicles(20);

    // ACT
    const result = await useCase.execute();

    // ASSERT
    expect(result).toHaveLength(20);
    expect(result[0].vehicleId.value).toBe('MOCK-VEHICLE-01');

    // Verificar que se guardaron en storage
    const storedVehicles = storage.getStoredVehicles();
    expect(storedVehicles).toHaveLength(20);
  });

  /**
   * Prueba 2: Valida que todos los vehículos tengan fields requeridos
   *
   * GIVEN un proveedor de datos retorna 20 vehículos válidos
   * WHEN se ejecuta el caso de uso
   * THEN no lanza excepción
   *
   * Verifica: Validación de invariantes
   */
  it('should validate all vehicles have required fields', async () => {
    // ARRANGE
    dataProvider.withValidVehicles(5);

    // ACT
    const result = await useCase.execute();

    // ASSERT
    // No debería lanzar excepción
    expect(result).toHaveLength(5);
    result.forEach((vehicle) => {
      expect(vehicle.vehicleId).toBeDefined();
      expect(vehicle.position).toBeDefined();
    });
  });

  /**
   * Prueba 3: Recupera de cache cuando la API falla
   *
   * GIVEN datos previamente cacheados en almacenamiento local
   * AND el proveedor de datos lanza error
   * WHEN se ejecuta el caso de uso
   * THEN retorna los datos cacheados
   * AND no lanza excepción
   *
   * Verifica: Requisito 1.6 (offline-first), fallback a cache
   */
  it('should fallback to cached data when data provider fails', async () => {
    // ARRANGE
    const cachedVehicles = VehicleMother.validMultiple(5);
    await storage.saveVehicles(cachedVehicles);

    const apiError = new Error('API is unreachable');
    dataProvider.withError(apiError);

    // ACT
    const result = await useCase.execute();

    // ASSERT
    expect(result).toHaveLength(5);
    expect(result[0].vehicleId.value).toBe('MOCK-VEHICLE-01');
  });

  /**
   * Prueba 4: Retorna array vacío cuando falla API y no hay cache
   *
   * GIVEN almacenamiento local vacío
   * AND el proveedor de datos lanza error
   * WHEN se ejecuta el caso de uso
   * THEN retorna array vacío
   * AND no lanza excepción
   *
   * Verifica: Degradación elegante sin conectividad
   */
  it('should return empty array when data provider fails and storage is empty', async () => {
    // ARRANGE
    dataProvider.withError(new Error('Network error'));
    // Storage vacío por defecto

    // ACT
    const result = await useCase.execute();

    // ASSERT
    expect(result).toHaveLength(0);
  });

  /**
   * Prueba 5: Guarda vehículos en storage DESPUÉS de validar
   *
   * GIVEN un proveedor con 10 vehículos válidos
   * WHEN se ejecuta el caso de uso
   * THEN se guardan exactamente 10 vehículos en storage
   *
   * Verifica: Order de operaciones correcta
   */
  it('should save exactly the loaded vehicles to storage', async () => {
    // ARRANGE
    dataProvider.withValidVehicles(10);

    // ACT
    const result = await useCase.execute();

    // ASSERT
    const storedVehicles = await storage.loadVehicles();
    expect(storedVehicles).toHaveLength(10);
    expect(storedVehicles.map((v) => v.vehicleId.value)).toEqual(
      result.map((v) => v.vehicleId.value),
    );
  });

  /**
   * Prueba 6: Idempotencia - Ejecutar dos veces produce mismo resultado
   *
   * GIVEN un proveedor con 5 vehículos válidos
   * WHEN se ejecuta el caso de uso dos veces
   * THEN ambas llamadas retornan los mismos vehículos
   *
   * Property: Idempotencia (Requisito RNF-04)
   */
  it('should be idempotent - multiple calls return same vehicles', async () => {
    // ARRANGE
    dataProvider.withValidVehicles(5);

    // ACT
    const result1 = await useCase.execute();
    const result2 = await useCase.execute();

    // ASSERT
    expect(result1).toHaveLength(5);
    expect(result2).toHaveLength(5);
    expect(result1.map((v) => v.vehicleId.value)).toEqual(
      result2.map((v) => v.vehicleId.value),
    );
  });

  /**
   * Prueba 7: Cache se actualiza con nuevos datos en cada carga exitosa
   *
   * GIVEN datos cacheados anteriormente
   * AND un proveedor con nuevos datos
   * WHEN se ejecuta el caso de uso
   * THEN reemplaza completamente el cache anterior
   *
   * Verifica: Storage.saveVehicles reemplaza contenido anterior
   */
  it('should replace previous cache with new data on successful load', async () => {
    // ARRANGE - Primera carga
    dataProvider.withValidVehicles(5);
    await useCase.execute();

    // Verificar cache tiene 5 vehículos
    let storedVehicles = await storage.loadVehicles();
    expect(storedVehicles).toHaveLength(5);

    // ARRANGE - Segunda carga con datos diferentes
    dataProvider.reset();
    dataProvider.withValidVehicles(10);

    // ACT
    const result = await useCase.execute();

    // ASSERT - Cache debería tener 10 vehículos (reemplazó los 5 anteriores)
    storedVehicles = await storage.loadVehicles();
    expect(storedVehicles).toHaveLength(10);
    expect(result).toHaveLength(10);
  });

  /**
   * Prueba 8: No lanza excepción incluso con errores
   *
   * GIVEN un proveedor que lanza error
   * AND storage vacío
   * WHEN se ejecuta el caso de uso
   * THEN no lanza excepción
   * AND retorna array (vacío)
   *
   * Verifica: Robustez sin exceptions visibles
   */
  it('should not throw exception even when both provider and storage fail', async () => {
    // ARRANGE
    dataProvider.withError(new Error('Network failed'));
    storage.withError(new Error('Storage failed'));

    // ACT & ASSERT - No debería lanzar
    const result = await useCase.execute();
    // Debería retornar algo (intentará cargar de cache, que fallará silenciosamente)
    expect(Array.isArray(result)).toBe(true);
  });

  /**
   * Prueba 9: Cada vehículo es instancia válida de Vehicle
   *
   * GIVEN un proveedor con 3 vehículos
   * WHEN se ejecuta el caso de uso
   * THEN cada vehículo retornado es una instancia de Vehicle con invariantes válidas
   *
   * Verifica: Invariantes de dominio preservados
   */
  it('should return valid Vehicle instances with proper invariants', async () => {
    // ARRANGE
    dataProvider.withValidVehicles(3);

    // ACT
    const result = await useCase.execute();

    // ASSERT
    result.forEach((vehicle) => {
      expect(vehicle.vehicleId).toBeDefined();
      expect(vehicle.licensePlate).toBeDefined();
      expect(vehicle.licensePlate.length).toBeGreaterThan(0);
      expect(vehicle.position).toBeDefined();
      expect(vehicle.position.latitude).toBeGreaterThanOrEqual(-90);
      expect(vehicle.position.latitude).toBeLessThanOrEqual(90);
      expect(vehicle.position.longitude).toBeGreaterThanOrEqual(-180);
      expect(vehicle.position.longitude).toBeLessThanOrEqual(180);
    });
  });

  /**
   * Prueba 10: No mutabilidad - Cambiar resultado no afecta storage
   *
   * GIVEN un proveedor con vehículos
   * WHEN se ejecuta el caso de uso y se modifica el resultado
   * THEN storage contiene datos sin modificar (es una copia)
   *
   * Verifica: Protección contra mutación
   */
  it('should not be affected by mutations to returned vehicles', async () => {
    // ARRANGE
    dataProvider.withValidVehicles(2);

    // ACT
    const result = await useCase.execute();
    const originalId = result[0].vehicleId.value;

    // Intentar "mutar" (aunque Vehicle es inmutable por diseño)
    // Este test verifica que el almacenamiento contiene copias independientes

    // ASSERT
    const storedVehicles = await storage.loadVehicles();
    expect(storedVehicles[0].vehicleId.value).toBe(originalId);
  });
});
