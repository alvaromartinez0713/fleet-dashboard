import React from 'react';
import { useVehicles } from '@presentation/hooks/useVehicles';
import { useSSEConnection } from '@presentation/hooks/useSSEConnection';
import {
  ConnectionStatus,
  DemoModeBanner,
  ErrorAlert,
  LoadingSpinner,
  VehicleMap,
} from '@presentation/components';
import '../styles/vehicle-dashboard.css';

export const VehicleDashboard: React.FC = () => {
  const { vehicles, loading, error, refetch } = useVehicles();
  const { status: connectionStatus, error: sseError } = useSSEConnection();

  const displayError = error || sseError;
  const isDemoMode = import.meta.env.VITE_DATA_MODE === 'mock';

  return (
    <div className="vehicle-dashboard" data-testid="vehicle-dashboard">
      <header className="dashboard-header" data-testid="dashboard-header">
        <ConnectionStatus status={connectionStatus} />
        {isDemoMode && <DemoModeBanner />}
      </header>

      <main className="dashboard-main" data-testid="dashboard-main">
        {loading && (
          <div className="dashboard-loading" data-testid="dashboard-loading">
            <LoadingSpinner message="Cargando vehículos..." />
          </div>
        )}

        {displayError && (
          <div className="dashboard-error" data-testid="dashboard-error">
            <ErrorAlert
              error={displayError}
              onDismiss={() => refetch()}
              onRetry={refetch}
              showRetry={true}
            />
          </div>
        )}

        {!loading && (
          <div className="dashboard-map-container" data-testid="dashboard-map-container">
            {vehicles.length === 0 && !error && (
              <div className="dashboard-empty-state" data-testid="dashboard-empty-state">
                <p>No hay vehículos disponibles</p>
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
