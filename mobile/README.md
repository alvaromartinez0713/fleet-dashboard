# Mobile App - Fleet Dashboard GPS Capture

Aplicación móvil en React Native con TypeScript para captura automática de coordenadas GPS, almacenamiento local en SQLite y sincronización inteligente con exponential backoff.

## Arquitectura

La aplicación sigue **Clean Architecture** con cuatro capas principales:

```
PRESENTATION LAYER (React Native Components)
        ↓
APPLICATION LAYER (Use Cases, Coordinators)
        ↓
DOMAIN LAYER (Entities, Value Objects, Rules)
        ↓
INFRASTRUCTURE LAYER (Adapters: Location, Persistence, API, etc.)
```

Las dependencias siempre apuntan **hacia el dominio**, nunca desde el dominio hacia las capas externas.

## Estructura de Directorios

```
mobile/
├── src/
│   ├── domain/                    # Lógica de negocio pura (sin frameworks)
│   │   ├── models/
│   │   │   ├── Position.ts       # Entidad principal: ubicación capturada
│   │   │   ├── Coordinate.ts     # Objeto de valor: lat/lon/accuracy
│   │   │   └── TraceContext.ts   # Objeto de valor: trace_id, idempotency_key
│   │   ├── errors/
│   │   │   ├── DomainError.ts
│   │   │   ├── InvalidCoordinateError.ts
│   │   │   ├── PersistenceError.ts
│   │   │   └── LocationError.ts
│   │   └── repositories/
│   │       └── IPositionRepository.ts  # Contrato sin implementación
│   ├── application/               # Coordinación de operaciones
│   ├── infrastructure/            # Adaptadores e integraciones
│   ├── presentation/              # Componentes React Native
│   ├── services/                  # Inyección de dependencias
│   └── config/                    # Configuración por entorno
│
└── tests/
    ├── support/
    │   ├── mothers/               # Object Mother para datos de prueba
    │   └── fakes/                 # Implementaciones fake de adapters
    └── unit/
        └── domain/                # Pruebas unitarias del dominio
```

## Modelos de Dominio

### Position (Entidad)

Representa una ubicación GPS capturada, con estados y lógica de sincronización:

```typescript
// Estados posibles
type PositionStatus = 'pending' | 'sent' | 'error' | 'validation_error';

// Métodos principales
position.markAsSent()              // pending → sent (202 Accepted)
position.markAsError(reason, isValidation, clock)  // pending → error/validation_error
position.canBeSynced(now)          // ¿Está listo para sincronizar?

// Helpers para UI
position.getStatusLabel()          // "Pendiente", "Enviado", etc.
position.getStatusIcon()           // "🔵", "✓", "⚠️", "❌"
position.getNextRetryText(now)     // "Reintentar en 32s"
```

**Características clave:**
- Inmutable: todos los cambios retornan nuevas instancias
- Exponential backoff automático: `min(2s * 2^(attempts-1), 300s)`
- Diferencia entre errores transitorios (reintento) y de validación (sin reintento)

### Coordinate (Objeto de Valor)

Representa una coordenada geográfica validada:

```typescript
const coord = new Coordinate(
  40.4168,  // latitude: [-90, 90]
  -3.7038,  // longitude: [-180, 180]
  5         // accuracy: >= 0 (opcional)
);

coord.equals(otherCoord)  // Comparación por valor
```

### TraceContext (Objeto de Valor)

Contiene información de trazabilidad e idempotencia:

```typescript
const context = new TraceContext(
  traceId,        // UUID para correlación
  idempotencyKey  // UUID para deduplicación en backend
);
```

## Configuración y Uso

### Instalación

```bash
cd mobile
npm install
```

### Ejecución de Pruebas

```bash
# Pruebas unitarias
npm test

# Pruebas con coverage
npm test:coverage

# Pruebas en modo watch
npm test:watch
```

### Verificación de Tipos

```bash
npm run type-check
```

### Linting

```bash
npm run lint
```

## Requisitos Funcionales Cubiertos

### RF-01-MOB: Captura automática de GPS cada 30 segundos
- ✓ Estructura para LocationProvider interface
- ✓ Modelo Position con campos requeridos
- Implementación: Próximas tareas

### RF-02-MOB: Almacenamiento local persistente en SQLite
- ✓ Modelo Position con estados: 'pending', 'sent', 'error', 'validation_error'
- ✓ IPositionRepository interface
- ✓ Estructura para SQLitePersistenceProvider
- Implementación: Próximas tareas

### RF-03-MOB: Envío inmediato después de captura
- ✓ Campos en Position para rastrear intentos: lastSyncAttempt, syncAttemptCount
- ✓ Métodos markAsSent() y markAsError()
- Implementación: Próximas tareas

### RF-04-MOB: Manejo de 202 Accepted
- ✓ Método markAsSent() para transición pending → sent
- ✓ Preservación del registro en SQLite (no se borra)

### RF-05-MOB & RF-06-MOB: Diferenciación de errores y exponential backoff
- ✓ Parámetro isValidationError en markAsError()
- ✓ Cálculo automático de nextRetryAt con backoff exponencial
- ✓ Validación en canBeSynced()
- ✓ Método getNextRetryText() para UI

### RNF-01-MOB: Clean Architecture
- ✓ Dominio sin dependencias de frameworks
- ✓ Interfaces claramente definidas
- ✓ Separación de capas
- Implementación: Próximas tareas (adapters, use cases, UI)

## Estándares de Código

### Lenguaje y Estilo

- **Código**: Todo en inglés (nombres, variables, comentarios)
- **Mensajes de usuario**: En español
- **Documentación**: En español con términos técnicos en inglés

### Convenciones

- Clases y interfaces: PascalCase
- Variables y funciones: camelCase
- Constantes: UPPER_SNAKE_CASE
- Interfaces: prefijo `I` (e.g., `IPositionRepository`)
- Tipos: sufijo `Type` (e.g., `PositionStatus`)

### Pruebas

Seguimos `testing.md`:
- Object Mother para datos deterministas
- Fakes en lugar de mocks
- Pruebas enfocadas en comportamiento observable
- Sin snapshots
- Validación de contratos con el backend

## Object Mother para Datos de Prueba

Patrones para crear datos de test válidos y predecibles:

```typescript
// Coordenada válida
const coord = CoordinateMother.valid().withAccuracy(10).build();

// TraceContext determinístico
const trace = TraceContextMother.deterministic().build();

// Position con diferentes estados
const pending = PositionMother.valid().build();
const sent = PositionMother.sent().build();
const errored = PositionMother.withError().build();
const validation = PositionMother.withValidationError().build();
```

## Estado Actual: Checkpoint 1

✅ **Completado:**
- [x] Estructura de proyecto React Native + TypeScript
- [x] Configuración de TypeScript (tsconfig.json, path aliases)
- [x] Configuración de Jest para tests
- [x] Dominio puro sin dependencias de frameworks
  - [x] Entidad Position con exponential backoff
  - [x] Objeto de valor Coordinate validado
  - [x] Objeto de valor TraceContext
  - [x] Custom error classes (DomainError, InvalidCoordinateError, etc.)
- [x] Interfaces de repositorio (IPositionRepository)
- [x] Object Mother para datos de prueba
  - [x] PositionMother con escenarios
  - [x] CoordinateMother
  - [x] TraceContextMother
- [x] Pruebas unitarias del dominio
  - [x] Coordinate.test.ts: validación, equality, boundary cases
  - [x] TraceContext.test.ts: validación, equality
  - [x] Position.test.ts: state transitions, backoff, immutability

⏭️ **Próximas Tareas:**
- [ ] Task 2: Implementar métodos extendidos de Position (si no completados)
- [ ] Task 3: Value objects adicionales (EventId, VehicleId, TraceId)
- [ ] Task 4: Interfaces de aplicación (LocationProvider, PersistenceProvider, etc.)
- [ ] Task 5: Casos de uso (CapturePosition, GetCapturedPositions, etc.)
- [ ] Task 6+: Implementaciones reales (SQLite, API, Conectividad)
- [ ] UI: Componentes React Native y hooks

## Próximas Acciones

1. Ejecutar pruebas para validar comportamiento del dominio
2. Implementar casos de uso de aplicación
3. Crear adapters de infraestructura (Location, Persistence, API)
4. Implementar SQLitePersistenceProvider
5. Crear componentes React Native y hooks
6. Integración E2E offline → capture → sync

## Referencias

- `product.md`: Requisitos del producto
- `tech.md`: Stack tecnológico
- `structure.md`: Arquitectura general
- `testing.md`: Estándares de pruebas (secciones 10-12 para mobile)
- `security.md`: Estándares de seguridad

## Notas

- La aplicación está diseñada para operar **offline-first**: captura y almacena localmente, sincroniza cuando hay conectividad
- Todos los datos tienen `eventId` e `idempotencyKey` únicos para garantizar deduplicación en el backend
- El backoff exponencial previene sobrecargar el servidor en caso de fallos transitorios
- Las pruebas son deterministas (reloj inyectable, datos predecibles) para reproducibilidad
