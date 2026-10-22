import React, { useState } from 'react';

/**
 * ErrorType - Categorization of error types for visual styling
 */
type ErrorType = 'api' | 'network' | 'sse' | 'unknown';

/**
 * ErrorAlertProps - Props for the ErrorAlert component
 *
 * @property error - The error object to display, or null to hide the alert
 * @property onDismiss - Optional callback when the dismiss button is clicked
 * @property onRetry - Optional callback to trigger a retry action
 * @property showRetry - Whether to show the retry button (default: false)
 */
interface ErrorAlertProps {
  error: Error | null;
  onDismiss?: () => void;
  onRetry?: () => void;
  showRetry?: boolean;
}

/**
 * ErrorAlertTypeInfo - Information about error type for display
 */
interface ErrorTypeInfo {
  type: ErrorType;
  icon: string;
  color: string;
  backgroundColor: string;
  borderColor: string;
  textColor: string;
  ariaLabel: string;
}

/**
 * Categorize error based on message content
 *
 * Determines error type (API, network, SSE) based on error message patterns.
 * This allows visual differentiation and context-specific messaging.
 *
 * @param error - The error to categorize
 * @returns ErrorType indicating the category
 */
const categorizeError = (error: Error | null): ErrorType => {
  if (!error) return 'unknown';

  const message = error.message.toLowerCase();

  // Network errors: connection refused, timeout, offline, network error
  if (
    message.includes('network') ||
    message.includes('offline') ||
    message.includes('timeout') ||
    message.includes('refused') ||
    message.includes('no internet') ||
    message.includes('econnrefused')
  ) {
    return 'network';
  }

  // SSE-specific errors: connection, sse, stream, event source
  if (
    message.includes('sse') ||
    message.includes('stream') ||
    message.includes('event source') ||
    message.includes('reconect') ||
    message.includes('connection')
  ) {
    return 'sse';
  }

  // API errors: api, http, fetch, response, status
  if (
    message.includes('api') ||
    message.includes('http') ||
    message.includes('fetch') ||
    message.includes('response') ||
    message.includes('status')
  ) {
    return 'api';
  }

  // Default to unknown
  return 'unknown';
};

/**
 * Get error type info for display
 *
 * Returns styling and icon information based on error type.
 * Each type has a distinct visual appearance for user recognition.
 *
 * @param errorType - The error type
 * @returns ErrorTypeInfo with styling and icon
 */
const getErrorTypeInfo = (errorType: ErrorType): ErrorTypeInfo => {
  switch (errorType) {
    case 'network':
      return {
        type: 'network',
        icon: '📡', // network/wifi icon
        color: '#dc2626', // red-600
        backgroundColor: '#fee2e2', // red-100
        borderColor: '#fecaca', // red-200
        textColor: '#7f1d1d', // red-900
        ariaLabel: 'Error de conectividad de red',
      };

    case 'api':
      return {
        type: 'api',
        icon: '⚙️', // settings/API icon
        color: '#ea580c', // orange-600
        backgroundColor: '#fed7aa', // orange-100
        borderColor: '#fdba74', // orange-200
        textColor: '#7c2d12', // orange-900
        ariaLabel: 'Error de la API',
      };

    case 'sse':
      return {
        type: 'sse',
        icon: '🔌', // connection/plug icon
        color: '#d97706', // amber-600
        backgroundColor: '#fef3c7', // amber-100
        borderColor: '#fde68a', // amber-200
        textColor: '#78350f', // amber-900
        ariaLabel: 'Error de conexión en tiempo real',
      };

    case 'unknown':
    default:
      return {
        type: 'unknown',
        icon: '⚠️', // generic warning icon
        color: '#6366f1', // indigo-600
        backgroundColor: '#e0e7ff', // indigo-100
        borderColor: '#c7d2fe', // indigo-200
        textColor: '#312e81', // indigo-900
        ariaLabel: 'Error desconocido',
      };
  }
};

/**
 * ErrorAlert - React component that displays error messages with categorization
 *
 * Features:
 * - Error type categorization (API, network, SSE, unknown)
 * - Color-coded visual styling based on error type
 * - Type-specific icons for quick recognition
 * - Dismiss button to hide the alert
 * - Optional retry button for actionable recovery
 * - Readable error message display
 * - Accessible with ARIA labels and roles
 * - Responsive design suitable for header/alert placement
 *
 * Error Categorization:
 * - Network: Connection issues, offline, timeouts (red theme)
 * - API: HTTP/API errors, responses, status codes (orange theme)
 * - SSE: Real-time connection, stream, event source (amber theme)
 * - Unknown: Other errors (indigo theme)
 *
 * Usage:
 * ```typescript
 * // Basic error display with dismiss
 * <ErrorAlert
 *   error={error}
 *   onDismiss={() => setError(null)}
 * />
 *
 * // With retry functionality
 * <ErrorAlert
 *   error={error}
 *   onDismiss={() => setError(null)}
 *   onRetry={handleRetry}
 *   showRetry={true}
 * />
 *
 * // Hide when error is null
 * <ErrorAlert error={null} />
 * ```
 *
 * Verifies: RF-07 (error handling), RNF-06 (resiliencia)
 * Testing: caja negra - verifies observable behavior (what user sees)
 */
export const ErrorAlert: React.FC<ErrorAlertProps> = ({
  error,
  onDismiss,
  onRetry,
  showRetry = false,
}) => {
  // Do not render if no error
  if (!error) {
    return null;
  }

  // Determine error type and get styling info
  const errorType = categorizeError(error);
  const typeInfo = getErrorTypeInfo(errorType);

  // State for retry button feedback
  const [isRetrying, setIsRetrying] = useState(false);

  /**
   * Handle dismiss button click
   */
  const handleDismiss = () => {
    if (onDismiss) {
      onDismiss();
    }
  };

  /**
   * Handle retry button click with visual feedback
   */
  const handleRetry = () => {
    if (onRetry) {
      setIsRetrying(true);
      try {
        onRetry();
      } finally {
        // Reset retry state after a short delay for visual feedback
        setTimeout(() => setIsRetrying(false), 1500);
      }
    }
  };

  const containerStyles: React.CSSProperties = {
    backgroundColor: typeInfo.backgroundColor,
    border: `2px solid ${typeInfo.borderColor}`,
    borderRadius: '6px',
    padding: '16px',
    marginBottom: '16px',
    display: 'flex',
    alignItems: 'flex-start',
    gap: '12px',
    color: typeInfo.textColor,
  };

  const iconStyles: React.CSSProperties = {
    fontSize: '24px',
    minWidth: '24px',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    flexShrink: 0,
  };

  const contentStyles: React.CSSProperties = {
    flex: 1,
    display: 'flex',
    flexDirection: 'column',
    gap: '8px',
  };

  const messageStyles: React.CSSProperties = {
    margin: 0,
    fontSize: '14px',
    lineHeight: '1.5',
    wordBreak: 'break-word',
  };

  const titleStyles: React.CSSProperties = {
    margin: 0,
    fontSize: '15px',
    fontWeight: 600,
    lineHeight: '1.4',
  };

  const buttonsContainerStyles: React.CSSProperties = {
    display: 'flex',
    gap: '8px',
    alignItems: 'center',
    flexShrink: 0,
    marginLeft: '12px',
  };

  const buttonStyles: React.CSSProperties = {
    backgroundColor: 'transparent',
    border: `1px solid ${typeInfo.color}`,
    color: typeInfo.color,
    padding: '6px 12px',
    borderRadius: '4px',
    cursor: 'pointer',
    fontSize: '13px',
    fontWeight: 500,
    transition: 'all 0.2s ease',
    display: 'flex',
    alignItems: 'center',
    gap: '4px',
    whiteSpace: 'nowrap',
  };

  const dismissButtonStyles: React.CSSProperties = {
    ...buttonStyles,
    padding: '6px 8px',
    minWidth: 'auto',
  };

  const retryButtonStyles: React.CSSProperties = {
    ...buttonStyles,
  };

  /**
   * Get error message for display
   *
   * Extracts a readable message from the error object,
   * limiting length to prevent layout issues.
   */
  const getErrorMessage = (): string => {
    if (!error) return '';

    let message = error.message || 'Error desconocido';

    // Limit message length to prevent extreme line breaks
    if (message.length > 150) {
      message = message.substring(0, 150) + '...';
    }

    return message;
  };

  /**
   * Get error title based on type
   */
  const getErrorTitle = (): string => {
    switch (errorType) {
      case 'network':
        return 'Problema de conectividad';
      case 'api':
        return 'Error en la solicitud';
      case 'sse':
        return 'Error de conexión en tiempo real';
      default:
        return 'Se ha producido un error';
    }
  };

  return (
    <div
      style={containerStyles}
      role="alert"
      aria-live="assertive"
      aria-label={typeInfo.ariaLabel}
      data-testid="error-alert"
      data-error-type={errorType}
    >
      <style>
        {`
          .error-alert-button:hover:not(:disabled) {
            background-color: rgba(0, 0, 0, 0.05);
          }
          
          .error-alert-button:focus-visible {
            outline: 2px solid currentColor;
            outline-offset: 2px;
          }
          
          .error-alert-button:disabled {
            opacity: 0.6;
            cursor: not-allowed;
          }
        `}
      </style>

      {/* Error icon */}
      <div style={iconStyles} aria-hidden="true" data-testid="error-alert-icon">
        {typeInfo.icon}
      </div>

      {/* Error content */}
      <div style={contentStyles}>
        <h3 style={titleStyles} data-testid="error-alert-title">
          {getErrorTitle()}
        </h3>
        <p style={messageStyles} data-testid="error-alert-message">
          {getErrorMessage()}
        </p>
      </div>

      {/* Dismiss and retry buttons */}
      <div style={buttonsContainerStyles}>
        {/* Retry button (optional) */}
        {showRetry && onRetry && (
          <button
            className="error-alert-button"
            style={retryButtonStyles}
            onClick={handleRetry}
            disabled={isRetrying}
            aria-label="Reintentar la acción"
            title="Reintentar"
            data-testid="error-alert-retry-button"
          >
            {isRetrying ? '⏳ Reintentando...' : '🔄 Reintentar'}
          </button>
        )}

        {/* Dismiss button */}
        {onDismiss && (
          <button
            className="error-alert-button"
            style={dismissButtonStyles}
            onClick={handleDismiss}
            aria-label="Descartar alerta de error"
            title="Descartar"
            data-testid="error-alert-dismiss-button"
          >
            ✕
          </button>
        )}
      </div>
    </div>
  );
};

export default ErrorAlert;
