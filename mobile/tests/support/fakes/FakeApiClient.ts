import { IApiClient, ApiSendResponse, PositionPayload } from '../../../src/application/ports/IApiClient';

/**
 * Configuration for FakeApiClient response behavior
 */
export interface FakeApiClientConfig {
  /**
   * HTTP status code to return (202, 400, 500, etc.)
   */
  status: number;

  /**
   * Error message if status indicates failure
   */
  errorMessage?: string;

  /**
   * Whether to simulate a network timeout
   */
  shouldTimeout?: boolean;
}

/**
 * FakeApiClient
 *
 * Test implementation of IApiClient for testing sync use cases.
 * Allows tests to configure response behavior (success, validation error, server error, timeout).
 */
export class FakeApiClient implements IApiClient {
  private config: FakeApiClientConfig;
  callCount: number = 0;
  lastPayloads: PositionPayload[] = [];

  constructor(config: FakeApiClientConfig = { status: 202 }) {
    this.config = config;
  }

  /**
   * Update the response configuration
   */
  setConfig(config: FakeApiClientConfig): void {
    this.config = config;
  }

  /**
   * Send positions with configured response
   */
  async sendPositions(payloads: PositionPayload[]): Promise<ApiSendResponse> {
    this.callCount++;
    this.lastPayloads = payloads;

    if (this.config.shouldTimeout) {
      throw new Error('Request timeout');
    }

    return {
      status: this.config.status,
      error: this.config.status !== 202 && this.config.status !== 200 && this.config.status !== 201
        ? {
            status: this.config.status,
            message: this.config.errorMessage || `HTTP ${this.config.status}`,
            type: this.config.status >= 400 && this.config.status < 500 ? 'validation_error' : 'server_error'
          }
        : undefined
    };
  }

  /**
   * Reset call tracking for test isolation
   */
  reset(): void {
    this.callCount = 0;
    this.lastPayloads = [];
  }
}
