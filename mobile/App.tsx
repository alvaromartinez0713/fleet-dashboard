import React, { useState, useEffect, useRef } from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  Button,
  FlatList,
  ActivityIndicator,
  Alert,
} from 'react-native';
import { setupDependencies } from './src/infrastructure/di/setupDependencies';
import { DependencyContainer } from './src/infrastructure/di/DependencyContainer';

interface SimplePosition {
  eventId: string;
  vehicleId: string;
  status: 'pending' | 'sent' | 'error' | 'validation_error';
  latitude: number;
  longitude: number;
  accuracy: number;
  capturedAt: string;
  syncAttemptCount: number;
}

interface StatusCounts {
  pending: number;
  sent: number;
  error: number;
  validation_error: number;
}

export default function App() {
  const [container, setContainer] = useState<DependencyContainer | null>(null);
  const [isCapturing, setIsCapturing] = useState(false);
  const [isSyncing, setIsSyncing] = useState(false);
  const [isOnline, setIsOnline] = useState(true);
  const [positions, setPositions] = useState<SimplePosition[]>([]);
  const [statusCounts, setStatusCounts] = useState<StatusCounts>({
    pending: 0,
    sent: 0,
    error: 0,
    validation_error: 0,
  });
  const [initialized, setInitialized] = useState(false);
  const [dbSize, setDbSize] = useState('');

  const captureIntervalRef = useRef<NodeJS.Timeout | null>(null);
  const captureCountRef = useRef(0);

  // Initialize database on mount
  useEffect(() => {
    const initializeApp = async () => {
      try {
        console.log('[App] Initializing app...');
        const deps = await setupDependencies();
        setContainer(deps);

        // Load initial positions from SQLite
        const persistenceProvider = deps.getPersistenceProvider();
        const positions = await persistenceProvider.getAllPositions();
        
        const simplePositions: SimplePosition[] = positions.map(p => ({
          eventId: p.eventId,
          vehicleId: p.vehicleId,
          status: p.status,
          latitude: p.coordinate.latitude,
          longitude: p.coordinate.longitude,
          accuracy: p.coordinate.accuracy,
          capturedAt: p.capturedAt.toISOString(),
          syncAttemptCount: p.syncAttemptCount,
        }));

        setPositions(simplePositions);
        updateCounts(simplePositions);

        // Calculate DB size estimate
        const count = simplePositions.length;
        const estimatedSize = (count * 0.2).toFixed(2); // ~200 bytes per record
        setDbSize(`${count} registros (~${estimatedSize} KB)`);

        console.log(`[App] Loaded ${count} positions from SQLite`);
        setInitialized(true);
      } catch (error) {
        console.error('[App] Initialization failed:', error);
        setInitialized(true);
      }
    };

    initializeApp();
  }, []);

  const updateCounts = (positionsList: SimplePosition[]) => {
    const counts: StatusCounts = {
      pending: 0,
      sent: 0,
      error: 0,
      validation_error: 0,
    };
    positionsList.forEach(p => {
      counts[p.status]++;
    });
    setStatusCounts(counts);
  };

  const handleStartCapture = () => {
    if (isCapturing) {
      handleStopCapture();
      return;
    }

    if (!container) {
      Alert.alert('Error', 'App no está inicializado');
      return;
    }

    console.log('[App] Starting capture...');
    setIsCapturing(true);
    captureCountRef.current = 0;

    // Add first position immediately
    addCapturedPosition();

    // Then add every 30 seconds
    captureIntervalRef.current = setInterval(() => {
      addCapturedPosition();
    }, 30000);

    Alert.alert('✓ Captura iniciada', 'Capturando posiciones cada 30 segundos');
  };

  const addCapturedPosition = async () => {
    if (!container) return;

    captureCountRef.current++;
    console.log(`[App] Capturing position #${captureCountRef.current}...`);

    try {
      const persistenceProvider = container.getPersistenceProvider();
      const clock = container.getClock();
      const idGenerator = container.getIDGenerator();

      // Create position object matching the domain model
      const newPosition = {
        eventId: idGenerator.generateUUID(),
        vehicleId: 'VH-001',
        coordinate: {
          latitude: 40.7128 + (Math.random() - 0.5) * 0.01,
          longitude: -74.006 + (Math.random() - 0.5) * 0.01,
          accuracy: 5 + Math.random() * 15,
        },
        capturedAt: clock.now(),
        traceContext: {
          traceId: idGenerator.generateUUID(),
          idempotencyKey: idGenerator.generateUUID(),
        },
        status: 'pending' as const,
        lastSyncAttempt: undefined,
        syncAttemptCount: 0,
        nextRetryAt: undefined,
        updatedAt: clock.now(),
        errorReason: undefined,
        isValidationError: false,
      };

      // Save to SQLite
      await persistenceProvider.savePosition(newPosition);

      // Update UI
      const simplePosition: SimplePosition = {
        eventId: newPosition.eventId,
        vehicleId: newPosition.vehicleId,
        status: newPosition.status,
        latitude: newPosition.coordinate.latitude,
        longitude: newPosition.coordinate.longitude,
        accuracy: newPosition.coordinate.accuracy,
        capturedAt: newPosition.capturedAt.toISOString(),
        syncAttemptCount: newPosition.syncAttemptCount,
      };

      setPositions(prev => {
        const updated = [simplePosition, ...prev];
        updateCounts(updated);
        
        // Update DB size
        const estimatedSize = (updated.length * 0.2).toFixed(2);
        setDbSize(`${updated.length} registros (~${estimatedSize} KB)`);
        
        return updated;
      });
    } catch (error) {
      console.error('[App] Error adding position:', error);
    }
  };

  const handleStopCapture = () => {
    if (captureIntervalRef.current) {
      clearInterval(captureIntervalRef.current);
      captureIntervalRef.current = null;
    }
    setIsCapturing(false);
    console.log('[App] Capture paused');
    Alert.alert('⏸ Captura pausada', 'Los datos están guardados en SQLite');
  };

  const handleSync = async () => {
    if (!isOnline) {
      Alert.alert('🔴 Sin conexión', 'La app está en modo offline');
      return;
    }

    if (!container) return;

    setIsSyncing(true);
    console.log('[App] Syncing positions...');

    try {
      const persistenceProvider = container.getPersistenceProvider();
      const pendingCount = positions.filter(p => p.status === 'pending').length;

      // Update all pending positions to sent
      for (const pos of positions) {
        if (pos.status === 'pending') {
          await persistenceProvider.updatePositionStatus(pos.eventId, 'sent');
        }
      }

      // Update UI
      setPositions(prev => {
        const updated = prev.map(p =>
          p.status === 'pending' ? { ...p, status: 'sent' as const, syncAttemptCount: p.syncAttemptCount + 1 } : p
        );
        updateCounts(updated);
        return updated;
      });

      Alert.alert('✓ Sincronización completa', `${pendingCount} posiciones enviadas`);
      console.log('[App] Sync complete');
    } catch (error) {
      console.error('[App] Sync error:', error);
      Alert.alert('Error en sincronización', String(error));
    } finally {
      setIsSyncing(false);
    }
  };

  const handleRetryPosition = async (eventId: string) => {
    if (!container) return;

    console.log(`[App] Retrying position ${eventId}`);

    try {
      const persistenceProvider = container.getPersistenceProvider();
      await persistenceProvider.updatePositionStatus(eventId, 'sent');

      setPositions(prev => {
        const updated = prev.map(p =>
          p.eventId === eventId ? { ...p, status: 'sent' as const, syncAttemptCount: p.syncAttemptCount + 1 } : p
        );
        updateCounts(updated);
        return updated;
      });
    } catch (error) {
      console.error('[App] Retry error:', error);
    }
  };


  if (!initialized) {
    return (
      <View style={styles.centerContainer}>
        <ActivityIndicator size="large" color="#007AFF" />
        <Text style={styles.loadingText}>Inicializando SQLite...</Text>
      </View>
    );
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'sent':
        return '#34C759';
      case 'error':
        return '#FF3B30';
      case 'validation_error':
        return '#FF9500';
      default:
        return '#007AFF';
    }
  };

  const getStatusLabel = (status: string) => {
    switch (status) {
      case 'pending':
        return 'Pendiente';
      case 'sent':
        return 'Enviado';
      case 'error':
        return 'Error';
      case 'validation_error':
        return 'Error de datos';
      default:
        return status;
    }
  };

  const renderPositionItem = ({ item }: { item: SimplePosition }) => (
    <View style={styles.positionCard}>
      <View style={styles.cardHeader}>
        <Text style={styles.cardTitle}>Evento: {item.eventId.slice(0, 12)}</Text>
        <Text style={[styles.statusBadge, { backgroundColor: getStatusColor(item.status) }]}>
          {getStatusLabel(item.status)}
        </Text>
      </View>
      <Text style={styles.cardText}>Vehículo: {item.vehicleId}</Text>
      <Text style={styles.cardText}>
        📍 {item.latitude.toFixed(4)}, {item.longitude.toFixed(4)}
      </Text>
      <Text style={styles.cardText}>Precisión: {item.accuracy.toFixed(2)}m</Text>
      <Text style={styles.cardText}>Intentos: {item.syncAttemptCount}</Text>
      <Text style={styles.cardText}>{new Date(item.capturedAt).toLocaleTimeString()}</Text>
      {isOnline && item.status !== 'sent' && item.status !== 'validation_error' && (
        <View style={styles.retryButton}>
          <Button
            title="Reintentar"
            onPress={() => handleRetryPosition(item.eventId)}
            color="#007AFF"
          />
        </View>
      )}
    </View>
  );

  return (
    <View style={styles.container}>
      <ScrollView style={styles.scrollView}>
        {/* Header */}
        <View style={styles.header}>
          <Text style={styles.appTitle}>Fleet Mobile</Text>
          <Text style={styles.appSubtitle}>GPS Capture & Sync</Text>
        </View>

        {/* Storage Info */}
        {dbSize && (
          <View style={styles.storageInfo}>
            <Text style={styles.storageText}>
              💾 SQLite: {dbSize}
            </Text>
          </View>
        )}

        {/* Connectivity Status */}
        <View style={[styles.connectivityBanner, { backgroundColor: isOnline ? '#34C759' : '#FF3B30' }]}>
          <Text style={styles.connectivityText}>
            {isOnline ? '🟢 Conectado' : '🔴 Sin conexión'}
          </Text>
        </View>

        {/* Capture Controls */}
        <View style={styles.controlsSection}>
          <Text style={styles.sectionTitle}>Control de Captura</Text>
          <View style={styles.buttonRow}>
            <View style={[styles.flexButton, { marginRight: 5 }]}>
              <Button
                title={isCapturing ? '⏸ Pausar' : '▶ Iniciar'}
                onPress={handleStartCapture}
                color={isCapturing ? '#FF3B30' : '#34C759'}
              />
            </View>
            <View style={[styles.flexButton, { marginLeft: 5 }]}>
              <Button
                title={isSyncing ? '↻ Sincronizando...' : '↻ Sincronizar'}
                onPress={handleSync}
                color="#007AFF"
                disabled={!isOnline || isSyncing}
              />
            </View>
          </View>
          {isCapturing && (
            <Text style={styles.capturingIndicator}>✓ Captura activa cada 30 segundos</Text>
          )}
        </View>

        {/* Stats */}
        <View style={styles.statsBox}>
          <Text style={styles.statsTitle}>Estado del Sistema</Text>
          <View style={styles.statsGrid}>
            <View style={styles.statItem}>
              <Text style={styles.statNumber}>{positions.length}</Text>
              <Text style={styles.statLabel}>Total</Text>
            </View>
            <View style={styles.statItem}>
              <Text style={[styles.statNumber, { color: '#007AFF' }]}>
                {statusCounts.pending}
              </Text>
              <Text style={styles.statLabel}>Pendientes</Text>
            </View>
            <View style={styles.statItem}>
              <Text style={[styles.statNumber, { color: '#34C759' }]}>
                {statusCounts.sent}
              </Text>
              <Text style={styles.statLabel}>Enviadas</Text>
            </View>
            <View style={styles.statItem}>
              <Text style={[styles.statNumber, { color: '#FF3B30' }]}>
                {statusCounts.error}
              </Text>
              <Text style={styles.statLabel}>Errores</Text>
            </View>
          </View>
        </View>

        {/* Positions List */}
        <View style={styles.listContainer}>
          <View style={styles.listHeader}>
            <Text style={styles.listTitle}>Posiciones (más reciente primero)</Text>
          </View>
          {positions.length === 0 ? (
            <Text style={styles.emptyText}>
              No hay posiciones. Presiona "▶ Iniciar" para comenzar.
            </Text>
          ) : (
            <FlatList
              data={positions}
              renderItem={renderPositionItem}
              keyExtractor={(item) => item.eventId}
              scrollEnabled={false}
            />
          )}
        </View>
      </ScrollView>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#f8f9fa',
  },
  centerContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: '#f8f9fa',
  },
  loadingText: {
    marginTop: 10,
    fontSize: 16,
    color: '#666',
  },
  scrollView: {
    flex: 1,
  },
  header: {
    backgroundColor: '#007AFF',
    padding: 20,
    alignItems: 'center',
    marginBottom: 10,
  },
  appTitle: {
    fontSize: 28,
    fontWeight: 'bold',
    color: '#fff',
  },
  appSubtitle: {
    fontSize: 14,
    color: '#e8e8e8',
    marginTop: 5,
  },
  storageInfo: {
    backgroundColor: '#E8F5E9',
    padding: 10,
    marginHorizontal: 15,
    borderRadius: 8,
    marginBottom: 10,
    borderLeftWidth: 4,
    borderLeftColor: '#4CAF50',
  },
  storageText: {
    fontSize: 13,
    color: '#2E7D32',
    fontWeight: '600',
  },
  connectivityBanner: {
    padding: 10,
    alignItems: 'center',
    marginBottom: 15,
  },
  connectivityText: {
    color: '#fff',
    fontSize: 14,
    fontWeight: 'bold',
  },
  controlsSection: {
    backgroundColor: '#fff',
    padding: 15,
    marginHorizontal: 15,
    borderRadius: 8,
    marginBottom: 15,
    borderLeftWidth: 4,
    borderLeftColor: '#34C759',
  },
  sectionTitle: {
    fontSize: 16,
    fontWeight: 'bold',
    marginBottom: 12,
    color: '#333',
  },
  buttonRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
  },
  flexButton: {
    flex: 1,
  },
  capturingIndicator: {
    marginTop: 12,
    fontSize: 13,
    color: '#34C759',
    fontWeight: '600',
  },
  statsBox: {
    backgroundColor: '#fff',
    padding: 15,
    marginHorizontal: 15,
    borderRadius: 8,
    marginBottom: 15,
    borderLeftWidth: 4,
    borderLeftColor: '#007AFF',
  },
  statsTitle: {
    fontSize: 16,
    fontWeight: 'bold',
    marginBottom: 12,
    color: '#333',
  },
  statsGrid: {
    flexDirection: 'row',
    justifyContent: 'space-around',
  },
  statItem: {
    alignItems: 'center',
  },
  statNumber: {
    fontSize: 24,
    fontWeight: 'bold',
    color: '#333',
  },
  statLabel: {
    fontSize: 12,
    color: '#999',
    marginTop: 4,
  },
  listContainer: {
    marginHorizontal: 15,
    marginBottom: 20,
  },
  listHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 10,
  },
  listTitle: {
    fontSize: 16,
    fontWeight: 'bold',
    color: '#333',
    flex: 1,
  },
  positionCard: {
    backgroundColor: '#fff',
    padding: 12,
    marginBottom: 10,
    borderRadius: 8,
    borderLeftWidth: 3,
    borderLeftColor: '#007AFF',
  },
  cardHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 8,
  },
  cardTitle: {
    fontSize: 14,
    fontWeight: 'bold',
    color: '#007AFF',
  },
  statusBadge: {
    paddingHorizontal: 8,
    paddingVertical: 4,
    borderRadius: 4,
    color: '#fff',
    fontSize: 11,
    fontWeight: 'bold',
  },
  cardText: {
    fontSize: 12,
    color: '#666',
    marginVertical: 2,
  },
  retryButton: {
    marginTop: 8,
  },
  emptyText: {
    textAlign: 'center',
    color: '#999',
    fontSize: 14,
    marginVertical: 20,
  },
});
