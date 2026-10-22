import React, { useMemo } from 'react';
import { Vehicle } from '../../domain/entities/vehicle';
import './VehicleTooltip.css';

/**
 * Props for the VehicleTooltip component.
 *
 * @interface VehicleTooltipProps
 * @property {Vehicle} vehicle - The vehicle entity to display information for
 */
interface VehicleTooltipProps {
  vehicle: Vehicle;
}

/**
 * Calculates a human-readable relative time string (e.g., "hace 5 minutos").
 *
 * @param {Date} timestamp - The timestamp to calculate relative time for
 * @returns {string} A localized Spanish relative time string
 *
 * @example
 * getRelativeTime(new Date(Date.now() - 5 * 60 * 1000)); // "hace 5 minutos"
 */
function getRelativeTime(timestamp: Date): string {
  const now = new Date();
  const diffMs = now.getTime() - timestamp.getTime();
  const diffSeconds = Math.floor(diffMs / 1000);

  if (diffSeconds < 60) {
    return 'hace menos de un minuto';
  }

  const diffMinutes = Math.floor(diffSeconds / 60);
  if (diffMinutes < 60) {
    return `hace ${diffMinutes} ${diffMinutes === 1 ? 'minuto' : 'minutos'}`;
  }

  const diffHours = Math.floor(diffMinutes / 60);
  if (diffHours < 24) {
    return `hace ${diffHours} ${diffHours === 1 ? 'hora' : 'horas'}`;
  }

  const diffDays = Math.floor(diffHours / 24);
  return `hace ${diffDays} ${diffDays === 1 ? 'día' : 'días'}`;
}

/**
 * VehicleTooltip - React component that displays detailed information about a vehicle.
 *
 * Responsibilities:
 * - Display vehicle identification (ID and license plate)
 * - Display geographic coordinates with 4 decimal precision
 * - Display relative timestamp (e.g., "Última actualización: hace 5 minutos")
 * - Format information in a human-readable way
 * - Handle stale position indicators
 *
 * This component displays detailed information about a vehicle in an InfoWindow.
 * The information is shown within the Google Maps marker info window.
 *
 * @component
 * @param {VehicleTooltipProps} props - Component properties
 * @returns {React.ReactElement} The tooltip content
 *
 * @example
 * const vehicle = new Vehicle(
 *   new VehicleId('VH-001'),
 *   'MAD-0001',
 *   new Position(40.4168, -3.7038)
 * );
 *
 * <VehicleTooltip vehicle={vehicle} />
 */
export const VehicleTooltip: React.FC<VehicleTooltipProps> = ({ vehicle }) => {
  /**
   * Memoized relative time string to avoid recalculation on every render.
   * In a real app, this could be updated at intervals if needed.
   */
  const relativeTime = useMemo(
    () => getRelativeTime(vehicle.position.timestamp),
    [vehicle.position.timestamp],
  );

  /**
   * Determine if the position is stale for visual styling purposes.
   */
  const isStale = vehicle.isStalePosition(60 * 60 * 1000);

  return (
    <div className="vehicle-tooltip" data-testid="vehicle-tooltip">
      {/* Vehicle identification section */}
      <div className="tooltip-section">
        <div className="tooltip-row">
          <span className="tooltip-label">Vehículo:</span>
          <span className="tooltip-value" data-testid="tooltip-vehicle-id">
            {vehicle.vehicleId.value}
          </span>
        </div>
        <div className="tooltip-row">
          <span className="tooltip-label">Placa:</span>
          <span className="tooltip-value" data-testid="tooltip-license-plate">
            {vehicle.licensePlate}
          </span>
        </div>
      </div>

      {/* Position section */}
      <div className="tooltip-section">
        <div className="tooltip-row">
          <span className="tooltip-label">Posición:</span>
          <span className="tooltip-value" data-testid="tooltip-coordinates">
            {vehicle.position.toString()}
          </span>
        </div>
        <div className="tooltip-row">
          <span className="tooltip-label">Latitud:</span>
          <span className="tooltip-value" data-testid="tooltip-latitude">
            {vehicle.position.latitude.toFixed(4)}°
          </span>
        </div>
        <div className="tooltip-row">
          <span className="tooltip-label">Longitud:</span>
          <span className="tooltip-value" data-testid="tooltip-longitude">
            {vehicle.position.longitude.toFixed(4)}°
          </span>
        </div>
      </div>

      {/* Timestamp section */}
      <div className="tooltip-section">
        <div className={`tooltip-row ${isStale ? 'stale' : ''}`}>
          <span className="tooltip-label">Última actualización:</span>
          <span className="tooltip-value" data-testid="tooltip-timestamp">
            {relativeTime}
          </span>
        </div>
        {isStale && (
          <div className="tooltip-warning">
            ⚠️ Posición desactualizada ({">"}1 hora)
          </div>
        )}
      </div>
    </div>
  );
};
