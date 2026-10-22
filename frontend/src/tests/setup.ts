import '@testing-library/jest-dom';
import { afterAll, afterEach, beforeAll, vi } from 'vitest';
import { server } from './support/network/server';

/**
 * Global Test Setup for Fleet Dashboard
 *
 * Following testing.md:
 * - Section 5.1: Real dependencies replaced with fakes/mocks
 * - Section 5.2: Integration test suite with MSW for network interception
 * - Section 8: Network interception without mocking client libraries
 * - Section 6: Harness that isolates framework coupling
 *
 * Responsibilities:
 * 1. Initialize MSW server before all tests
 * 2. Reset handlers between tests (ensures isolation)
 * 3. Close server after all tests (cleanup)
 * 4. Configure Vitest globals
 *
 * The MSW server intercepts ALL HTTP requests in tests:
 * - No real network calls are made
 * - All requests must be explicitly mocked or tests fail (onUnhandledRequest: 'error')
 * - This ensures network behavior is deterministic and fast (< 2s per test)
 */

/**
 * Start MSW server before all tests
 * Configuration:
 * - onUnhandledRequest: 'error': Fail tests if unmocked requests occur
 *   This catches bugs where tests forget to mock an endpoint
 * Following testing.md Section 8: Network interception
 */
beforeAll(() => {
  server.listen({ onUnhandledRequest: 'error' });
});

/**
 * Reset handlers after each test
 * Ensures test isolation - handler changes in one test don't affect others
 * Also clears all Vitest mocks (vi.fn() calls)
 * Following testing.md Section 5.2: Handler reset between tests
 */
afterEach(() => {
  server.resetHandlers();
  vi.clearAllMocks();
});

/**
 * Close server after all tests
 * Cleanup to prevent resource leaks and ensure clean shutdown
 */
afterAll(() => {
  server.close();
});
