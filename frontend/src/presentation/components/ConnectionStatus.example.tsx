/**
 * ConnectionStatus Component Usage Examples
 *
 * This file demonstrates various ways to use the ConnectionStatus component
 * in different parts of the dashboard application.
 */

import React from 'react';
import { ConnectionStatus } from './ConnectionStatus';

/**
 * Example 1: Minimal usage - just the indicator
 *
 * Shows only the status indicator dot, useful for compact header placement.
 */
export const ConnectionStatusMinimal: React.FC = () => {
  return <ConnectionStatus showLabel={false} showCountdown={false} />;
};

/**
 * Example 2: Standard dashboard header placement
 *
 * Shows indicator, label, and countdown - typical dashboard header setup.
 */
export const ConnectionStatusHeader: React.FC = () => {
  return <ConnectionStatus showLabel={true} showCountdown={true} />;
};

/**
 * Example 3: With manual retry handler
 *
 * Demonstrates handling retry attempts with custom logic.
 */
export const ConnectionStatusWithRetry: React.FC = () => {
  const handleRetry = () => {
    console.log('User clicked retry - custom retry logic can be triggered here');
    // Trigger custom retry logic, e.g., restart SSE connection
    // refreshSSEConnection();
  };

  return <ConnectionStatus showLabel={true} showCountdown={true} onRetry={handleRetry} />;
};

/**
 * Example 4: Integration in a dashboard header/toolbar
 *
 * Shows how ConnectionStatus would typically be integrated into a header component.
 */
export const DashboardHeaderWithConnectionStatus: React.FC = () => {
  return (
    <header
      style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        padding: '16px',
        borderBottom: '1px solid #e5e7eb',
        backgroundColor: '#ffffff',
      }}
    >
      <div>
        <h1 style={{ margin: 0, fontSize: '24px', fontWeight: 'bold' }}>
          Panel de Control de Flotas
        </h1>
      </div>

      <div
        style={{
          display: 'flex',
          gap: '16px',
          alignItems: 'center',
        }}
      >
        {/* Connection status indicator */}
        <ConnectionStatus showLabel={true} showCountdown={true} />

        {/* Other header controls */}
        <button
          style={{
            padding: '8px 16px',
            backgroundColor: '#3b82f6',
            color: 'white',
            border: 'none',
            borderRadius: '6px',
            cursor: 'pointer',
          }}
        >
          Configuración
        </button>
      </div>
    </header>
  );
};

/**
 * Example 5: Status bar at bottom of page
 *
 * Shows ConnectionStatus as a status bar at the bottom of the application.
 */
export const ConnectionStatusBar: React.FC = () => {
  return (
    <footer
      style={{
        position: 'fixed',
        bottom: 0,
        left: 0,
        right: 0,
        padding: '12px 16px',
        backgroundColor: '#f3f4f6',
        borderTop: '1px solid #e5e7eb',
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
      }}
    >
      <div style={{ fontSize: '12px', color: '#6b7280' }}>Fleet Dashboard v1.0.0</div>

      <ConnectionStatus showLabel={true} showCountdown={true} />

      <div style={{ fontSize: '12px', color: '#6b7280' }}>© 2024 Fleet Corporation</div>
    </footer>
  );
};

/**
 * Example 6: Compact version for sidebar
 *
 * Minimal version suitable for a compact sidebar placement.
 */
export const SidebarConnectionStatus: React.FC = () => {
  return (
    <div
      style={{
        padding: '12px',
        backgroundColor: '#f9fafb',
        borderRadius: '6px',
        margin: '12px 0',
      }}
    >
      <div style={{ fontSize: '12px', color: '#6b7280', marginBottom: '6px', fontWeight: '500' }}>
        Estado de Actualización
      </div>
      <ConnectionStatus showLabel={true} showCountdown={false} />
    </div>
  );
};

export default {
  ConnectionStatusMinimal,
  ConnectionStatusHeader,
  ConnectionStatusWithRetry,
  DashboardHeaderWithConnectionStatus,
  ConnectionStatusBar,
  SidebarConnectionStatus,
};
