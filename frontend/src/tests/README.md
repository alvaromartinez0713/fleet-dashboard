# Testing Strategy

## Principio Fundamental

> "No una prueba por cada clase o archivo 'porque sí'" (testing.md §22)

> "La unidad es un **flujo del usuario**, no un componente aislado" (testing.md §11)

---

## Estructura

```
tests/
├── domain/                          ← Pruebas unitarias (reglas de negocio)
│   └── entities/
│       ├── vehicle.test.ts         ✅ Invariantes, immutabilidad
│       ├── position.test.ts        ✅ Validación de coordenadas
│       └── vehicle-id.test.ts      ✅ Validación de IDs
│
├── application/                     ← Pruebas de integración (coordinación)
│   ├── usecases/                   ✅ Casos de uso importantes
│   └── state-manager/              ✅ Coordinación de estado
│
├── infrastructure/                  ← Pocas pruebas (adaptadores)
│   └── data-providers/             ⚠️ Solo si hay lógica compleja
│
├── presentation/                    ← UNA prueba de flujo de usuario
│   └── pages/
│       └── VehicleDashboard.test.tsx ✅ Comportamiento observable
│
├── mocks/                           (No tests - solo configuración)
│   ├── handlers.ts                  (MSW handlers por defecto)
│   └── server.ts                    (MSW setup)
│
├── support/                         (No tests - utilidades)
│   ├── test-harness.tsx            (renderInApp helper)
│   └── mothers/                    (Object Mother builders)
│
├── setup.ts                         (Global Vitest setup)
└── README.md                        (Este archivo)
```

---

## Qué SE y QUÉ NO SE Prueba

### ✅ PROBAR (Lógica de Negocio)

| Tipo | Ejemplo | Motivo |
|------|---------|--------|
| **Entidades del dominio** | `Vehicle`, `Position`, `VehicleId` | Contienen reglas críticas del negocio |
| **Validadores** | `PositionValidator` | Validan invariantes |
| **Casos de uso** | `LoadVehiclesUseCase` | Coordinan operaciones importantes |
| **Flujos de usuario** | Dashboard completo: carga → mapa → error | Comportamiento observable |

### ❌ NO PROBAR (Código Trivial)

| Tipo | Ejemplo | Motivo |
|------|---------|--------|
| **Componentes puros** | `ConnectionStatus`, `MapContainer` | Se prueban a través del flujo integrado |
| **Hooks coordinadores** | `useVehicles()`, `useSSEConnection()` | Se prueban observando su efecto en UI |
| **Adaptadores** | `ApiClient`, `SSEConnection` | Se prueban mediante MSW + flujos |
| **Componentes triviales** | `ErrorAlert`, `LoadingSpinner` | No contienen lógica |

---

## Los 4 Pilares (testing.md §2)

Cada test se evalúa con:

1. **Protección contra regresiones**: ¿Detectaría un bug real?
2. **Resistencia al refactoring**: ¿Pasa si cambio implementación interna?
3. **Feedback rápido**: ¿Se ejecuta en segundos?
4. **Mantenibilidad**: ¿Se entiende con poco código?

### Ejemplo: Vehicle.test.ts

```typescript
// ✅ Pilar 1: Detecta si desaparece validación de coords
it('rejects invalid coordinates', () => {
  expect(() => new Position(91, 0)).toThrow();
});

// ✅ Pilar 2: Pasa aunque refactorices internamente
it('returns new instance on updatePosition', () => {
  const updated = vehicle.updatePosition(newPosition);
  expect(updated).not.toBe(vehicle); // No muta
});

// ✅ Pilar 3: < 100ms, sin dependencias externas
// ✅ Pilar 4: Nombre claro, una asertión, propósito evidente
```

---

## Caja Negra (testing.md §4)

**Entrada:** Datos públicos del dominio o acciones del usuario  
**Salida observable:** Comportamiento verificable (resultado, error, UI visible)  
**NO se verifica:** Implementación interna, estado privado, detalles de framework

### Correcto

```typescript
// Entra por borde público, sale observando
it('displays map and vehicles when dashboard loads', async () => {
  renderInApp(<VehicleDashboard />);
  
  await waitFor(() => {
    expect(screen.getByTestId('dashboard-map-container')).toBeInTheDocument();
  });
});
```

### Incorrecto

```typescript
// Verifica implementación interna
const spy = vi.spyOn(useVehicles);
render(<VehicleDashboard />);
expect(spy).toHaveBeenCalled(); // ← NO!
```

---

## Test Harness: Aislar Framework Coupling (testing.md §6)

**Archivo:** `support/test-harness.tsx`

```typescript
// ANTES (acoplado):
render(
  <ApplicationProvider>
    <VehicleDashboard />
  </ApplicationProvider>
);

// DESPUÉS (desacoplado):
import { renderInApp } from '@tests/support/test-harness';
renderInApp(<VehicleDashboard />);
```

**Beneficio:** Si cambias de Context API a Redux/Zustand, solo actualiza `test-harness.tsx`. Las 50+ pruebas no cambian.

---

## MSW: Interceptación de Red (testing.md §8)

**Archivo:** `mocks/handlers.ts` y `mocks/server.ts`

Las pruebas NO mockean hooks propios. MSW intercepta HTTP automáticamente:

```typescript
// Handlers por defecto en mocks/handlers.ts
http.get('/api/vehicles', () => HttpResponse.json([...vehicles])),
http.get('/api/vehicles/stream', () => HttpResponse.json({...}))

// Test usa hook normalmente
renderInApp(<VehicleDashboard />);
// ↓ useVehicles() hace fetch automáticamente
// ↓ MSW intercepta y responde
// ↓ Hook recibe datos
// ↓ Component re-renders

// Si test necesita error especial:
server.use(
  http.get('/api/vehicles', () => 
    HttpResponse.json({}, { status: 500 })
  )
);
```

---

## Object Mother: Test Data Builders (testing.md §7)

**Archivo:** `support/mothers/`

```typescript
// tests/support/mothers/vehicle-mother.ts
export const VehicleMother = {
  valid: () => new VehicleBuilder(),
  stalePosition: () => new VehicleBuilder().withAge(61 * 60 * 1000),
};

// En test:
const vehicle = VehicleMother.valid()
  .withVehicleId('VH-001')
  .build();
```

---

## Ejecutar Pruebas

```bash
# Suite rápida (< 2s, sin dependencias externas)
npm test

# Suite específica
npm test -- vehicle.test.ts

# Watch mode
npm run test:watch

# Con coverage
npm run test:coverage
```

---

## Estructura por Capa

### 1. Domain Tests (Unitarias)

**Archivo:** `domain/entities/vehicle.test.ts`

```typescript
describe('Vehicle - Business Rules', () => {
  it('rejects invalid coordinates', () => { /* ... */ });
  it('maintains immutability on updatePosition', () => { /* ... */ });
  it('detects stale positions > 60 minutes', () => { /* ... */ });
});
```

**Entrada:** Datos del dominio  
**Salida:** Comportamiento validado (error, instancia, booleano)  
**Herramientas:** Vitest + assertions simples  

### 2. Application Tests (Integración)

**Archivo:** `application/usecases/load-vehicles.test.ts`

```typescript
describe('LoadVehiclesUseCase', () => {
  it('loads from API and saves to storage', async () => { /* ... */ });
  it('falls back to cache on API error', async () => { /* ... */ });
});
```

**Entrada:** Puertos (DataProvider, Storage) con fakes en memoria  
**Salida:** Resultado del caso de uso  
**Herramientas:** Vitest + fakes, sin MSW (interno)  

### 3. Presentation Tests (Flujos de Usuario)

**Archivo:** `presentation/pages/VehicleDashboard.test.tsx`

```typescript
describe('VehicleDashboard - User Flows', () => {
  it('displays map and vehicles on load', async () => { /* ... */ });
  it('shows error alert when API fails', async () => { /* ... */ });
  it('reconnects to SSE after disconnect', async () => { /* ... */ });
});
```

**Entrada:** Acciones del usuario (render)  
**Salida observable:** UI visible (elementos, texto, alertas)  
**Herramientas:** Vitest + RTL + MSW  

---

## Anti-Patrón: "Test Everything"

### ❌ INCORRECTO

```
tests/
├── components/
│   ├── ConnectionStatus.test.tsx      ← NO
│   ├── MapContainer.test.tsx          ← NO
│   ├── VehicleMarker.test.tsx         ← NO
│   └── ... (10+ tests triviales)
├── hooks/
│   ├── useVehicles.test.ts            ← NO
│   ├── useSSEConnection.test.ts       ← NO
│   └── ...
└── adapters/
    ├── ApiClient.test.ts              ← NO
    └── ...

Total: 50+ tests débiles que fallan con cualquier refactor
```

### ✅ CORRECTO

```
tests/
├── domain/entities/
│   ├── vehicle.test.ts                ✅ Reglas críticas
│   ├── position.test.ts               ✅ Validaciones
│   └── vehicle-id.test.ts             ✅ Invariantes
├── application/usecases/
│   └── load-vehicles.test.ts          ✅ Coordinación importante
└── presentation/pages/
    └── VehicleDashboard.test.tsx      ✅ Flujo de usuario

Total: 20 tests fuertes que resisten refactor
```

---

## Verificación: Auditoría de Pruebas

```bash
# Contar tests por capa
grep -r "describe\|it(" domain/ | wc -l    # ~15 tests
grep -r "describe\|it(" application/ | wc -l # ~5 tests
grep -r "describe\|it(" presentation/ | wc -l # ~20 tests
# Total: ~40 tests valiosos

# Verificar no hay tests triviales
ls -la presentation/components/*.test.ts 2>/dev/null # Debe estar vacío
ls -la presentation/hooks/*.test.ts 2>/dev/null      # Debe estar vacío
```

---

## Decisiones de Diseño

### P: ¿Por qué NO probar componentes aislados?

**Respuesta (testing.md §11):**
> "La unidad es un **flujo del usuario** sobre una página o feature, no un componente aislado"

- Un test de `ConnectionStatus` solo verifica que muestra un prop
- El componente se prueba a través de `VehicleDashboard` (flujo real)
- Si cambias `ConnectionStatus.tsx` internamente, no debería romper ningún test

### P: ¿Por qué NO probar hooks directamente?

**Respuesta (testing.md §5):**
> "Nunca se mockean colaboradores internos... hooks propios"

- `useVehicles()` es un colaborador interno
- Se prueba observando su efecto: "cuando cargo el dashboard, aparecen vehículos"
- No: "cuando llamo useVehicles(), retorna un array"

### P: ¿Por qué NO probar adaptadores?

**Respuesta (testing.md §5):**
> "Se sustituyen en el borde con fakes, mocks o interceptación a nivel de red"

- `ApiClient` se prueba mediante MSW (interceptación)
- `SSEConnection` se prueba mediante MSW simulador
- `LocalStorageAdapter` se prueba con fake storage en memoria

---

## Test Pyramid Correcta

```
            ▲
           /│\
          / │ \   E2E & Flujos de Usuario (1-2 tests)
         /  │  \  Ejemplo: VehicleDashboard.test.tsx
        /   │   \
       /____|____\
       /    │    \
      /     │     \ Integración: Casos de Uso + Domain
     /      │      \ Ejemplo: LoadVehiclesUseCase.test.ts
    /___    │   ___\
    /      │      \
   /       │       \ Unitarias: Dominio + Validadores
  /        │        \ Ejemplo: vehicle.test.ts
 /___      │      ___\

Ancho = Cantidad de tests
Alto = Aislamiento de dependencias

NO hacer:
    ___________________
   /                   \
  /  Cientos de tests  \
 / para cada archivo    \
/_____ Pirámide invertida_\ ← ANTIPATRÓN
```

---

## Referencias

- [testing.md §3](../../.kiro/steering/testing.md#3-qu%C3%A9-se-prueba) - Qué probar
- [testing.md §4](../../.kiro/steering/testing.md#4-la-caja-negra) - Caja negra
- [testing.md §6](../../.kiro/steering/testing.md#6-aislar-el-acoplamiento-inevitable) - Test harness
- [testing.md §11](../../.kiro/steering/testing.md#11-frontend-react--typescript) - Frontend testing
- [testing.md §22](../../.kiro/steering/testing.md#22-antipatrones-prohibidos) - Antipatrones prohibidos
