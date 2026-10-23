import React from 'react';
import { View, Text, TouchableOpacity, StyleSheet, ActivityIndicator } from 'react-native';
import { Position } from '../../domain/models/Position';

export interface PositionItemProps {
  position: Position;
  onSync?: (position: Position) => void;
  isOnline: boolean;
  isSyncing?: boolean;
  now?: Date;
}

/**
 * PositionItem
 *
 * Displays a single captured GPS position with status indicator and sync button.
 */
export const PositionItem: React.FC<PositionItemProps> = ({
  position,
  onSync,
  isOnline,
  isSyncing = false,
  now = new Date(),
}) => {
  // Determine if sync button should be enabled
  const canSync =
    isOnline && (position.status === 'pending' || position.canBeSynced(now));
  const isSyncDisabled =
    !isOnline || position.status === 'sent' || position.status === 'validation_error';

  const handleSync = () => {
    if (!isSyncDisabled && !isSyncing && onSync) {
      onSync(position);
    }
  };

  const coordText = `${position.coordinate.latitude.toFixed(4)}, ${position.coordinate.longitude.toFixed(4)}`;
  const timeText = position.capturedAt.toLocaleTimeString();
  const retryText = position.getNextRetryText(now);

  return (
    <View style={styles.container}>
      <View style={styles.content}>
        {/* Status icon */}
        <Text style={styles.statusIcon}>{position.getStatusIcon()}</Text>

        {/* Position info */}
        <View style={styles.info}>
          <Text style={styles.coord}>{coordText}</Text>
          <Text style={styles.time}>{timeText}</Text>
          <View style={styles.statusRow}>
            <Text style={styles.status}>{position.getStatusLabel()}</Text>
            {retryText && <Text style={styles.retry}>{retryText}</Text>}
          </View>
        </View>

        {/* Sync button */}
        <TouchableOpacity
          style={[styles.button, isSyncDisabled && styles.buttonDisabled]}
          onPress={handleSync}
          disabled={isSyncDisabled || isSyncing}
        >
          {isSyncing ? (
            <ActivityIndicator size="small" color="#007AFF" />
          ) : (
            <Text style={[styles.buttonText, isSyncDisabled && styles.buttonTextDisabled]}>
              {position.status === 'pending' ? 'Enviar' : 'Reintentar'}
            </Text>
          )}
        </TouchableOpacity>
      </View>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    borderBottomWidth: 1,
    borderBottomColor: '#E0E0E0',
    paddingVertical: 12,
    paddingHorizontal: 16,
  },
  content: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
  },
  statusIcon: {
    fontSize: 24,
  },
  info: {
    flex: 1,
  },
  coord: {
    fontSize: 14,
    fontWeight: '600',
    color: '#333',
  },
  time: {
    fontSize: 12,
    color: '#666',
    marginTop: 2,
  },
  statusRow: {
    flexDirection: 'row',
    alignItems: 'center',
    marginTop: 4,
    gap: 8,
  },
  status: {
    fontSize: 12,
    color: '#0066cc',
    fontWeight: '500',
  },
  retry: {
    fontSize: 11,
    color: '#ff9500',
    fontStyle: 'italic',
  },
  button: {
    backgroundColor: '#007AFF',
    paddingHorizontal: 12,
    paddingVertical: 8,
    borderRadius: 6,
  },
  buttonDisabled: {
    backgroundColor: '#CCCCCC',
  },
  buttonText: {
    fontSize: 12,
    fontWeight: '600',
    color: 'white',
  },
  buttonTextDisabled: {
    color: '#999',
  },
});
