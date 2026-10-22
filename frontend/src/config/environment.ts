/**
 * Environment configuration for the frontend application.
 * Supports both mock and API modes for development and production.
 *
 * Environment variables:
 * - VITE_DATA_MODE: 'mock' or 'api' (default: 'mock')
 * - VITE_API_BASE_URL: Backend API base URL (default: http://localhost:8080)
 * - VITE_SSE_ENDPOINT: SSE endpoint URL (default: http://localhost:8080/api/vehicles/updates)
 */

export const environment = {
  dataMode: (import.meta.env.VITE_DATA_MODE || 'mock') as 'mock' | 'api',
  apiBaseUrl: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080',
  sseEndpoint: import.meta.env.VITE_SSE_ENDPOINT || 'http://localhost:8080/api/vehicles/updates',
};

/**
 * Check if running in mock mode (development/demo)
 */
export const isMockMode = (): boolean => environment.dataMode === 'mock';

/**
 * Check if running in API mode (production)
 */
export const isApiMode = (): boolean => environment.dataMode === 'api';
