import React, { useEffect, useState } from 'react';
import { ConnectionStatus as ConnectionStatusEnum } from '@application/ports/vehicle-realtime-port';
import { useSSEConnection } from '../hooks/useSSEConnection';

/**
 * ConnectionStatusProps - Props for the ConnectionStatus component.
 *
 * @property showLabel - Whether to show the status label text (default: true)
 * @property showCountdown - Whether to show reconnection countdown (default: true)
 * @property onRetry - Optional callback when retry button is clicked
 */
interface ConnectionStatusProps {
  showLabel?: boolean;
  showCountdown?: boolean;
  onRetry?: () => void;
}

/**
 * StatusIndicatorColor - Color coding for connection status.
 */
type StatusIndicatorColor = 'green' | 'yellow' | 'red';

/**
 * StatusInfo - Information about a connection status.
 */
interface StatusInfo {
  label: string;
  color: StatusIndicatorColor;
  ariaLabel: string;
}

/**
 * ConnectionStatus - React component that displays SSE connection status indicator.
 *
 * Features:
 * - Color-coded status indicator (green for connected, yellow for reconnecting, red for disconnected/error)
 * - Status text label
 * - Reconnection countdown timer when reconnecting
 * - Manual retry button when disconnected or in error state
 * - Accessible with ARIA labels and roles
 * - Responsive and compact design suitable for header/toolbar placement
 *
 * Connection Status Mapping:
 * - CONNECTED: Green indicator, "Conectado" label
 * - DISCONNECTED: Red indicator, "Desconectado" label
 * - RECONNECTING: Yellow indicator, "Reconectando" label with countdown
 * - ERROR: Red indicator, "Error de conexión" label with retry button
 *
 * Usage:
 * ```typescript
 * // In dashboard header
 * <ConnectionStatus showLabel={true} showCountdown={true} />
 *
 * // Minimal version (just indicator)
 * <ConnectionStatus showLabel={false} />
 * ```
 *
 * Verifies: RF-07 (dashboard reactivo), RNF-08 (actualización reactiva)
 */
export const ConnectionStatus: React.FC<ConnectionStatusProps> = ({
  showLabel = true,
  showCountdown = true,
  onRetry,
}) => {
  // Get connection status from hook
  const { status, error } = useSSEConnection();

  // State for reconnection countdown timer
  const [countdownSeconds, setCountdownSeconds] = useState(0);
  const [isRetrying, setIsRetrying] = useState(false);

  /**
   * Map connection status to display information.
   */
  const getStatusInfo = (connectionStatus: ConnectionStatusEnum): StatusInfo => {
    switch (connectionStatus) {
      case ConnectionStatusEnum.CONNECTED:
        return {
          label: 'Conectado',
          color: 'green',
          ariaLabel: 'Conexión establida al servidor de actualizaciones en tiempo real',
        };
      case ConnectionStatusEnum.DISCONNECTED:
        return {
          label: 'Desconectado',
          color: 'red',
          ariaLabel: 'Desconectado del servidor de actualizaciones en tiempo real',
        };
      case ConnectionStatusEnum.RECONNECTING:
        return {
          label: 'Reconectando',
          color: 'yellow',
          ariaLabel: 'Intentando reconectarse al servidor de actualizaciones en tiempo real',
        };
      case ConnectionStatusEnum.ERROR:
        return {
          label: 'Error de conexión',
          color: 'red',
          ariaLabel: 'Error de conexión al servidor de actualizaciones en tiempo real',
        };
      default:
        return {
          label: 'Estado desconocido',
          color: 'red',
          ariaLabel: 'Estado desconocido de la conexión',
        };
    }
  };

  const statusInfo = getStatusInfo(status);

  /**
   * Effect: Manage reconnection countdown timer.
   *
   * When status is RECONNECTING, start a countdown from 30 seconds.
   * Timer decrements every second and stops when status changes.
   */
  useEffect(() => {
    if (status === ConnectionStatusEnum.RECONNECTING) {
      setCountdownSeconds(30); // Start with 30 second countdown

      const timer = setInterval(() => {
        setCountdownSeconds((prev) => {
          if (prev <= 1) {
            return 0;
          }
          return prev - 1;
        });
      }, 1000);

      return () => clearInterval(timer);
    } else {
      setCountdownSeconds(0);
    }
  }, [status]);

  /**
   * Handle manual retry button click.
   *
   * Triggers optional onRetry callback and sets isRetrying flag
   * for visual feedback during retry attempt.
   */
  const handleRetry = () => {
    setIsRetrying(true);
    if (onRetry) {
      onRetry();
    }
    // Reset isRetrying after visual feedback delay
    setTimeout(() => setIsRetrying(false), 1500);
  };

  /**
   * Get indicator color CSS value.
   */
  const getIndicatorColor = (color: StatusIndicatorColor): string => {
    switch (color) {
      case 'green':
        return '#10b981'; // emerald-500
      case 'yellow':
        return '#f59e0b'; // amber-500
      case 'red':
        return '#ef4444'; // red-500
      default:
        return '#6b7280'; // gray-500
    }
  };

  /**
   * Render indicator dot.
   */
  const renderIndicator = () => (
    <div
      style={{
        display: 'inline-block',
        width: '12px',
        height: '12px',
        borderRadius: '50%',
        backgroundColor: getIndicatorColor(statusInfo.color),
        marginRight: '8px',
        animation:
          status === ConnectionStatusEnum.RECONNECTING
            ? 'pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite'
            : 'none',
      }}
      aria-hidden="true"
      data-testid="connection-status-indicator"
    />
  );

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: '8px',
        padding: '8px 12px',
        backgroundColor: '#f9fafb', // gray-50
        borderRadius: '6px',
        border: '1px solid #e5e7eb', // gray-200
        fontSize: '14px',
        lineHeight: '1.5',
      }}
      role="status"
      aria-live="polite"
      aria-label={statusInfo.ariaLabel}
      data-testid="connection-status-component"
    >
      <style>
        {`
          @keyframes pulse {
            0%, 100% {
              opacity: 1;
            }
            50% {
              opacity: 0.5;
            }
          }
          
          .connection-status-button {
            background: none;
            border: none;
            color: #3b82f6;
            cursor: pointer;
            padding: 0;
            font-size: 14px;
            text-decoration: underline;
            line-height: 1.5;
          }
          
          .connection-status-button:hover {
            color: #1d4ed8;
            text-decoration: none;
          }
          
          .connection-status-button:disabled {
            color: #9ca3af;
            cursor: not-allowed;
            text-decoration: none;
          }
        `}
      </style>

      {/* Indicator dot */}
      {renderIndicator()}

      {/* Status label and countdown */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: '4px',
        }}
      >
        {showLabel && (
          <span
            style={{
              color: '#374151', // gray-700
              fontWeight: '500',
            }}
            data-testid="connection-status-label"
          >
            {statusInfo.label}
          </span>
        )}

        {/* Countdown timer for reconnecting state */}
        {showCountdown && status === ConnectionStatusEnum.RECONNECTING && countdownSeconds > 0 && (
          <span
            style={{
              color: '#6b7280', // gray-500
              fontSize: '12px',
            }}
            data-testid="connection-status-countdown"
          >
            ({countdownSeconds}s)
          </span>
        )}
      </div>

      {/* Retry button for disconnected/error states */}
      {(status === ConnectionStatusEnum.DISCONNECTED || status === ConnectionStatusEnum.ERROR) && (
        <button
          className="connection-status-button"
          onClick={handleRetry}
          disabled={isRetrying}
          aria-label="Reintentar conexión"
          data-testid="connection-status-retry-button"
          style={{
            marginLeft: 'auto',
          }}
        >
          {isRetrying ? 'Reintentando...' : 'Reintentar'}
        </button>
      )}

      {/* Error message */}
      {error && (
        <div
          style={{
            marginLeft: 'auto',
            fontSize: '12px',
            color: '#dc2626', // red-600
            title: error.message,
          }}
          data-testid="connection-status-error-message"
        >
          {error.message.substring(0, 50)}
          {error.message.length > 50 ? '...' : ''}
        </div>
      )}
    </div>
  );
};

export default ConnectionStatus;
