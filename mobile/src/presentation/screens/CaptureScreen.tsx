import React, { useState } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  StyleSheet,
  StatusBar,
  Alert,
} from 'react-native';
import { DependencyContainer } from '../../infrastructure/di/DependencyContainer';
import { PositionsList } from '../components/PositionsList';
import {
  useAutoSync,
  useSyncOnReconnect,
  usePositionsList,
  useConnectivity,
  useLocationPermissions,
} from '../hooks';

export interface CaptureScreenProps {
  container: DependencyContainer;
}

/**
 * CaptureScreen
 *
 * Main screen for GPS capture and position synchronization.
 * Integrates:
 * - Location permissions
 * - Connectivity detection
 * - Automatic background sync
 * - Manual sync on reconnect
 * - Position list display
 */
export const CaptureScreen: React.FC<CaptureScreenProps> = ({ container }) => {
  const factory = container.getServiceFactory();
  const [isCaptureActive, setIsCaptureActive] = useState(false);
  const [isSyncing, setIsSyncing] = useState(false);

  // Use cases
  const captureUseCase = factory.createCapturePositionUseCase();
  const startCaptureUseCase = factory.createStartCaptureUseCase();
  const stopCaptureUseCase = factory.createStopCaptureUseCase();
  const syncPendingUseCase = factory.createSyncPendingPositionsUseCase();

  // Hooks
  const { hasPermission, isLoading: permLoading, requestPermission } =
    useLocationPermissions(container.getLocationProvider());

  const { isOnline } = useConnectivity(container.getConnectivityProvider());

  const { positions, statusCounts, isLoading, refresh } = usePositionsList(
    factory.createGetCapturedPositionsUseCase(),
    container.getPersistenceProvider()
  );

  // Auto sync every 10s
  useAutoSync(syncPendingUseCase, isOnline);

  // Sync on reconnect
  useSyncOnReconnect(container.getConnectivityProvider(), syncPendingUseCase);

  const handleStartCapture = async () => {
    if (!hasPermission) {
      const granted = await requestPermission();
      if (!granted) {
        Alert.alert(
          'Permiso requerido',
          'Se requiere acceso a la ubicación para capturar coordenadas'
        );
        return;
      }
    }

    try {
      startCaptureUseCase.execute();
      setIsCaptureActive(true);
    } catch (error) {
      Alert.alert('Error', 'No se pudo iniciar la captura de GPS');
    }
  };

  const handleStopCapture = () => {
    stopCaptureUseCase.execute();
    setIsCaptureActive(false);
  };

  const handleSyncPosition = async (position: any) => {
    const syncUseCase = factory.createSyncPositionUseCase();
    try {
      const result = await syncUseCase.execute(position);
      if (!result.success) {
        Alert.alert('Error', result.error || 'No se pudo sincronizar la posición');
      }
    } catch (error) {
      Alert.alert('Error', 'Ocurrió un error durante la sincronización');
    }
  };

  const handleRetryAll = async () => {
    setIsSyncing(true);
    try {
      await syncPendingUseCase.execute();
      await refresh();
    } catch (error) {
      Alert.alert('Error', 'Error al sincronizar posiciones');
    } finally {
      setIsSyncing(false);
    }
  };

  if (permLoading) {
    return (
      <View style={styles.container}>
        <Text>Cargando permisos...</Text>
      </View>
    );
  }

  return (
    <View style={styles.container}>
      <StatusBar barStyle="dark-content" backgroundColor="#f5f5f5" />

      {/* Header with capture controls */}
      <View style={styles.header}>
        <Text style={styles.title}>Captura de GPS</Text>
        <TouchableOpacity
          style={[
            styles.captureButton,
            isCaptureActive && styles.captureButtonActive,
          ]}
          onPress={isCaptureActive ? handleStopCapture : handleStartCapture}
        >
          <Text style={styles.captureButtonText}>
            {isCaptureActive ? '⊟ Detener' : '◉ Iniciar'}
          </Text>
        </TouchableOpacity>
      </View>

      {/* Status indicator */}
      <View style={styles.statusBar}>
        <Text style={styles.statusText}>
          {isCaptureActive ? '✓ Captura activa' : '○ Captura inactiva'}
        </Text>
        <Text style={[styles.statusText, { color: isOnline ? '#34C759' : '#ff3333' }]}>
          {isOnline ? '📡 En línea' : '📡 Sin conexión'}
        </Text>
      </View>

      {/* Positions list */}
      <PositionsList
        positions={positions}
        statusCounts={statusCounts}
        isOnline={isOnline}
        isLoading={isLoading}
        isRetrying={isSyncing}
        onSyncPosition={handleSyncPosition}
        onRetryAll={handleRetryAll}
        onRefresh={refresh}
      />
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: 'white',
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingHorizontal: 16,
    paddingVertical: 12,
    backgroundColor: '#f5f5f5',
    borderBottomWidth: 1,
    borderBottomColor: '#E0E0E0',
  },
  title: {
    fontSize: 18,
    fontWeight: 'bold',
    color: '#333',
  },
  captureButton: {
    backgroundColor: '#007AFF',
    paddingHorizontal: 16,
    paddingVertical: 8,
    borderRadius: 6,
  },
  captureButtonActive: {
    backgroundColor: '#34C759',
  },
  captureButtonText: {
    fontSize: 14,
    fontWeight: '600',
    color: 'white',
  },
  statusBar: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingVertical: 8,
    backgroundColor: '#f9f9f9',
    borderBottomWidth: 1,
    borderBottomColor: '#E0E0E0',
  },
  statusText: {
    fontSize: 12,
    color: '#666',
  },
});
