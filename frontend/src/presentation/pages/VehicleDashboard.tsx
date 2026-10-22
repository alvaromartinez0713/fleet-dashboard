import React from 'react';
import { useVehicles } from '@presentation/hooks/useVehicles';
import { useSSEConnection } from '@presentation/hooks/useSSEConnection';
import { VehicleMap } from '@presentation/components/VehicleMap';
import { ConnectionStatus } from '@application/ports/vehicle-realtime-port';
import config from '@config/environment';
import '../styles/vehicle-dashboard.css';

export const VehicleDashboard: React.FC = () => {
  const { vehicles, loading, error, refetch } = useVehicles();
  const { status: connectionStatus, error: sseError } = useSSEConnection();

  const displayError = error || sseError;
  const isDemoMode = config.dataMode === 'mock';

  // Format connection status for display
  const getStatusDisplay = () => {
    switch (connectionStatus) {
      case ConnectionStatus.CONNECTED:
        return '🟢 Connected';
      case ConnectionStatus.RECONNECTING:
        return '🟡 Connecting...';
      case ConnectionStatus.DISCONNECTED:
        return '🔴 Offline';
      case ConnectionStatus.ERROR:
        return '❌ Error';
      default:
        return '⚪ Unknown';
    }
  };

  return (
    <div className="vehicle-dashboard" data-testid="vehicle-dashboard">
      <header className="dashboard-header" data-testid="dashboard-header">
        <div className="header-title">
          <h1>Fleet Dashboard</h1>
          {isDemoMode && <span className="demo-badge">Modo Demo</span>}
        </div>
        <div className="header-connection-status">
          {getStatusDisplay()}
        </div>
      </header>

      <main className="dashboard-main" data-testid="dashboard-main">
        {loading && (
          <div className="dashboard-loading" data-testid="dashboard-loading">
            <p>⏳ Cargando vehículos...</p>
          </div>
        )}

        {displayError && (
          <div className="dashboard-error" data-testid="dashboard-error">
            <p>⚠️ Error: {displayError instanceof Error ? displayError.message : String(displayError)}</p>
            <button onClick={() => refetch()}>Reintentar</button>
          </div>
        )}

        {!loading && (
          <div className="dashboard-map-container" data-testid="dashboard-map-container">
            {vehicles.length === 0 && !error && (
              <div className="dashboard-empty-state" data-testid="dashboard-empty-state">
                <p>📍 No hay vehículos disponibles</p>
              </div>
            )}
            <VehicleMap vehicles={vehicles} />
          </div>
        )}
      </main>

      <footer className="dashboard-footer" data-testid="dashboard-footer"></footer>
    </div>
  );
};
