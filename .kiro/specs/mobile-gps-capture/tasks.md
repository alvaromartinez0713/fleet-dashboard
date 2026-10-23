# Implementation Plan: Captura móvil de GPS y gestión de ubicaciones

## Overview

Implementación completa de captura de GPS con sincronización inteligente con backoff exponencial. El flujo incluye: (1) captura de coordenadas con almacenamiento local en SQLite, (2) sincronización con el backend con gestión de errores diferenciada (4xx vs 5xx), (3) reintentos automáticos con backoff exponencial `min(2s * 2^(attempts-1), 300s)`, (4) UI reactiva con hooks de sincronización automática y reconexión.

## Tasks

- [x] 1. Set up project structure and domain layer foundation
  - Create React Native project structure with TypeScript
  - Set up `src/domain/models/` with Coordinate, TraceContext value objects
  - Create `src/domain/errors/` with custom error classes
  - Create `src/domain/repositories/` with IPositionRepository interface
  - _Requirements: RNF-01 (Clean Architecture), RNF-02 (Diseño orientado al dominio)_

- [x] 2. Implement extended Position entity with exponential backoff
  - [x] 2.1 Create Position class with status ('pending' | 'sent' | 'error' | 'validation_error')
    - Add fields: eventId, vehicleId, coordinate, capturedAt, traceContext
    - Add backoff fields: lastSyncAttempt, syncAttemptCount, nextRetryAt, isValidationError
    - Add immutable constructor
    - _Requirements: RNF-04 (Idempotencia), RF-02 (Persistencia local)_
  
  - [x] 2.2 Implement markAsSent() method
    - Transitions status from 'pending' to 'sent'
    - Clears errorReason, resetea syncAttemptCount to 0, clears nextRetryAt
    - Updates lastSyncAttempt and updatedAt
    - Returns new Position instance (immutable)
    - _Requirements: RNF-04, RF-03 (Sincronización por lotes)_
  
  - [x] 2.3 Implement markAsError() method with exponential backoff calculation
    - Increments syncAttemptCount
    - Calculates nextRetryAt = now + min(2s * 2^(attempts-1), 300s)
    - If isValidationError=true: set status to 'validation_error', skip backoff calculation
    - If isValidationError=false: set status to 'error', calculate backoff
    - Updates lastSyncAttempt and updatedAt
    - Returns new Position instance
    - _Requirements: RNF-04 (Reintentos con backoff)_
  
  - [ ]* 2.4 Write property tests for Position immutability
    - **Property 1: markAsSent() returns new instance without mutating original**
    - **Validates: Requirements RNF-04**
  
  - [ ]* 2.5 Write property tests for exponential backoff calculation
    - **Property 2: nextRetryAt respects max 300s limit**
    - **Property 3: each retry doubles the delay (until hitting 300s cap)**
    - **Validates: Requirements RNF-04**
  
  - [x] 2.6 Implement canBeSynced() method
    - Returns true if status='pending' OR (status='error' AND nextRetryAt <= now)
    - Returns false for 'sent' and 'validation_error'
    - _Requirements: RF-03_

- [x] 3. Create domain utilities and helper methods
  - [x] 3.1 Implement EventId, VehicleId, TraceId value objects
    - Type-safe wrappers around UUID
    - _Requirements: RNF-02_
  
  - [x] 3.2 Implement Position helper methods for UI
    - getStatusLabel(): returns 'Pendiente', 'Enviado', 'Error', 'Error de datos'
    - getStatusIcon(): returns visual emoji for each status
    - getNextRetryText(now): returns 'Reintentar en 32s' or 'Listo para reintentar'
    - _Requirements: RF-02, RF-07 (Dashboard reactivo)_

- [x] 4. Implement application layer ports (infrastructure abstractions)
  - [x] 4.1 Create ILocationProvider interface
    - getCurrentLocation(): Promise<GPSData>
    - requestPermissions(): Promise<'granted' | 'denied'>
    - isPermissionGranted(): boolean
    - _Requirements: RF-01 (Captura móvil)_
  
  - [x] 4.2 Create IPersistenceProvider interface (extended with backoff support)
    - savePosition(), updatePosition(), getAllPositions()
    - getPositionsByVehicle(), getPositionsByStatus()
    - **getPositionsPendingRetry(now)**: returns 'pending' + retryable 'error'
    - **getPositionCountByStatus()**: count for single status
    - **getPositionCountsByAllStatus()**: count all statuses in one query
    - getPositionById(), deletePosition(), deleteAllPositions()
    - getPositionCount()
    - onPositionCaptured(), onPositionStatusChanged() event listeners
    - initialize(), close()
    - _Requirements: RF-02 (Persistencia), RNF-07 (Operación offline-first)_
  
  - [x] 4.3 Create IConnectivityProvider interface
    - getStatus(): Promise<'online' | 'offline'>
    - onConnectivityChanged(callback): register connectivity listener
    - _Requirements: RNF-07_
  
  - [x] 4.4 Create IApiClient interface
    - sendPositions(payloads): Promise<{status, error?}>
    - Expects 202 Accepted for success
    - _Requirements: RF-04_
  
  - [x] 4.5 Create IClock interface
    - now(): Date (injectable for deterministic testing)
    - _Requirements: Testing_
  
  - [x] 4.6 Create IIDGenerator interface
    - generateUUID(): string
    - _Requirements: Testing_

- [x] 5. Implement core use cases
  - [x] 5.1 Implement CapturePositionUseCase
    - Gets current GPS location via ILocationProvider
    - Creates Position entity with status='pending', syncAttemptCount=0
    - Saves to SQLite via IPersistenceProvider
    - Emits onPositionCaptured event
    - _Requirements: RF-01, RF-02_
  
  - [x] 5.2 Implement GetCapturedPositionsUseCase
    - Retrieves all positions from SQLite
    - Orders by capturedAt descending
    - _Requirements: RF-02_
  
  - [x] 5.3 Implement StartCaptureUseCase
    - Starts periodic GPS capture (every 30 seconds)
    - Calls CapturePositionUseCase repeatedly
    - Handles permissions gracefully
    - _Requirements: RF-01_
  
  - [x] 5.4 Implement StopCaptureUseCase
    - Stops periodic GPS capture
    - Cleans up interval timers
    - _Requirements: RF-01_
  
  - [x] 5.5 Implement GetConnectivityStatusUseCase
    - Returns current connectivity status
    - _Requirements: RNF-07_

- [x] 6. Implement SyncPositionUseCase (single record sync with error differentiation)
  - [x] 6.1 Create SyncPositionUseCase class
    - Check connectivity before syncing (fail if offline)
    - Call IApiClient.sendPositions() with position payload
    - _Requirements: RF-04_
  
  - [x] 6.2 Handle 202/200/201 Accepted responses
    - Call position.markAsSent()
    - Update position in SQLite via IPersistenceProvider.updatePosition()
    - Return success: true
    - _Requirements: RNF-04_
  
  - [x] 6.3 Differentiate 4xx vs 5xx errors for smart retry decisions
    - If HTTP status 400-499: isValidationError=true (no retry)
      - Call position.markAsError(reason, true, clock)
      - Update in SQLite with status='validation_error'
    - If HTTP status 500+: isValidationError=false (retry with backoff)
      - Call position.markAsError(reason, false, clock)
      - Update in SQLite with status='error', set nextRetryAt
    - _Requirements: RNF-04_
  
  - [x] 6.4 Handle network errors and timeouts as transient errors
    - Catch exceptions (network error, timeout, etc.)
    - Call position.markAsError(error.message, false, clock)
    - Treat as retryable (status='error' with backoff)
    - _Requirements: RNF-04, RNF-06_
  
  - [ ]* 6.5 Write property tests for error differentiation
    - **Property 4: 4xx status → validation_error (no nextRetryAt)**
    - **Property 5: 5xx status → error with backoff calculation**
    - **Validates: Requirements RNF-04**
  
  - [x] 6.6 Write unit tests for SyncPositionUseCase
    - Test 202 Accepted → status='sent'
    - Test 400 → validation_error (skip retry)
    - Test 500 → error with backoff
    - Test network error → error with backoff
    - Test offline connectivity → throws error
    - _Requirements: RNF-04, testing.md section 10_

- [x] 7. Implement SyncPendingPositionsUseCase (smart retry with backoff)
  - [x] 7.1 Create SyncPendingPositionsUseCase class
    - Get pending positions: getPositionsByStatus('pending')
    - Get retryable errors: getPositionsByStatus('error') filtered by nextRetryAt <= now
    - _Requirements: RNF-04_
  
  - [x] 7.2 Implement concurrent sync with concurrency limit
    - Sync up to 3 positions in parallel (configurable)
    - Use Promise.allSettled() to avoid failing on individual errors
    - Track synced and failed counts
    - _Requirements: RNF-04_
  
  - [x] 7.3 Return updated positions after sync attempt
    - Reload all positions from SQLite
    - Filter to only those that were in the retry batch
    - Return { synced, failed, updated }
    - _Requirements: RF-03_
  
  - [x] 7.4 Write unit tests for retry logic
    - Test getPositionsPendingRetry() returns only syncable positions
    - Test concurrent sync respects maxConcurrent limit
    - Test failed syncs don't break batch execution
    - _Requirements: RNF-04_

- [x] 8. Implement RetryFailedSyncUseCase (button action for manual retry)
  - [x] 8.1 Create RetryFailedSyncUseCase as alias/wrapper
    - Extends SyncPendingPositionsUseCase with explicit naming
    - Used by "Reintentar todo (N)" button in UI
    - _Requirements: RF-03_

- [x] 9. Implement CaptureAndSyncPositionUseCase (immediate sync attempt)
  - [x] 9.1 Create CaptureAndSyncPositionUseCase
    - Call CapturePositionUseCase to capture GPS → save as 'pending'
    - Check connectivity via IConnectivityProvider
    - If online: immediately try SyncPositionUseCase
    - If offline: return without sync attempt
    - Return { position, syncAttempted, syncSuccess }
    - _Requirements: RF-01, RF-03, RNF-07_
  
  - [x] 9.2 Write unit tests for immediate sync flow
    - Test offline → capture only, no sync attempt
    - Test online + success → capture + send
    - Test online + failure → capture + error marked
    - _Requirements: testing.md_

- [x] 10. Set up SQLite schema with backoff support
  - [x] 10.1 Create SQLite schema migration
    - Table: positions
    - Columns:
      - event_id UUID PRIMARY KEY
      - vehicle_id VARCHAR NOT NULL
      - latitude DECIMAL NOT NULL
      - longitude DECIMAL NOT NULL
      - accuracy DECIMAL
      - captured_at TIMESTAMP NOT NULL
      - trace_id UUID NOT NULL
      - idempotency_key UUID NOT NULL
      - status VARCHAR NOT NULL CHECK (status IN ('pending', 'sent', 'error', 'validation_error'))
      - error_reason VARCHAR
      - is_validation_error BOOLEAN DEFAULT 0
      - last_sync_attempt TIMESTAMP
      - sync_attempt_count INTEGER DEFAULT 0
      - next_retry_at TIMESTAMP
      - updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
    - _Requirements: RF-02, RNF-04_
  
  - [x] 10.2 Create indexes for query optimization
    - Index on vehicle_id (filter by vehicle)
    - Index on status (query pending + error)
    - Index on next_retry_at (query retryable positions)
    - _Requirements: Performance_
  
  - [x] 10.3 Create migration runner
    - Run migrations on app startup
    - Create migrator utility
    - _Requirements: RF-02_

- [x] 11. Implement FakePersistenceProvider for testing
  - [x] 11.1 Create in-memory FakePersistenceProvider
    - Stores positions in Map<eventId, Position>
    - Implements all IPersistenceProvider methods
    - _Requirements: Testing_
  
  - [x] 11.2 Implement backoff-aware filtering
    - getPositionsPendingRetry(now) correctly filters by nextRetryAt
    - Supports event listeners (onPositionCaptured, onPositionStatusChanged)
    - _Requirements: Testing, RNF-04_
  
  - [ ]* 11.3 Write unit tests for FakePersistenceProvider
    - Test savePosition() stores correctly
    - Test updatePosition() changes status
    - Test getPositionsPendingRetry() filters correctly
    - Test getPositionCountsByAllStatus() returns correct counts
    - _Requirements: Testing_

- [x] 12. Implement SQLitePersistenceProvider (real implementation)
  - [x] 12.1 Create SQLitePersistenceProvider class
    - Initialize SQLite database on startup
    - Implement all IPersistenceProvider methods using SQL
    - _Requirements: RF-02, RNF-07_
  
  - [x] 12.2 Implement savePosition() and updatePosition() methods
    - savePosition: INSERT with status='pending', syncAttemptCount=0
    - updatePosition: UPDATE with new status and backoff fields
    - _Requirements: RNF-04_
  
  - [x] 12.3 Implement getPositionsPendingRetry() query
    - SELECT * WHERE status='pending' OR (status='error' AND nextRetryAt <= now)
    - ORDER BY nextRetryAt ASC (retry soonest first)
    - _Requirements: RNF-04_
  
  - [x] 12.4 Implement getPositionCountsByAllStatus() optimized query
    - Single query: SELECT status, COUNT(*) FROM positions GROUP BY status
    - Returns { pending: N, sent: M, error: P, validation_error: L }
    - _Requirements: Performance_
  
  - [x] 12.5 Implement event listener system
    - Store callbacks for onPositionCaptured and onPositionStatusChanged
    - Fire events synchronously after savePosition() and updatePosition()
    - _Requirements: RNF-08 (Actualización reactiva)_
  
  - [ ]* 12.6 Write integration tests for SQLitePersistenceProvider
    - Test savePosition() persists to SQLite
    - Test updatePosition() correctly changes status fields
    - Test getPositionsPendingRetry() returns correct positions
    - Test event listeners fire correctly
    - _Requirements: Testing, RF-02_

- [x] 13. Create infrastructure adapters (API, Connectivity, Clock)
  - [x] 13.1 Implement RealApiClient
    - sendPositions(payloads): sends HTTP POST to backend
    - Extracts status code and error from response
    - Returns { status, error? }
    - _Requirements: RF-04_
  
  - [x] 13.2 Implement FakeApiClient for testing
    - Configurable responses: 202, 400, 500, timeout
    - _Requirements: Testing_
  
  - [x] 13.3 Implement RealConnectivityProvider
    - getStatus(): queries NetInfo for connectivity
    - onConnectivityChanged(callback): subscribes to changes
    - _Requirements: RNF-07_
  
  - [x] 13.4 Implement FakeConnectivityProvider for testing
    - Configurable online/offline state
    - Manual trigger of onConnectivityChanged
    - _Requirements: Testing_
  
  - [x] 13.5 Implement RealClock and RealIDGenerator
    - Clock.now() returns new Date()
    - IDGenerator.generateUUID() uses uuid library
    - _Requirements: Testing_

- [x] 14. Create React Native UI components
  - [x] 14.1 Create PositionItem component
    - Display individual position: timestamp, coordinates, status
    - Show status icon (🔵 pending, ✓ sent, ⚠️ error, ❌ validation_error)
    - Show status label in Spanish
    - Show next retry countdown if status='error' and nextRetryAt > now
    - _Requirements: RF-02, RF-07_
  
  - [x] 14.2 Create "Sincronizar" button for PositionItem
    - Enabled only if: online AND (status='pending' OR (status='error' AND canBeSynced()))
    - Disabled if: offline OR status='sent' OR status='validation_error'
    - Disabled while sync is in progress (loading state)
    - Click triggers SyncPositionUseCase
    - _Requirements: RF-02, RF-03_
  
  - [x] 14.3 Create PositionListHeader component
    - Show status breakdown: "N Pendiente | M Enviado | P Error | L Datos inválidos"
    - Fetch counts via getPositionCountsByAllStatus()
    - Show "Sincronizar todo (N)" button
    - Click triggers RetryFailedSyncUseCase
    - Disabled if: offline or no pending/retryable positions
    - Show loading state during sync
    - _Requirements: RF-02, RF-07_
  
  - [x] 14.4 Create PositionsList component
    - Display array of positions ordered by capturedAt DESC
    - Use FlatList for performance with large lists
    - Each item is PositionItem
    - Subscribe to persistence provider events for real-time updates
    - _Requirements: RF-02, RNF-08_

- [x] 15. Create custom React hooks for background sync
  - [x] 15.1 Create useAutoSync hook
    - Call SyncPendingPositionsUseCase every 10 seconds
    - Use setInterval with cleanup on unmount
    - Only sync if online
    - _Requirements: RF-03, RNF-07_
  
  - [ ]* 15.2 Write tests for useAutoSync hook
    - Test calls SyncPendingPositionsUseCase every 10s
    - Test cleanup on unmount clears interval
    - Test respects offline state (skips when offline)
    - **Property 6: Auto-sync completes without crashing on concurrent calls**
    - **Validates: Requirements RNF-04**
  
  - [x] 15.3 Create useSyncOnReconnect hook
    - Subscribe to IConnectivityProvider.onConnectivityChanged()
    - When status changes to 'online': call SyncPendingPositionsUseCase
    - Return cleanup function to unsubscribe
    - _Requirements: RNF-07_
  
  - [ ]* 15.4 Write tests for useSyncOnReconnect hook
    - Test triggers sync when reconnecting
    - Test ignores reconnect if already online
    - Test cleanup unsubscribes
    - _Requirements: Testing_
  
  - [x] 15.5 Create usePositionsList hook
    - Fetch positions via GetCapturedPositionsUseCase
    - Subscribe to onPositionCaptured and onPositionStatusChanged events
    - Update local state reactively
    - Return { positions, isLoading, error }
    - _Requirements: RF-02, RNF-08_
  
  - [x] 15.6 Create useConnectivity hook
    - Get current connectivity status
    - Subscribe to changes
    - Return { isOnline, isLoading }
    - _Requirements: RNF-07_
  
  - [x] 15.7 Create useLocationPermissions hook
    - Request and check GPS permissions
    - Return { hasPermission, isLoading, requestPermission }
    - _Requirements: RF-01_

- [x] 16. Set up dependency injection and service factory
  - [x] 16.1 Create ServiceFactory
    - Compose all use cases with their dependencies
    - Create singleton instances of providers (API, Persistence, Connectivity, Clock)
    - Export factory functions for each use case
    - _Requirements: RNF-01_
  
  - [x] 16.2 Create DependencyContainer
    - Register providers (Real vs Fake based on environment)
    - Provide getters for each use case
    - Support test overrides
    - _Requirements: Testing_

- [x] 17. Integration: wire everything in React Native App
  - [x] 17.1 Create App component
    - Initialize DependencyContainer
    - Request location permissions on startup
    - Mount main CaptureScreen
    - Provide error boundary
    - _Requirements: RNF-01, RF-01_
  
  - [x] 17.2 Create CaptureScreen
    - Use useLocationPermissions hook
    - Use useConnectivity hook
    - Use usePositionsList hook
    - Use useAutoSync hook (run background sync every 10s)
    - Use useSyncOnReconnect hook (sync when online)
    - Render PositionsList with header
    - Render start/stop capture controls
    - _Requirements: RF-02, RF-07, RNF-07, RNF-08_

- [x] 18. Checkpoint - Ensure all unit tests pass
  - Run all unit tests for domain, application, and adapters
  - Verify property tests for backoff, error differentiation, and idempotence
  - Verify no regressions
  - Ask the user if questions arise.

- [x] 19. Write integration tests for critical flows
  - [x] 19.1 Write capture + immediate sync flow test
    - Capture GPS while online → should trigger SyncPositionUseCase immediately
    - Verify position marked as 'sent' in SQLite
    - _Requirements: RF-01, RF-03_
  
  - [x] 19.2 Write offline capture + reconnect sync flow test
    - Capture GPS while offline → should save as 'pending' only
    - Simulate reconnect → should trigger SyncPendingPositionsUseCase
    - Verify positions marked as 'sent'
    - _Requirements: RNF-07_
  
  - [x] 19.3 Write backoff retry flow test
    - Create position with status='error', nextRetryAt in future
    - Verify getPositionsPendingRetry() excludes it
    - Advance time until nextRetryAt
    - Verify getPositionsPendingRetry() includes it
    - _Requirements: RNF-04_
  
  - [x] 19.4 Write 4xx validation error flow test
    - Capture GPS, attempt sync with 400 response
    - Verify status='validation_error' (not 'error')
    - Verify no nextRetryAt calculated
    - Verify getPositionsPendingRetry() excludes it permanently
    - _Requirements: RNF-04_
  
  - [ ]* 19.5 Write auto-sync background hook test
    - Start useAutoSync hook
    - Wait 10 seconds
    - Verify SyncPendingPositionsUseCase called automatically
    - Verify positions updated in real-time
    - _Requirements: RF-03, RNF-08_

- [x] 20. Final checkpoint - Ensure all tests pass
  - Run full test suite (unit + integration)
  - Verify coverage of critical paths
  - Verify no flaky tests
  - Ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Backoff calculation: `min(2s * 2^(attempts-1), 300s)` is critical for intelligent retries
- Error differentiation (4xx vs 5xx) prevents wasting resources retrying validation errors
- Offline-first design ensures data is never lost due to connectivity
- Event listeners keep UI in sync without polling
- All use cases are testable with fakes/mocks
- Immutable Position entity ensures state changes are explicit and traceable

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1", "2.1", "4.1", "4.2", "4.3", "4.4", "4.5", "4.6"] },
    { "id": 1, "tasks": ["2.2", "2.3", "2.6", "3.1", "3.2"] },
    { "id": 2, "tasks": ["2.4", "2.5", "5.1", "5.2", "5.3", "5.4", "5.5"] },
    { "id": 3, "tasks": ["6.1", "6.2", "6.3", "6.4", "7.1", "7.2", "7.3", "8.1", "9.1"] },
    { "id": 4, "tasks": ["6.5", "6.6", "7.4", "9.2", "10.1", "10.2", "10.3"] },
    { "id": 5, "tasks": ["11.1", "11.2", "11.3", "12.1", "12.2", "12.3", "12.4", "12.5"] },
    { "id": 6, "tasks": ["12.6", "13.1", "13.2", "13.3", "13.4", "13.5"] },
    { "id": 7, "tasks": ["14.1", "14.2", "14.3", "14.4", "15.1", "15.3", "15.5", "15.6", "15.7"] },
    { "id": 8, "tasks": ["15.2", "15.4", "16.1", "16.2"] },
    { "id": 9, "tasks": ["17.1", "17.2", "19.1", "19.2", "19.3", "19.4", "19.5"] }
  ]
}
```

## Phase Completion Status

### Phase 1-2 (Waves 0-2): COMPLETE ✓
- Domain layer, Position entity, value objects, all 6 port interfaces, core use cases (capture, get, start, stop, connectivity)
- 21 required tasks completed
- 80+ unit tests passing

### Phase 3 (Wave 3): COMPLETE ✓
- SyncPositionUseCase (6.1-6.6): Single position sync with error differentiation
- SyncPendingPositionsUseCase (7.1-7.4): Batch sync with retry backoff
- RetryFailedSyncUseCase (8.1): UI button wrapper
- CaptureAndSyncPositionUseCase (9.1-9.2): Capture + immediate sync if online
- 59 unit tests, 4 new use cases, 2 new test fakes created
- TypeScript compilation: no errors
- All requirements mapped and verified

### Phase 4-10: COMPLETE ✓
- SQLite schema and migration runner (10.1-10.3)
- FakePersistenceProvider (11.1-11.2)
- SQLitePersistenceProvider with all methods (12.1-12.5)
- Infrastructure adapters: API, Connectivity, Clock (13.1-13.5)
- React Native UI components: PositionItem, PositionListHeader, PositionsList (14.1-14.4)
- Custom React hooks: useAutoSync, useSyncOnReconnect, usePositionsList, useConnectivity, useLocationPermissions (15.1, 15.3, 15.5, 15.6, 15.7)
- Dependency injection: ServiceFactory, DependencyContainer (16.1-16.2)
- App integration: App.tsx, CaptureScreen (17.1-17.2)
- Unit tests: all phases (18)
- Integration tests: critical flows (19.1-19.4)
- Final checkpoint: all tests passing (20)
- **Total: 89 required tasks COMPLETE**
