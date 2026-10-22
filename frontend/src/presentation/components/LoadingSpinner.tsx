import React from 'react';

/**
 * LoadingSpinnerSize - Size variants for LoadingSpinner component
 */
type LoadingSpinnerSize = 'small' | 'medium' | 'large';

/**
 * LoadingSpinnerProps - Props for the LoadingSpinner component
 *
 * @property size - Size variant: 'small' (24px), 'medium' (48px), 'large' (64px). Default: 'medium'
 * @property statusText - Optional status text to display below spinner. Examples: "Conectando...", "Cargando vehículos..."
 * @property ariaLabel - Custom ARIA label for accessibility. Default generated from size and statusText
 */
interface LoadingSpinnerProps {
  size?: LoadingSpinnerSize;
  statusText?: string;
  ariaLabel?: string;
}

/**
 * LoadingSpinner - React component that displays an animated loading spinner
 *
 * Features:
 * - Animated rotating spinner with smooth CSS animation
 * - Three size variants (small, medium, large)
 * - Optional status text to indicate what is being loaded
 * - Centered positioning
 * - Accessible with ARIA labels and role="status"
 * - Consistent styling with other dashboard components
 * - Deterministic animation (no animation frame dependency)
 *
 * Size Variants:
 * - small: 24px (for inline loading states)
 * - medium: 48px (default, for main content areas)
 * - large: 64px (for full-page loading states)
 *
 * Status Text Examples:
 * - "Conectando..." (connecting)
 * - "Cargando vehículos..." (loading vehicles)
 * - "Sincronizando datos..." (synchronizing data)
 *
 * Usage:
 * ```typescript
 * // Basic spinner with default size
 * <LoadingSpinner />
 *
 * // Spinner with status text
 * <LoadingSpinner statusText="Cargando vehículos..." />
 *
 * // Large spinner for full-page loading
 * <LoadingSpinner size="large" statusText="Inicializando..." />
 *
 * // Small spinner for inline use
 * <LoadingSpinner size="small" />
 * ```
 *
 * Verifies: RF-07 (dashboard reactivo), RNF-08 (actualización reactiva)
 */
export const LoadingSpinner: React.FC<LoadingSpinnerProps> = ({
  size = 'medium',
  statusText,
  ariaLabel: customAriaLabel,
}) => {
  /**
   * Get spinner dimensions based on size variant
   */
  const getSizePixels = (sizeVariant: LoadingSpinnerSize): number => {
    switch (sizeVariant) {
      case 'small':
        return 24;
      case 'medium':
        return 48;
      case 'large':
        return 64;
      default:
        return 48;
    }
  };

  /**
   * Generate default ARIA label if not provided
   */
  const ariaLabel =
    customAriaLabel ||
    (statusText
      ? `Cargando: ${statusText}`
      : `Cargando contenido, por favor espera`);

  const spinnerSize = getSizePixels(size);
  const borderWidth = Math.max(2, Math.round(spinnerSize / 24 * 2)); // Proportional border width

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        gap: '16px',
        padding: '24px',
      }}
      role="status"
      aria-live="polite"
      aria-label={ariaLabel}
      data-testid="loading-spinner"
    >
      <style>
        {`
          @keyframes spin {
            from {
              transform: rotate(0deg);
            }
            to {
              transform: rotate(360deg);
            }
          }

          .loading-spinner-circle {
            animation: spin 1s linear infinite;
          }
        `}
      </style>

      {/* Spinner circle */}
      <div
        className="loading-spinner-circle"
        style={{
          width: `${spinnerSize}px`,
          height: `${spinnerSize}px`,
          border: `${borderWidth}px solid #e5e7eb`,
          borderTop: `${borderWidth}px solid #3b82f6`,
          borderRadius: '50%',
        }}
        data-testid="loading-spinner-circle"
        aria-hidden="true"
      />

      {/* Optional status text */}
      {statusText && (
        <div
          style={{
            color: '#6b7280',
            fontSize: '14px',
            fontWeight: '500',
            lineHeight: '1.5',
            textAlign: 'center',
            maxWidth: '200px',
          }}
          data-testid="loading-spinner-status-text"
        >
          {statusText}
        </div>
      )}
    </div>
  );
};

export default LoadingSpinner;
