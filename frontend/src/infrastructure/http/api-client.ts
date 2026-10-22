import { environment } from '@config/environment';

/**
 * HTTP client for communicating with the backend API.
 *
 * Responsibilities:
 * - Make HTTP requests to the backend
 * - Handle network errors
 * - Provide type-safe request/response handling
 *
 * Usage:
 * ```typescript
 * const client = new ApiClient();
 * const vehicles = await client.get<VehicleDTO[]>('/api/vehicles');
 * ```
 */
export class ApiClient {
  private baseUrl: string;

  constructor(baseUrl?: string) {
    this.baseUrl = baseUrl || environment.apiBaseUrl;
  }

  /**
   * Make a GET request to the API.
   *
   * @param path - The API endpoint path (e.g., '/api/vehicles')
   * @returns Promise that resolves to the parsed JSON response
   * @throws Error if the request fails or response is not OK
   */
  async get<T>(path: string): Promise<T> {
    const url = `${this.baseUrl}${path}`;

    try {
      const response = await fetch(url, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        const errorMessage = `HTTP ${response.status}: ${response.statusText}`;
        throw new Error(errorMessage);
      }

      return await response.json() as T;
    } catch (error) {
      // Re-throw with context about which endpoint failed
      throw new Error(`Failed to fetch from ${url}: ${error instanceof Error ? error.message : String(error)}`);
    }
  }
}

/**
 * Error thrown when vehicle data cannot be loaded from API.
 */
export class VehicleLoadError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'VehicleLoadError';
  }
}
