import React from 'react';
import { View, Text, TouchableOpacity, StyleSheet } from 'react-native';

export interface PositionListHeaderProps {
  pending: number;
  sent: number;
  error: number;
  validationError: number;
  isOnline: boolean;
  isRetrying?: boolean;
  onRetryAll?: () => void;
}

/**
 * PositionListHeader
 *
 * Shows status breakdown and "Reintentar todo" button.
 */
export const PositionListHeader: React.FC<PositionListHeaderProps> = ({
  pending,
  sent,
  error,
  validationError,
  isOnline,
  isRetrying = false,
  onRetryAll,
}) => {
  const retryableCount = pending + error;
  const canRetry = isOnline && retryableCount > 0;

  return (
    <View style={styles.container}>
      <View style={styles.statusRow}>
        <StatusBadge label="Pendiente" count={pending} />
        <StatusBadge label="Enviado" count={sent} />
        <StatusBadge label="Error" count={error} color="#ff3333" />
        <StatusBadge label="Datos inválidos" count={validationError} color="#cc00cc" />
      </View>

      {retryableCount > 0 && (
        <TouchableOpacity
          style={[styles.retryButton, !canRetry && styles.retryButtonDisabled]}
          onPress={onRetryAll}
          disabled={!canRetry || isRetrying}
        >
          <Text style={[styles.retryButtonText, !canRetry && styles.retryButtonTextDisabled]}>
            Sincronizar todo ({retryableCount})
          </Text>
        </TouchableOpacity>
      )}

      {!isOnline && <Text style={styles.offlineWarning}>📡 Sin conexión</Text>}
    </View>
  );
};

interface StatusBadgeProps {
  label: string;
  count: number;
  color?: string;
}

const StatusBadge: React.FC<StatusBadgeProps> = ({ label, count, color = '#0066cc' }) => (
  <View style={styles.badge}>
    <Text style={styles.badgeLabel}>{label}</Text>
    <Text style={[styles.badgeCount, { color }]}>{count}</Text>
  </View>
);

const styles = StyleSheet.create({
  container: {
    paddingVertical: 16,
    paddingHorizontal: 16,
    backgroundColor: '#f5f5f5',
  },
  statusRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginBottom: 12,
  },
  badge: {
    alignItems: 'center',
    flex: 1,
  },
  badgeLabel: {
    fontSize: 11,
    color: '#666',
    marginBottom: 2,
  },
  badgeCount: {
    fontSize: 18,
    fontWeight: 'bold',
  },
  retryButton: {
    backgroundColor: '#34C759',
    paddingVertical: 10,
    paddingHorizontal: 16,
    borderRadius: 6,
    alignItems: 'center',
  },
  retryButtonDisabled: {
    backgroundColor: '#CCCCCC',
  },
  retryButtonText: {
    fontSize: 14,
    fontWeight: '600',
    color: 'white',
  },
  retryButtonTextDisabled: {
    color: '#999',
  },
  offlineWarning: {
    fontSize: 12,
    color: '#ff3333',
    marginTop: 12,
    textAlign: 'center',
    fontWeight: '500',
  },
});
