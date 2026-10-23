import { PositionPayload } from '../../domain/models/Position';

/**
 * API response status codes and their meanings:
 * - 202 Accepted: Backend received the positions and queued them for processing in Kafka (success)
 * - 400 Bad Request: Invalid request format or missing required fields (validation error, don't retry)
 * - 422 Unprocessable Content: Request format is valid but data fails semantic validation (don't retry)
 * - 500 Internal Server Error: Unexpected server error (transient, should retry)
 * - 503 Service Unavailable: Server temporarily unavailable (transient, should retry)
 * - Other errors: Network timeout, connection refused, etc. (transient, should retry)
 */

/**
 * Error details from API response
 */
export interface ApiErrorResponse {
  /**
   * HTTP status code (e.g., 400, 422, 500)
   */
  status?: number;

  /**
   * Error message from the backend
   * Should be human-readable or loggable text
   *
   * For RFC 9457 Problem Details responses, this would correspond to 'detail' field.
   * Example: "Latitude out of valid range [-90, 90]"
   */
  message?: string;

  /**
   * Optional error type or code for programmatic handling
   * For RFC 9457 Problem Details, this would correspond to 'type' field.
   * Example: "validation_error" or "https://api.example.com/problems/validation-error"
   */
  type?: string;

  /**
   * Raw error details if available (for logging/debugging)
   */
  details?: unknown;
}

/**
 * Response from sendPositions operation
 */
export interface ApiSendResponse {
  /**
   * HTTP status code of the response
   * - 202: Success (backend queued for processing)
   * - 4xx: Validation or client error (don't retry)
   * - 5xx: Server error (should retry)
   * - Other numbers: Network errors or timeouts
   */
  status: number;

  /**
   * Error information if the request failed
   * Undefined if status indicates success
   */
  error?: ApiErrorResponse;
}

/**
 * Port interface for HTTP communication with the backend API
 *
 * Abstracts the HTTP client from the application layer, allowing the application to:
 * 1. Send position data to the backend without knowing HTTP details
 * 2. Handle responses consistently across the application
 * 3. Support different implementations (real HTTP, mock for testing, etc.)
 *
 * This interface supports:
 * - RF-03-MOB (Envío inmediato después de captura)
 * - RF-04 (Ingesta asíncrona de telemetría)
 * - Offline-first operation with retry logic
 *
 * Key design decisions:
 * - sendPositions returns the HTTP status code explicitly
 * - The application layer decides whether to retry based on status code
 * - No automatic retries in the client (retries are coordinated by use cases)
 * - 202 Accepted is the success indicator (not 200)
 */
export interface ApiClient {
  /**
   * Send one or more position records to the backend API for ingestion.
   *
   * The backend ingestion endpoint should:
   * 1. Accept the positions in the HTTP request body
   * 2. Publish each position to Kafka for asynchronous processing
   * 3. Return 202 Accepted immediately without waiting for persistence
   *
   * Implementations must:
   * - Include authentication headers (if required)
   * - Include trace_id and idempotency_key for request tracking
   * - Handle network timeouts gracefully
   * - NOT automatically retry on failure (let the application layer decide)
   *
   * @param payloads - Array of PositionPayload objects to send
   *                   May contain 1 to N positions depending on the use case:
   *                   - Immediate send after capture: typically 1 payload
   *                   - Batch send after reconnection: may contain multiple payloads
   *
   * @returns Promise that resolves to ApiSendResponse with:
   *          - status: The HTTP status code received from the backend
   *          - error: Optional error details if status indicates failure
   *
   *          Success (202 Accepted):
   *          {
   *            status: 202,
   *            error: undefined  // No error for success
   *          }
   *
   *          Validation error (422):
   *          {
   *            status: 422,
   *            error: {
   *              status: 422,
   *              message: "Latitude out of valid range [-90, 90]",
   *              type: "validation_error"
   *            }
   *          }
   *
   *          Server error (500):
   *          {
   *            status: 500,
   *            error: {
   *              status: 500,
   *              message: "Internal server error",
   *              type: "internal_error"
   *            }
   *          }
   *
   *          Network error / Timeout:
   *          {
   *            status: 0,  // 0 indicates network error, not HTTP status
   *            error: {
   *              message: "Network request timeout",
   *              type: "network_error"
   *            }
   *          }
   *
   * @throws Never throws; all errors are returned in the response object
   *         This allows the application layer to handle all outcomes uniformly
   *
   * @example
   * // Send single position after capture
   * const response = await apiClient.sendPositions([position.toPayload()]);
   *
   * if (response.status === 202) {
   *   // Success: backend queued the position
   *   console.log('Position sent successfully');
   * } else if (response.status >= 400 && response.status < 500) {
   *   // Validation error: don't retry
   *   console.error('Validation error:', response.error?.message);
   * } else {
   *   // Other error: could retry
   *   console.warn('Server error:', response.error?.message);
   * }
   *
   * @example
   * // Send batch of positions after reconnection
   * const pendingPositions = await persistence.getPositionsPendingRetry();
   * const payloads = pendingPositions.map(p => p.toPayload());
   * const response = await apiClient.sendPositions(payloads);
   */
  sendPositions(payloads: PositionPayload[]): Promise<ApiSendResponse>;
}
