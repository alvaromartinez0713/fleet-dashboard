/**
 * MSW Request Handlers
 */

import { http, HttpResponse } from 'msw';
import { ProblemDetailsTemplates } from './problem-details';

// Mock data: vehículos en Madrid con coordenadas variadas
const MADRID_CENTER = {
  lat: 40.4168,
  lng: -3.7038,
};

// Generar 5 vehículos con coordenadas alrededor de Madrid
const generateMockVehicles = () => {
  return Array.from({ length: 5 }, (_, i) => ({
    vehicleId: `VH-${String(i + 1).padStart(3, '0')}`,
    licensePlate: `MAD-${String(i + 1).padStart(4, '0')}`,
    status: 'active',
    lastUpdate: new Date().toISOString(),
    // Distribuir vehículos en área de ~5km alrededor de Madrid
    latitude: MADRID_CENTER.lat + (Math.random() - 0.5) * 0.05, // ±0.025 ≈ ±2.7km
    longitude: MADRID_CENTER.lng + (Math.random() - 0.5) * 0.05,
  }));
};

const getVehicles = http.get('/api/vehicles', ({ request }) => {
  const url = new URL(request.url);
  const page = parseInt(url.searchParams.get('page') || '1', 10);
  const limit = parseInt(url.searchParams.get('limit') || '20', 10);

  const mockVehicles = generateMockVehicles().slice(0, Math.min(limit, 5));

  return HttpResponse.json({
    data: mockVehicles,
    pagination: {
      page,
      limit,
      total: 150,
      pages: Math.ceil(150 / limit),
    },
  });
});

const getVehicleById = http.get('/api/vehicles/:vehicleId', ({ params }) => {
  const { vehicleId } = params;

  if (vehicleId === 'NOT_FOUND') {
    return ProblemDetailsTemplates.notFound('Vehicle', vehicleId);
  }

  return HttpResponse.json({
    vehicleId,
    licensePlate: `MAD-${typeof vehicleId === 'string' ? vehicleId.split('-')[1] : '0001'}`,
    status: 'active',
    lastUpdate: new Date().toISOString(),
    latitude: MADRID_CENTER.lat + (Math.random() - 0.5) * 0.05,
    longitude: MADRID_CENTER.lng + (Math.random() - 0.5) * 0.05,
    accuracy: 5.2,
  });
});

const getVehicleUpdates = http.get('/api/vehicles/updates', () => {
  const encoder = new TextEncoder();

  const stream = new ReadableStream({
    start(controller) {
      // Enviar eventos iniciales para cada vehículo
      const initialVehicles = generateMockVehicles();
      initialVehicles.forEach((vehicle) => {
        const initialEvent = {
          vehicleId: vehicle.vehicleId,
          latitude: vehicle.latitude,
          longitude: vehicle.longitude,
          timestamp: new Date().toISOString(),
        };

        controller.enqueue(
          encoder.encode(`data: ${JSON.stringify(initialEvent)}\n\n`)
        );
      });

      // Enviar actualizaciones cada 2 segundos
      const interval = setInterval(() => {
        const vehicleIdx = Math.floor(Math.random() * 5);
        const updateEvent = {
          vehicleId: `VH-${String(vehicleIdx + 1).padStart(3, '0')}`,
          latitude: MADRID_CENTER.lat + (Math.random() - 0.5) * 0.05,
          longitude: MADRID_CENTER.lng + (Math.random() - 0.5) * 0.05,
          timestamp: new Date().toISOString(),
        };

        controller.enqueue(
          encoder.encode(`data: ${JSON.stringify(updateEvent)}\n\n`)
        );
      }, 2000);

      return () => clearInterval(interval);
    },
  });

  return new HttpResponse(stream, {
    headers: {
      'Content-Type': 'text/event-stream',
      'Cache-Control': 'no-cache',
      Connection: 'keep-alive',
    },
  });
});

const getVehicleAlerts = http.get('/api/vehicles/:vehicleId/alerts', ({ params }) => {
  const { vehicleId } = params;

  const mockAlerts = [
    {
      id: 'ALERT-001',
      vehicleId,
      type: 'SPEED_LIMIT_EXCEEDED',
      severity: 'warning',
      message: 'Vehicle exceeded speed limit',
      timestamp: new Date().toISOString(),
    },
    {
      id: 'ALERT-002',
      vehicleId,
      type: 'GEOFENCE_EXIT',
      severity: 'info',
      message: 'Vehicle exited designated zone',
      timestamp: new Date().toISOString(),
    },
  ];

  return HttpResponse.json({
    vehicleId,
    alerts: mockAlerts,
    count: mockAlerts.length,
  });
});

const rateLimitError = http.get('/api/vehicles/error/429', () => {
  return ProblemDetailsTemplates.rateLimitExceeded(12, '60 seconds');
});

const internalServerError = http.get('/api/vehicles/error/500', () => {
  return ProblemDetailsTemplates.internalError();
});

const serviceUnavailable = http.get('/api/vehicles/error/503', () => {
  return ProblemDetailsTemplates.serviceUnavailable('Backend API');
});

export const handlers = [
  getVehicles,
  getVehicleById,
  getVehicleUpdates,
  getVehicleAlerts,
  rateLimitError,
  internalServerError,
  serviceUnavailable,
];
