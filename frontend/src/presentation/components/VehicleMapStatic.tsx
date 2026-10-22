import React, { useMemo } from 'react';
import { Vehicle } from '../../domain/entities/vehicle';
import './VehicleMap.css';

interface VehicleMapStaticProps {
  vehicles: Vehicle[];
  selectedVehicleId?: string;
  onVehicleSelect?: (vehicleId: string) => void;
}

/**
 * VehicleMapStatic - Fallback static map component for when Google Maps API is not available.
 * 
 * This is a simplified version that shows vehicle positions as a grid.
 * In production, this would be replaced with a proper map library.
 */
export const VehicleMapStatic: React.FC<VehicleMapStaticProps> = ({
  vehicles,
  selectedVehicleId,
  onVehicleSelect,
}) => {
  // Madrid bounds
  const MADRID_LAT_MIN = 40.35;
  const MADRID_LAT_MAX = 40.50;
  const MADRID_LNG_MIN = -3.80;
  const MADRID_LNG_MAX = -3.60;

  const mapWidth = 800;
  const mapHeight = 600;

  // Normalize coordinates to pixel positions
  const coordToPixel = (lat: number, lng: number) => {
    const x = ((lng - MADRID_LNG_MIN) / (MADRID_LNG_MAX - MADRID_LNG_MIN)) * mapWidth;
    const y = ((MADRID_LAT_MAX - lat) / (MADRID_LAT_MAX - MADRID_LAT_MIN)) * mapHeight;
    return { x, y };
  };

  const selectedVehicle = useMemo(
    () => vehicles.find((v) => v.vehicleId.value === selectedVehicleId),
    [vehicles, selectedVehicleId],
  );

  return (
    <div className="vehicle-map-static" style={{ width: '100%', height: '100%', position: 'relative' }}>
      <svg
        width="100%"
        height="100%"
        viewBox={`0 0 ${mapWidth} ${mapHeight}`}
        style={{ backgroundColor: '#e8f4f8', border: '1px solid #ccc' }}
      >
        {/* Grid background */}
        <defs>
          <pattern id="grid" width="50" height="50" patternUnits="userSpaceOnUse">
            <path d="M 50 0 L 0 0 0 50" fill="none" stroke="#f0f0f0" strokeWidth="1" />
          </pattern>
        </defs>
        <rect width={mapWidth} height={mapHeight} fill="url(#grid)" />

        {/* Vehicles as circles */}
        {vehicles.map((vehicle) => {
          const { x, y } = coordToPixel(vehicle.position.latitude, vehicle.position.longitude);
          const isSelected = vehicle.vehicleId.value === selectedVehicleId;
          const color = isSelected ? '#0066cc' : '#cc0000';
          const radius = isSelected ? 8 : 6;

          return (
            <g
              key={vehicle.vehicleId.value}
              onClick={() => onVehicleSelect?.(vehicle.vehicleId.value)}
              style={{ cursor: 'pointer' }}
            >
              {/* Outer circle */}
              <circle cx={x} cy={y} r={radius} fill={color} opacity="0.7" />
              {/* Inner circle */}
              <circle cx={x} cy={y} r={radius - 2} fill={color} opacity="0.9" />
            </g>
          );
        })}

        {/* Selected vehicle info */}
        {selectedVehicle && (
          <g>
            {(() => {
              const { x, y } = coordToPixel(
                selectedVehicle.position.latitude,
                selectedVehicle.position.longitude,
              );
              return (
                <rect
                  x={x + 15}
                  y={y - 30}
                  width="180"
                  height="60"
                  fill="white"
                  stroke="#0066cc"
                  strokeWidth="2"
                  rx="4"
                />
              );
            })()}
          </g>
        )}
      </svg>

      {/* Info panel */}
      {selectedVehicle && (
        <div style={{
          position: 'absolute',
          bottom: '16px',
          left: '16px',
          backgroundColor: 'white',
          padding: '12px',
          borderRadius: '4px',
          boxShadow: '0 2px 8px rgba(0,0,0,0.2)',
          maxWidth: '300px',
          fontSize: '14px',
        }}>
          <strong>{selectedVehicle.licensePlate}</strong>
          <div>ID: {selectedVehicle.vehicleId.value}</div>
          <div>Posición: {selectedVehicle.position.latitude.toFixed(4)}, {selectedVehicle.position.longitude.toFixed(4)}</div>
        </div>
      )}

      {/* Legend */}
      <div style={{
        position: 'absolute',
        top: '16px',
        left: '16px',
        backgroundColor: 'white',
        padding: '12px',
        borderRadius: '4px',
        boxShadow: '0 2px 8px rgba(0,0,0,0.2)',
        fontSize: '12px',
      }}>
        <div><strong>Vehículos ({vehicles.length})</strong></div>
        <div>🔴 Rojo: Disponible</div>
        <div>🔵 Azul: Seleccionado</div>
        <div style={{ marginTop: '8px', fontSize: '11px', color: '#666' }}>
          Nota: Mapa de vista previa. Obtén Google Maps API key para mapa interactivo.
        </div>
      </div>
    </div>
  );
};
