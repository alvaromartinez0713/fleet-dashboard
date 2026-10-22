import React, { useState } from 'react';
import { getDataProviderMode } from '@infrastructure/data-providers/data-provider-factory';

/**
 * DemoModeBanner - Component that displays a warning banner in mock/demo mode
 *
 * Shows prominently at the top of the dashboard when the application is running
 * with mock data instead of the production API. The banner includes:
 * - Warning-style styling (yellow/orange background)
 * - Clear text explaining that demo mode with mock data is active
 * - Optional close/dismiss button to hide the banner
 *
 * The component checks the data provider mode and only renders when running
 * in 'mock' mode (not 'api' mode).
 *
 * Verifies: RF-12, Product requirement for clear mode indication
 *
 * @example
 * // Place at the top of your dashboard or app layout
 * <DemoModeBanner />
 *
 * @returns React component that displays a banner in demo mode, or null if in API mode
 */
export const DemoModeBanner: React.FC = () => {
  const [isDismissed, setIsDismissed] = useState(false);
  const mode = getDataProviderMode();

  // Only show the banner when running in mock mode
  if (mode !== 'mock' || isDismissed) {
    return null;
  }

  const bannerStyles: React.CSSProperties = {
    backgroundColor: '#FFA500', // Orange/warning color
    color: '#333333', // Dark text for contrast
    padding: '16px 20px',
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    borderBottom: '3px solid #FF8C00',
    fontWeight: 500,
    fontSize: '14px',
    boxShadow: '0 2px 4px rgba(0, 0, 0, 0.1)',
  };

  const messageContainerStyles: React.CSSProperties = {
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
    flex: 1,
  };

  const warningIconStyles: React.CSSProperties = {
    fontSize: '20px',
    fontWeight: 'bold',
    minWidth: '20px',
    textAlign: 'center',
  };

  const messageStyles: React.CSSProperties = {
    margin: 0,
  };

  const dismissButtonStyles: React.CSSProperties = {
    backgroundColor: 'transparent',
    border: 'none',
    color: '#333333',
    cursor: 'pointer',
    fontSize: '20px',
    padding: '0 8px',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    minWidth: '32px',
    minHeight: '32px',
    borderRadius: '4px',
    transition: 'background-color 0.2s ease',
    fontWeight: 'bold',
  };

  const handleDismiss = () => {
    setIsDismissed(true);
  };

  const handleMouseEnter = (e: React.MouseEvent<HTMLButtonElement>) => {
    (e.currentTarget as HTMLButtonElement).style.backgroundColor = 'rgba(0, 0, 0, 0.1)';
  };

  const handleMouseLeave = (e: React.MouseEvent<HTMLButtonElement>) => {
    (e.currentTarget as HTMLButtonElement).style.backgroundColor = 'transparent';
  };

  return (
    <div style={bannerStyles} data-testid="demo-mode-banner">
      <div style={messageContainerStyles}>
        <span style={warningIconStyles} aria-hidden="true">
          ⚠️
        </span>
        <p style={messageStyles}>
          Modo Demo con Datos Simulados - Esta aplicación está utilizando datos ficticios
          para demostración. No es un entorno de producción.
        </p>
      </div>
      <button
        style={dismissButtonStyles}
        onClick={handleDismiss}
        onMouseEnter={handleMouseEnter}
        onMouseLeave={handleMouseLeave}
        aria-label="Cerrar banner de modo demo"
        title="Cerrar banner"
        data-testid="demo-banner-close-button"
      >
        ✕
      </button>
    </div>
  );
};

export default DemoModeBanner;
