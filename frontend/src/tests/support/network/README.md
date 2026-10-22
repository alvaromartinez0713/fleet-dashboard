# Network Interception (MSW - Mock Service Worker)

## Overview

This directory contains **network-level request interception** for Fleet Dashboard tests using **Mock Service Worker (MSW)**, following **testing.md §8**.

**Core Principle:** Tests intercept real HTTP requests at the network layer. The application makes real `fetch` requests, MSW intercepts them, and returns mock data. The application has zero awareness it's being tested.

```
❌ Anti-pattern (breaks on refactoring):
vi.mock('axios');  // If we switch to fetch, tests break

✅ Recommended (resilient):
MSW intercepts at network layer  // Works with any HTTP library
```

---

## Quick Start

### 1. Your First Test

```typescript
// frontend/src/tests/presentation/pages/Dashboard.test.tsx
import { render, screen, waitFor } from '@testing-library/react';
import { renderInApp } from '@tests/support/test-harness';
import { Dashboard } from '@presentation/pages/Dashboard';

describe('Dashboard', () => {
  it('displays vehicles', async () => {
    renderInApp(<Dashboard />);
    
    // MSW intercepts GET /api/vehicles automatically
    // Handler returns 5 mock vehicles
    
    await waitFor(() => {
      expect(screen.getByText('VH-001')).toBeInTheDocument();
    });
  });
});
```

### 2. Override Handler for Error Testing

```typescript
import { http } from 'msw';
import { server } from '@tests/support/network/server';
import { ProblemDetailsTemplates } from '@tests/support/network/problem-details';

it('handles API error', async () => {
  // Override default handler for this test
  server.use(
    http.get('/api/vehicles', () => {
      return ProblemDetailsTemplates.internalError();
    })
  );
  
  renderInApp(<Dashboard />);
  
  await waitFor(() => {
    expect(screen.getByText(/error/i)).toBeInTheDocument();
  });
});
```

### 3. Verify Request Headers

```typescript
it('sends trace ID header', async () => {
  let capturedHeaders: Record<string, string> = {};
  
  server.use(
    http.get('/api/vehicles', ({ request }) => {
      capturedHeaders = Object.fromEntries(request.headers);
      return HttpResponse.json([]);
    })
  );
  
  renderInApp(<Dashboard />);
  
  await waitFor(() => {
    expect(capturedHeaders['x-trace-id']).toBeDefined();
  });
});
```

---

## Files

- **`handlers.ts`** - MSW request handlers for all API endpoints
- **`server.ts`** - MSW Node.js server configuration
- **`problem-details.ts`** - RFC 9457 Problem Details helper
- **`README.md`** - This file (documentation + patterns + troubleshooting)

---

## Available Endpoints

### Success Endpoints
```
GET /api/vehicles                    → 200 with paginated list (5 vehicles)
GET /api/vehicles/:vehicleId         → 200 with single vehicle
GET /api/vehicles/:vehicleId/alerts  → 200 with 2 alert objects
GET /api/vehicles/updates            → 200 SSE stream
```

### Error Endpoints (for testing)
```
GET /api/vehicles/error/429  → 429 Rate Limit Exceeded
GET /api/vehicles/error/500  → 500 Internal Server Error
GET /api/vehicles/error/503  → 503 Service Unavailable
```

All error responses use RFC 9457 format with `Content-Type: application/problem+json`.

---

## RFC 9457 Problem Details

All error responses follow the standard format:

```json
{
  "type": "https://api.fleet-dashboard.local/problems/error",
  "title": "Internal Server Error",
  "status": 500,
  "detail": "An unexpected error occurred. Please try again later.",
  "instance": "/problems/occurrences/1234567890"
}
```

### Available Templates

```typescript
import { ProblemDetailsTemplates } from '@tests/support/network/problem-details';

// 404 Not Found
ProblemDetailsTemplates.notFound('Vehicle', 'VH-001')

// 422 Validation Error
ProblemDetailsTemplates.validationError('latitude', 'must be between -90 and 90')

// 429 Rate Limit
ProblemDetailsTemplates.rateLimitExceeded(12, '60 seconds')

// 500 Server Error
ProblemDetailsTemplates.internalError()

// 503 Unavailable
ProblemDetailsTemplates.serviceUnavailable('Backend API')

// 401 Unauthorized
ProblemDetailsTemplates.unauthorized()

// 403 Forbidden
ProblemDetailsTemplates.forbidden()
```

---

## Server-Sent Events (SSE)

The `GET /api/vehicles/updates` handler simulates a real SSE stream:

```typescript
import { render, waitFor, screen } from '@testing-library/react';

it('receives position updates via SSE', async () => {
  render(<Dashboard />);
  
  // MSW starts streaming position updates
  // Component subscribes to the stream
  
  await waitFor(() => {
    expect(screen.getByText(/updated/i)).toBeInTheDocument();
  }, { timeout: 3000 });
});
```

---

## How MSW Works in Tests

### 1. Test Startup (setup.ts)
```typescript
beforeAll(() => {
  server.listen({ onUnhandledRequest: 'error' });
  // MSW now intercepts ALL HTTP requests
  // Unmocked requests will fail with error
});

afterEach(() => {
  server.resetHandlers();
  // Reset to default handlers
});

afterAll(() => {
  server.close();
  // Cleanup
});
```

### 2. Test Execution
```typescript
// Application code makes a real fetch request
const response = await fetch('/api/vehicles');

// MSW intercepts it at network layer
// ↓
// Handler processes the request
// ↓
// Returns mock data
// ↓
// Application receives response (thinks it's from real server)
```

### 3. Key Points
- ✅ Real `fetch` calls (not mocked libraries)
- ✅ Application has zero awareness of testing
- ✅ Resistant to library refactoring
- ✅ Fast feedback (no real network)
- ✅ Deterministic mock data

---

## Common Patterns

### Pattern 1: Happy Path (Default Data)
```typescript
it('displays data from valid API', async () => {
  // Uses default handler (returns 5 vehicles)
  renderInApp(<Dashboard />);
  
  await waitFor(() => {
    expect(screen.getByText('VH-001')).toBeInTheDocument();
  });
});
```

### Pattern 2: Override Handler
```typescript
it('shows error when API fails', async () => {
  server.use(
    http.get('/api/vehicles', () => {
      return ProblemDetailsTemplates.internalError();
    })
  );
  
  renderInApp(<Dashboard />);
  
  await waitFor(() => {
    expect(screen.getByText(/error/i)).toBeInTheDocument();
  });
});
```

### Pattern 3: Capture Request Details
```typescript
it('sends correct headers', async () => {
  const requests: Request[] = [];
  
  server.use(
    http.get('/api/vehicles', ({ request }) => {
      requests.push(request);
      return HttpResponse.json([]);
    })
  );
  
  renderInApp(<Dashboard />);
  
  await waitFor(() => {
    expect(requests).toHaveLength(1);
    expect(requests[0].headers.get('x-trace-id')).toBeDefined();
  });
});
```

### Pattern 4: Simulate Network Failure
```typescript
it('reconnects when network fails', async () => {
  server.use(
    http.get('/api/vehicles/updates', () => {
      return HttpResponse.error();  // Simulate network error
    })
  );
  
  renderInApp(<Dashboard />);
  
  await waitFor(() => {
    expect(screen.getByText(/reconnecting/i)).toBeInTheDocument();
  });
});
```

---

## Troubleshooting

### ❌ Error: `[MSW] Unhandled GET /api/...`

**Cause:** Endpoint not mocked

**Solution:** Add handler in `handlers.ts`
```typescript
export const handlers = [
  // Add your handler here
  http.get('/api/new-endpoint', () => {
    return HttpResponse.json({ data: 'mock' });
  }),
];
```

### ❌ Test hangs waiting for response

**Cause:** Handler doesn't return `HttpResponse`

**Solution:** Verify handler returns response
```typescript
// ❌ Wrong
http.get('/api/vehicles', () => {
  console.log('This does nothing');
  // Missing: return HttpResponse.json(...)
});

// ✅ Correct
http.get('/api/vehicles', () => {
  return HttpResponse.json([]);
});
```

### ❌ Handler override doesn't work

**Cause:** Called after render

**Solution:** Call `server.use()` BEFORE rendering
```typescript
// ❌ Wrong
render(<Dashboard />);
server.use(handler);

// ✅ Correct
server.use(handler);
render(<Dashboard />);
```

### ❌ Imports not resolved

**Cause:** Wrong path

**Solution:** Use correct import
```typescript
// ❌ Wrong
import { server } from '@tests/mocks/server';  // Old path

// ✅ Correct
import { server } from '@tests/support/network/server';
```

---

## Verification Checklist

Before running tests, verify:

- [x] `frontend/src/tests/setup.ts` imports `support/network/server`
- [x] `beforeAll()` has `onUnhandledRequest: 'error'`
- [x] No `frontend/src/tests/mocks/` directory (old structure)
- [x] Build passes: `cd frontend && npm run build`
- [x] Handlers exported as array in `handlers.ts`

---

## Next Steps

1. **Implement Object Mothers** → Replace hardcoded mock data
   ```typescript
   // Instead of:
   const mockVehicles = [{ vehicleId: 'VH-001', ... }];
   
   // Use:
   const mockVehicles = VehicleMother.validMultiple(5);
   ```

2. **Create Presentation Tests** → User flow tests
   ```typescript
   // frontend/src/tests/presentation/pages/VehicleDashboard.test.tsx
   ```

3. **Validate Against OpenAPI** → When contract available
   - Handlers must comply with schema
   - Tests verify responses match contract

---

## References

- [MSW Documentation](https://mswjs.io/)
- [RFC 9457 - Problem Details](https://www.rfc-editor.org/rfc/rfc9457.html)
- [testing.md - §8 Network Interception](../../../.kiro/steering/testing.md)
- [testing.md - §6 Test Harness](../../../.kiro/steering/testing.md)
- [api-standards.md - RFC 9457 Implementation](../../../.kiro/steering/api-standards.md)

---

## Status

✅ MSW configured and working  
✅ 7 endpoints mocked  
✅ RFC 9457 error responses  
✅ SSE support  
✅ Ready for tests
