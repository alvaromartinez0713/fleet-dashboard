import React, { useEffect, useState } from 'react';
import {
  View,
  FlatList,
  StyleSheet,
  ActivityIndicator,
  Text,
  RefreshControl,
} from 'react-native';
import { Position } from '../../domain/models/Position';
import { PositionItem } from './PositionItem';
import { PositionListHeader } from './PositionListHeader';

export interface PositionsListProps {
  positions: Position[];
  statusCounts: Record<string, number>;
  isOnline: boolean;
  isLoading?: boolean;
  isRetrying?: boolean;
  onSyncPosition?: (position: Position) => void;
  onRetryAll?: () => void;
  onRefresh?: () => Promise<void>;
}

/**
 * PositionsList
 *
 * Displays list of captured positions with header showing status breakdown.
 */
export const PositionsList: React.FC<PositionsListProps> = ({
  positions,
  statusCounts,
  isOnline,
  isLoading = false,
  isRetrying = false,
  onSyncPosition,
  onRetryAll,
  onRefresh,
}) => {
  const [refreshing, setRefreshing] = useState(false);
  const [now, setNow] = useState(new Date());

  // Update "now" every second for retry countdowns
  useEffect(() => {
    const interval = setInterval(() => {
      setNow(new Date());
    }, 1000);

    return () => clearInterval(interval);
  }, []);

  const handleRefresh = async () => {
    if (onRefresh) {
      setRefreshing(true);
      try {
        await onRefresh();
      } finally {
        setRefreshing(false);
      }
    }
  };

  return (
    <View style={styles.container}>
      <PositionListHeader
        pending={statusCounts['pending'] || 0}
        sent={statusCounts['sent'] || 0}
        error={statusCounts['error'] || 0}
        validationError={statusCounts['validation_error'] || 0}
        isOnline={isOnline}
        isRetrying={isRetrying}
        onRetryAll={onRetryAll}
      />

      {isLoading && positions.length === 0 ? (
        <View style={styles.centerLoader}>
          <ActivityIndicator size="large" color="#0066cc" />
          <Text style={styles.loadingText}>Cargando posiciones...</Text>
        </View>
      ) : positions.length === 0 ? (
        <View style={styles.emptyContainer}>
          <Text style={styles.emptyText}>No hay posiciones capturadas</Text>
        </View>
      ) : (
        <FlatList
          data={positions}
          keyExtractor={item => item.eventId}
          renderItem={({ item }) => (
            <PositionItem
              position={item}
              onSync={onSyncPosition}
              isOnline={isOnline}
              now={now}
            />
          )}
          refreshControl={
            onRefresh ? (
              <RefreshControl refreshing={refreshing} onRefresh={handleRefresh} />
            ) : undefined
          }
          scrollEnabled={true}
        />
      )}
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: 'white',
  },
  centerLoader: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  loadingText: {
    marginTop: 12,
    fontSize: 14,
    color: '#666',
  },
  emptyContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  emptyText: {
    fontSize: 16,
    color: '#999',
  },
});
