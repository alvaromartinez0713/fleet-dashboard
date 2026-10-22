/**
 * Environment configuration loaded from window.ENV at runtime
 * 
 * This module reads configuration injected by docker-entrypoint.sh
 * or from import.meta.env in development
 */

declare global {
  interface Window {
    ENV?: {
      VITE_DATA_MODE?: string;
      VITE_API_BASE_URL?: string;
      VITE_SSE_ENDPOINT?: string;
      VITE_MAPBOX_TOKEN?: string;
    };
  }
}

// Read from window.ENV (injected at runtime) with fallbacks
export const config = {
  dataMode: (typeof window !== 'undefined' && window.ENV?.VITE_DATA_MODE) || 'mock',
  apiBaseUrl: (typeof window !== 'undefined' && window.ENV?.VITE_API_BASE_URL) || 'http://localhost:8080',
  sseEndpoint: (typeof window !== 'undefined' && window.ENV?.VITE_SSE_ENDPOINT) || 'http://localhost:8080/api/vehicles/updates',
  mapboxToken: (typeof window !== 'undefined' && window.ENV?.VITE_MAPBOX_TOKEN) || '',
};

export default config;

