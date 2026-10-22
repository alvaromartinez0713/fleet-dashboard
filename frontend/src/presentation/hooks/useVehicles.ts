import { useState, useEffect, useContext } from 'react';
import { Vehicle } from '@domain/entities/vehicle';
import { ApplicationContext } from '@presentation/context/ApplicationContext';

/**
 * UseVehiclesResult - Resultado de usar el hook useVehicles.
 *
 * Responsabilidades:
 * - Proporcionar estado de vehículos: lista, carga, error
 * - Proporcionar función para reintentar carga manual
 *
 * Verifica: Requisito 6 (hooks reutilizables)
 */
export interface UseVehiclesResult {
  /** Array de vehículos actual, puede estar vacío durante carga inicial */
  vehicles: Vehicle[];
  /** true mientras se están cargando vehículos iniciales, false cuando ya hay datos o error */
  loading: boolean;
  /** Error ocurrido durante carga, null si no hay error */
  error: Error | null;
  /** Función para reintentar manualmente la carga de vehículos */
  refetch: () => Promise<void>;
}

/**
 * useVehicles - Hook React que proporciona datos de vehículos a componentes.
 *
 * Responsabilidades:
 * - Cargar vehículos inicialmente en useEffect al montar
 * - Suscribirse a actualizaciones de vehículos en tiempo real
 * - Mantener estado de carga y error
 * - Limpiar suscripción al desmontar
 * - Proporcionar función refetch para reintentos manuales
 *
 * Arquitectura:
 * - Accede al VehicleStateManager a través de React Context
 * - No conoce detalles de infraestructura (mock vs real, SSE, storage)
 * - Delega toda la lógica al coordinador de aplicación (VehicleStateManager)
 * - Maneja únicamente estado local del componente y ciclo de vida React
 *
 * Flujo:
 * 1. Al montar: ejecuta VehicleStateManager.loadVehicles()
 * 2. Si éxito: actualiza state con vehículos cargados
 * 3. Si falla: actualiza state con error, mantiene vehículos vacíos
 * 4. Suscripción: registra callback en VehicleStateManager para actualizaciones en tiempo real
 * 5. Actualizaciones: cada vez que llegan vehículos actualizados, React re-renderiza componente
 * 6. Al desmontar: desuscripción automática mediante return de useEffect
 *
 * Uso en componentes:
 * ```typescript
 * const { vehicles, loading, error, refetch } = useVehicles();
 *
 * if (loading) return <LoadingSpinner />;
 * if (error) return <ErrorAlert error={error} onDismiss={refetch} />;
 *
 * return (
 *   <MapContainer vehicles={vehicles} />
 * );
 * ```
 *
 * Verifica: Requisito 6 (hooks reutilizables), Requisito 1.6 (offline-first)
 *
 * @returns UseVehiclesResult con vehículos, estado de carga/error y función refetch
 * @throws No lanza excepciones; errores se capturan y almacenan en state.error
 */
export const useVehicles = (): UseVehiclesResult => {
  // Obtener VehicleStateManager del contexto
  const applicationContext = useContext(ApplicationContext);
  if (!applicationContext) {
    throw new Error('useVehicles must be used within ApplicationProvider');
  }
  const vehicleStateManager = applicationContext.vehicleStateManager;

  // Estados locales del hook
  const [vehicles, setVehicles] = useState<Vehicle[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  /**
   * Función interna para ejecutar la carga de vehículos.
   * Se usa tanto en el efecto inicial como en refetch.
   */
  const loadVehicles = async (): Promise<void> => {
    try {
      setLoading(true);
      setError(null);

      // Ejecutar caso de uso a través del coordinador de aplicación
      const loaded = await vehicleStateManager.loadVehicles();
      setVehicles(loaded);
    } catch (err) {
      // Convertir error a instancia de Error si es necesario
      const error = err instanceof Error ? err : new Error(String(err));
      setError(error);
      // Mantener vehículos vacíos en caso de error en carga inicial
      setVehicles([]);
    } finally {
      setLoading(false);
    }
  };

  /**
   * Efecto 1: Carga inicial de vehículos cuando el componente se monta.
   *
   * Se ejecuta una sola vez al montar (array de dependencias vacío).
   * Limpia cualquier estado anterior y realiza carga fresca.
   *
   * Verifica: Requisito 1.6 (carga inicial con fallback offline)
   */
  useEffect(() => {
    loadVehicles();
  }, [vehicleStateManager]);

  /**
   * Efecto 2: Suscripción a actualizaciones en tiempo real de vehículos.
   *
   * Registra un callback que se invoca cada vez que hay actualización de vehículos.
   * Las actualizaciones pueden venir de:
   * - Carga inicial
   * - Eventos SSE de actualización de posición
   * - Reconexión después de error
   *
   * El return de useEffect desuscribe automáticamente al desmontar.
   * Esto previene memory leaks (callbacks en listas de VehicleStateManager).
   *
   * Verifica: Requisito 3.3 (actualizaciones en tiempo real)
   */
  useEffect(() => {
    // Callback que se invoca cuando VehicleStateManager tiene nuevos vehículos
    const handleVehicleUpdate = (updatedVehicles: Vehicle[]) => {
      setVehicles(updatedVehicles);
      setError(null); // Limpiar error si había uno previo
    };

    // Suscribirse a actualizaciones globales de vehículos
    const unsubscribe = vehicleStateManager.subscribeToUpdates(handleVehicleUpdate);

    // Retornar función de cleanup que se ejecuta al desmontar o cuando cambia vehicleStateManager
    return () => {
      unsubscribe();
    };
  }, [vehicleStateManager]);

  /**
   * Función refetch que permite al componente consumidor reintentar carga manual.
   *
   * Útil para UX: botón "Reintentar" en caso de error, o refresh manual.
   *
   * Verifica: Requisito 7.1 (permitir reintento manual después de error)
   */
  const refetch = async (): Promise<void> => {
    await loadVehicles();
  };

  return {
    vehicles,
    loading,
    error,
    refetch,
  };
};
