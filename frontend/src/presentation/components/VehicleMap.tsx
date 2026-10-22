import React, { useEffect, useRef, useMemo } from 'react';
import mapboxgl, { Map as MapboxMap } from 'mapbox-gl';
import { Vehicle } from '../../domain/entities/vehicle';
import 'mapbox-gl/dist/mapbox-gl.css';
import './VehicleMap.css';

interface VehicleMapProps {
  vehicles: Vehicle[];
  selectedVehicleId?: string;
  onVehicleSelect?: (vehicleId: string) => void;
}

/**
 * VehicleMap - React component for displaying vehicles on Mapbox GL JS.
 */
export const VehicleMap: React.FC<VehicleMapProps> = ({
  vehicles,
  selectedVehicleId,
  onVehicleSelect,
}) => {
  const mapContainer = useRef<HTMLDivElement>(null);
  const map = useRef<MapboxMap | null>(null);
  const mapboxToken = import.meta.env.VITE_MAPBOX_TOKEN;

  // Madrid center
  const center = useMemo(() => ({ lat: 40.4168, lng: -3.7038 }), []);

  if (!mapboxToken) {
    return (
      <div className="vehicle-map-error">
        <div>
          <p>❌ Mapbox token no configurado</p>
          <p style={{ fontSize: '13px', marginTop: '12px' }}>Para usar Mapbox:</p>
          <ol style={{ fontSize: '12px', textAlign: 'left', marginTop: '8px', paddingLeft: '20px' }}>
            <li>Ve a: <a href="https://account.mapbox.com/auth/signup/" target="_blank" rel="noopener noreferrer">Mapbox Signup</a></li>
            <li>Crea una cuenta gratuita</li>
            <li>Ve a <a href="https://account.mapbox.com/tokens/" target="_blank" rel="noopener noreferrer">Tokens</a></li>
            <li>Crea un token público</li>
            <li>Copia el token y agrega a <code>.env.local</code>:</li>
            <li><code style={{ display: 'block', marginTop: '4px', backgroundColor: '#f5f5f5', padding: '4px' }}>VITE_MAPBOX_TOKEN=pk.eyJ...</code></li>
            <li>Recarga la página</li>
          </ol>
        </div>
      </div>
    );
  }

  // Initialize map
  useEffect(() => {
    if (!mapContainer.current || map.current) return;

    mapboxgl.accessToken = mapboxToken;

    map.current = new mapboxgl.Map({
      container: mapContainer.current,
      style: 'mapbox://styles/mapbox/streets-v12',
      center: [center.lng, center.lat],
      zoom: 12,
    });

    // Add controls
    map.current.addControl(new mapboxgl.NavigationControl(), 'top-left');
    map.current.addControl(new mapboxgl.FullscreenControl(), 'top-left');
    map.current.addControl(new mapboxgl.GeolocateControl(), 'top-left');

    console.log('✅ Mapbox initialized');

    return () => {
      if (map.current) {
        map.current.remove();
        map.current = null;
      }
    };
  }, [mapboxToken, center]);

  // Update markers
  useEffect(() => {
    if (!map.current) return;

    console.log(`🎯 Updating ${vehicles.length} markers`);

    // Remove existing markers
    document.querySelectorAll('.vehicle-marker').forEach(el => el.remove());

    // Add new markers
    vehicles.forEach((vehicle) => {
      const isSelected = vehicle.vehicleId.value === selectedVehicleId;

      const el = document.createElement('div');
      el.className = `vehicle-marker ${isSelected ? 'selected' : ''}`;
      el.innerHTML = `
        <div class="marker-icon">${isSelected ? '🔵' : '🔴'}</div>
        <div class="marker-label">${vehicle.licensePlate}</div>
        ${isSelected ? `
          <div class="marker-info">
            <div><strong>${vehicle.licensePlate}</strong></div>
            <div class="info-row"><strong>ID:</strong> ${vehicle.vehicleId.value}</div>
            <div class="info-row"><strong>Pos:</strong> ${vehicle.position.latitude.toFixed(4)}, ${vehicle.position.longitude.toFixed(4)}</div>
            <div class="info-row"><strong>Hora:</strong> ${vehicle.position.timestamp.toLocaleTimeString('es-ES')}</div>
          </div>
        ` : ''}
      `;

      el.addEventListener('click', () => {
        onVehicleSelect?.(vehicle.vehicleId.value);
      });

      new mapboxgl.Marker(el)
        .setLngLat([vehicle.position.longitude, vehicle.position.latitude])
        .addTo(map.current!);
    });
  }, [vehicles, selectedVehicleId, onVehicleSelect]);

  const selectedCount = vehicles.filter(v => v.vehicleId.value === selectedVehicleId).length;
  const availableCount = vehicles.length - selectedCount;

  return (
    <div className="vehicle-map-wrapper">
      <div ref={mapContainer} className="vehicle-map-container" />
      
      {/* Legend */}
      <div className="map-legend">
        <div className="legend-title">
          <strong>Vehículos ({vehicles.length})</strong>
        </div>
        <div className="legend-item">
          <span className="legend-icon">🔴</span>
          <span className="legend-label">Disponibles: {availableCount}</span>
        </div>
        <div className="legend-item">
          <span className="legend-icon">🔵</span>
          <span className="legend-label">Seleccionado: {selectedCount}</span>
        </div>
        <div className="legend-divider"></div>
        <div style={{ fontSize: '11px', color: '#666', marginTop: '8px' }}>
          Haz clic en un vehículo para seleccionar
        </div>
      </div>
    </div>
  );
};
