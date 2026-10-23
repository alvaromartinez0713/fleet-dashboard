import { IApiClient, ApiSendResponse, PositionPayload } from '../../application/ports/IApiClient';

/**
 * RealApiClient
 *
 * Production implementation of IApiClient using fetch API.
 * Sends position data to the backend API.
 */
export class RealApiClient implements IApiClient {
  private baseUrl: string;
  private timeout: number = 10000; // 10 seconds

  constructor(baseUrl: string = 'http://localhost:3000/api') {
    this.baseUrl = baseUrl;
  }

  /**
   * Send positions to backend API
   */
  async sendPositions(payloads: PositionPayload[]): Promise<ApiSendResponse> {
    try {
      const response = await this.fetchWithTimeout(
        `${this.baseUrl}/v1/positions`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            positions: payloads.map(p => this.payloadToJson(p)),
          }),
        },
        this.timeout
      );

      // Extract status and parse error if needed
      const status = response.status;
      let error = undefined;

      if (status !== 202 && status !== 200 && status !== 201) {
        error = await this.parseErrorResponse(response);
      }

      return { status, error };
    } catch (err) {
      // Network error or timeout
      const message = err instanceof Error ? err.message : 'Unknown network error';
      return {
        status: 0, // 0 indicates network error, not HTTP status
        error: {
          message,
          type: 'network_error',
        },
      };
    }
  }

  /**
   * Fetch with timeout
   */
  private fetchWithTimeout(
    url: string,
    options: RequestInit,
    timeoutMs: number
  ): Promise<Response> {
    return Promise.race([
      fetch(url, options),
      new Promise<Response>((_, reject) =>
        setTimeout(
          () => reject(new Error(`Request timeout after ${timeoutMs}ms`)),
          timeoutMs
        )
      ),
    ]);
  }

  /**
   * Parse error response from backend
   */
  private async parseErrorResponse(response: Response): Promise<{
    status: number;
    message: string;
    type?: string;
  }> {
    try {
      const contentType = response.headers.get('Content-Type') || '';
      
      if (contentType.includes('application/json')) {
        const json = await response.json();
        // RFC 9457 Problem Details
        return {
          status: json.status || response.status,
          message: json.detail || json.message || `HTTP ${response.status}`,
          type: json.type || 'error',
        };
      }
    } catch (err) {
      // Parsing error, use default
    }

    return {
      status: response.status,
      message: `HTTP ${response.status}: ${response.statusText}`,
      type: 'error',
    };
  }

  /**
   * Convert PositionPayload to JSON (serialize dates)
   */
  private payloadToJson(payload: PositionPayload): any {
    return {
      eventId: payload.eventId,
      vehicleId: payload.vehicleId,
      latitude: payload.latitude,
      longitude: payload.longitude,
      accuracy: payload.accuracy,
      capturedAt: payload.capturedAt.toISOString(),
      traceId: payload.traceId,
      idempotencyKey: payload.idempotencyKey,
    };
  }
}
