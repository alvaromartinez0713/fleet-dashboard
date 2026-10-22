# Requisitos: Dashboard de Mapa de Vehículos

## Introducción

El Dashboard de Mapa de Vehículos es un componente central del Portal Corporativo que visualiza la flota de vehículos en un mapa interactivo, mostrando sus últimas posiciones conocidas y permitiendo actualizaciones en tiempo real mediante Server-Sent Events (SSE).

El dashboard demuestra la arquitectura reactiva del sistema, aplicando Clean Architecture en el frontend y proporcionando una base sólida para futuras expansiones como alertas visuales, filtrado de vehículos y análisis de movimientos.

Esta especificación define los requisitos funcionales y no funcionales para la versión inicial del MVP, incluida la integración con datos mock durante la fase de desarrollo y la eventual conexión con las APIs del backend.

## Glosario

- **Dashboard**: Componente central de la interfaz que agrupa el mapa y los controles de visualización
- **Mapa**: Representación geográfica interactiva donde se ubican y visualizan los vehículos
- **Vehículo**: Entidad del dominio que representa un recurso móvil de la flota, identificado por un `vehicleId` único
- **Posición**: Objeto de valor que contiene latitud, longitud y marca temporal de la ubicación de un vehículo
- **SSE (Server-Sent Events)**: Protocolo de comunicación unidireccional del servidor al cliente para entrega de eventos en tiempo real
- **MockDataProvider**: Adaptador que proporciona datos ficticios de vehículos para demostración inicial
- **VehicleStateManager**: Coordinador de aplicación que gestiona el estado de vehículos y sus actualizaciones
- **Reconexión**: Proceso automático de restablecimiento de la conexión SSE tras una desconexión
- **Desconexión**: Pérdida de conectividad con el servidor SSE que interrumpe la entrega de actualizaciones
- **Marcador de vehículo**: Elemento visual en el mapa que representa la posición de un vehículo individual
- **Estado de conexión**: Indicador del estado actual de la conexión SSE (conectado, desconectado, en reconexión, error)

## Requisitos

### Requisito 1: Visualizar mapa interactivo con vehículos

**User Story:** Como usuario del portal, quiero ver un mapa interactivo con los vehículos de mi flota, para conocer sus ubicaciones geográficas actuales en un vistazo.

#### Criterios de aceptación

1. WHEN el dashboard se carga, THEN el MapContainer debe renderizar un elemento de mapa centrado en una ubicación por defecto
2. WHEN el dashboard se inicializa, THEN el VehicleRenderer debe mostrar marcadores de vehículos para todos los vehículos disponibles
3. WHEN un marcador de vehículo es visible en el mapa, THEN el Marker debe mostrar el identificador del vehículo como una etiqueta
4. WHEN el usuario interactúa con el mapa (desplazar, zoom), THEN el MapContainer debe permitir operaciones estándar de mapa sin restricciones
5. WHERE la librería de mapa lo soporta, THEN el MapContainer debe permitir cambiar entre diferentes estilos de mapa (satélite, calle, híbrido)
6. WHEN el dashboard se carga sin conectividad, THEN el MapContainer debe seguir renderizándose y ser usable con datos de vehículos previamente cacheados

### Requisito 2: Mostrar últimas posiciones conocidas de vehículos

**User Story:** Como operador de flota, quiero ver la última posición conocida de cada vehículo, para saber dónde está estacionado o circulando actualmente.

#### Criterios de aceptación

1. THE el VehicleDataProvider debe devolver objetos de vehículos con campos `vehicleId`, `licensePlate`, `latitude`, `longitude` y `timestamp`
2. WHEN una posición de vehículo se carga, THEN el PositionRenderer debe mostrarla con una precisión de al menos 4 decimales para latitud/longitud
3. WHEN se muestran múltiples vehículos, THEN los VehicleMarkers deben mostrar identificadores distintos para evitar confusión
4. IF la marca de tiempo de un vehículo tiene más de 60 minutos, THEN el Marker debe mostrar un indicador visual (color desvanecido o distintivo "antiguo")
5. WHEN el usuario se pasa sobre un marcador de vehículo, THEN el Tooltip debe mostrar: identificador del vehículo, placa, latitud, longitud y marca de tiempo de la última posición
6. WHERE se puede verificar la validez de datos de geolocalización, THEN el PositionValidator debe rechazar posiciones con latitud fuera de [-90, 90] o longitud fuera de [-180, 180]

### Requisito 3: Actualizar posiciones en tiempo real mediante SSE

**User Story:** Como usuario del dashboard, quiero que las posiciones de los vehículos se actualicen automáticamente, para ver los movimientos en tiempo real sin recargar la página.

#### Criterios de aceptación

1. WHEN el dashboard se carga, THEN la SSEConnection debe establecer una conexión con el punto de acceso SSE del backend
2. WHEN llega un evento SSE con una actualización de posición, THEN el EventProcessor debe analizar los datos del evento y actualizar la posición del vehículo
3. WHEN la posición de un vehículo se actualiza vía SSE, THEN el MapContainer debe re-renderizar el marcador del vehículo afectado sin recargar el mapa completo
4. WHEN las actualizaciones de posición llegan con alta frecuencia (>10 actualizaciones/segundo), THEN el EventProcessor debe agruparlas apropiadamente para evitar re-renderizaciones excesivas
5. WHEN una conexión SSE se cierra inesperadamente, THEN el ReconnectionManager debe intentar automáticamente reconectarse después de un retraso con retroceso exponencial (1s, 2s, 4s, máx 30s)
6. IF la reconexión tiene éxito, THEN la SSEConnection debe reanudar la recepción de actualizaciones desde el punto de reconexión
7. WHEN el usuario se aleja del dashboard, THEN la SSEConnection debe terminarse y los recursos deben liberarse
8. IF el usuario regresa al dashboard, THEN la SSEConnection debe restablecerse

### Requisito 4: Usar datos mock para demostración inicial

**User Story:** Como desarrollador de features, quiero usar datos ficticios para demostrar el dashboard, para poder trabajar sin depender de la API backend.

#### Criterios de aceptación

1. THE el MockDataProvider debe generar un conjunto determinista de 20 vehículos con coordenadas válidas en diferentes zonas geográficas
2. WHEN el dashboard se carga en modo mock, THEN el VehicleStateManager debe cargar vehículos desde MockDataProvider en lugar de la API del backend
3. THE el MockDataProvider debe generar datos de vehículos con: vehicleId único, formato de placa válido, pares latitud/longitud realistas y marca de tiempo actual
4. WHEN el modo mock está activo, THEN el SSESimulator debe emitir eventos de actualización de posición que simulen movimiento de vehículos realista (cambios de coordenadas incrementales)
5. WHEN la aplicación está configurada para datos mock, THEN debe haber un indicador visual (pancarta, etiqueta) que muestre "Modo Demo" para distinguir de datos de producción
6. WHERE la API del backend está disponible, THEN el DataProviderSelector debe permitir cambiar de datos mock a datos reales sin cambios de código (configuración de variable de entorno)

### Requisito 5: Mantener separación arquitectónica (dominio, aplicación, infraestructura)

**User Story:** Como arquitecto del sistema, quiero que la lógica del dashboard siga Clean Architecture, para facilitar pruebas, mantenimiento y futuras expansiones.

#### Criterios de aceptación

1. THE el directorio VehicleMapDomain debe contener objetos de dominio: Vehicle (entidad), Position (objeto de valor), VehicleId (objeto de valor)
2. THE los objetos de dominio NO deben depender de React, clientes HTTP o librerías SSE
3. THE el directorio VehicleApplicationLayer debe contener casos de uso: LoadVehicles, SubscribeToPositionUpdates, HandlePositionUpdate
4. THE la capa de aplicación NO debe depender de componentes de UI o frameworks, solo de dominio y puertos
5. THE el directorio InfrastructureLayer debe contener adaptadores: SSEConnection, MockDataProvider, MapAdapter, EventParser
6. THE la capa de Presentation (componentes React) debe usar solo interfaces de la capa de aplicación, nunca acceder directamente a adaptadores de infraestructura
7. THE la interfaz DataProviderPort debe definir el contrato: `loadVehicles(): Promise<Vehicle[]>`, `subscribeToUpdates(callback): Subscription`
8. THE la interfaz VehicleRealTimePort debe definir contrato de suscripción SSE: `connect(): Promise<void>`, `onUpdate(callback): void`, `disconnect(): void`

### Requisito 6: Implementar hooks reutilizables para SSE y estado

**User Story:** Como desarrollador, quiero hooks React reutilizables para gestionar SSE y estado de vehículos, para facilitar la implementación consistente en diferentes componentes.

#### Criterios de aceptación

1. THE el hook `useVehicles()` debe devolver lista de vehículos actual y estados de carga/error
2. THE el hook `useSSEConnection()` debe gestionar el ciclo de vida de SSE: conexión, desconexión y limpieza
3. WHEN se llama a `useSSEConnection`, THEN el hook debe desencadenar la conexión SSE en montaje y limpieza en desmontaje
4. WHEN SSE recibe un evento de actualización, THEN el hook `useVehicles` debe actualizar su estado automáticamente
5. WHEN `useVehicles` se usa en múltiples componentes, THEN cada componente debe tener su propia instancia de estado independiente (sin estado mutable compartido)
6. THE el hook `useVehicleUpdates()` debe exponer métodos: `subscribeToVehicle(vehicleId, callback)`, `unsubscribeFromVehicle(vehicleId)` para suscripciones de grano fino
7. WHERE un componente se desmonta, THEN todas las suscripciones y conexiones SSE iniciadas por sus hooks deben terminarse y los recursos liberarse
8. WHEN la conexión SSE falla, THEN el hook NO debe reintentar indefinidamente; debe respetar la configuración de retroceso exponencial

### Requisito 7: Gestionar errores de conexión y reconexión

**User Story:** Como usuario, quiero que el dashboard maneje automáticamente desconexiones y errores, para mantener una experiencia confiable incluso con conectividad inestable.

#### Criterios de aceptación

1. WHEN la conexión SSE se pierde, THEN el ReconnectionManager debe mostrar una notificación "Reconectando..."
2. WHEN la reconexión tiene éxito, THEN la notificación debe descartarse y las actualizaciones deben reanudarse
3. WHEN la reconexión falla después de 5 intentos consecutivos (máx 2.5 minutos), THEN el ErrorHandler debe mostrar un mensaje de error accionable al usuario
4. WHEN el usuario hace clic en un botón "Reintentar", THEN el ReconnectionManager debe reiniciar el temporizador de retroceso e intentar reconexión inmediatamente
5. WHEN un evento SSE contiene datos inválidos, THEN el EventValidator debe registrar el error y continuar procesando eventos válidos sin fallar
6. IF una actualización de vehículo contiene una coordenada fuera de rango, THEN el PositionValidator debe rechazarla, registrar una advertencia y preservar la posición válida anterior
7. WHEN el mapa está desconectado, THEN el Dashboard debe mostrar vehículos cacheados con una marca de tiempo "Última actualización a HH:MM:SS"
8. WHERE el usuario cierra la pestaña del navegador o se aleja, THEN la SSEConnection debe terminarse adecuadamente para liberar recursos del servidor

### Requisito 8: Integración con OpenAPI y contrato de API

**User Story:** Como integrador, quiero que el dashboard esté alineado con el contrato OpenAPI del backend, para garantizar compatibilidad actual y futura.

#### Criterios de aceptación

1. THE el formato de respuesta VehicleDTO debe coincidir con el esquema OpenAPI: `vehicleId` (string), `licensePlate` (string), `latitude` (número), `longitude` (número), `timestamp` (string ISO-8601)
2. WHEN el backend expone un punto de acceso `GET /api/vehicles`, THEN el DataProviderSelector debe cambiar para usarlo en lugar de datos mock
3. WHEN el backend expone un punto de acceso SSE `/api/vehicles/positions/stream`, THEN la SSEConnection debe conectarse y analizar eventos de acuerdo con el contrato
4. THE el analizador de eventos SSE debe manejar eventos con estructura: `{ "type": "position_update", "data": { "vehicleId": "...", "latitude": ..., "longitude": ..., "timestamp": "..." } }`
5. WHERE el contrato OpenAPI incluye respuestas de error (400, 401, 403, 404, 500), THEN el ErrorHandler debe mapearlas a mensajes apropiados para el usuario
6. WHEN las respuestas de API violan el esquema OpenAPI, THEN el ResponseValidator debe rechazarlas y registrar un error de violación de contrato

## Criterios de aceptación globales del dashboard

### Funcionales

- El mapa se carga y es interactivo en primera carga
- Todos los vehículos mock se renderizan como marcadores inicialmente
- Las actualizaciones SSE se reflejan en el mapa sin recargar
- La desconexión SSE dispara reconexión automática con backoff exponencial
- Los errores de conexión se manejan de forma elegante con mensajes claros al usuario
- Los datos mock incluyen 20 vehículos con coordenadas realistas

### De calidad arquitectónica

- La lógica de dominio está completamente separada de React y del navegador
- Los hooks son reutilizables y no comparten estado mutable entre componentes
- Las transiciones de estado de conexión son predecibles y probables
- Los errores de API no rompen la aplicación; se informan y se recuperan

### De pruebas

Las pruebas siguen el estándar definido en `testing.md` del steering:
- Suite rápida: flujos de usuario principales con interceptación de red (MSW)
- Escenarios obligatorios: carga, datos, sin datos, error, desconexión/reconexión SSE
- Pruebas de dominio: entidades (Vehicle, Position, VehicleId) y validadores
- Property-based: idempotencia de actualización SSE, integridad de coordenadas
- Datos de prueba: Object Mother + Test Data Builder
- Interceptación de red: MSW para HTTP y SSE, sin mockear módulos
- Trazabilidad: cada criterio de aceptación referenciado en las pruebas

---

## Pruebas mínimas del frontend

Según `testing.md`, el frontend se prueba por **flujos de usuario** como caja negra, no por componentes aislados. Las siguientes pruebas son obligatorias:

### Prueba 1: Dashboard carga y muestra vehículos

**Verifies:** Requisito 1.2, Requisito 4.2

```gherkin
GIVEN el dashboard en modo mock
WHEN la página se carga
THEN se renderiza un mapa
AND se muestran marcadores de todos los 20 vehículos mock
AND cada marcador muestra el vehicleId visible
```

**Entrada:** Acción del usuario (cargar página del dashboard)
**Salida observable:** Mapa renderizado + 20 marcadores visibles en pantalla
**Suite:** Rápida (< 2s)
**Herramientas:** Vitest + RTL + MSW

---

### Prueba 2: SSE conecta y actualiza posiciones en tiempo real

**Verifies:** Requisito 3.1, Requisito 3.2, Requisito 3.3

```gherkin
GIVEN el dashboard cargado con 20 vehículos
WHEN se emite un evento SSE con actualización de posición para VH-001
THEN el marcador de VH-001 se re-renderiza en la nueva posición
AND el mapa completo NO se recarga
AND otros 19 marcadores permanecen en su lugar
```

**Entrada:** Evento SSE simulado vía MSW
**Salida observable:** Posición del marcador actualizada en pantalla
**Suite:** Rápida (< 2s)
**Herramientas:** Vitest + RTL + MSW (servidor SSE simulado)
**Propiedad:** Idempotencia - procesar el mismo evento 2 veces = mismo resultado

---

### Prueba 3: SSE desconecta y reconecta con backoff exponencial

**Verifies:** Requisito 3.5, Requisito 7.1, Requisito 7.2

```gherkin
GIVEN el dashboard con SSE conectada
WHEN la conexión SSE se cierra inesperadamente
THEN aparece notificación "Reconectando..."
AND se intenta reconectar con backoff: 1s, 2s, 4s
AND después de éxito, la notificación desaparece
AND SSE reanuda entrega de actualizaciones
```

**Entrada:** Simular cierre de EventSource via MSW
**Salida observable:** Notificación en pantalla + reconexión automática
**Suite:** Rápida (< 2s con timers mock)
**Herramientas:** Vitest + RTL + MSW + fake timers
**Propiedad:** Idempotencia - reconectar 2 veces deja mismo estado

---

### Prueba 4: Error tras máximo de reconexiones

**Verifies:** Requisito 7.3

```gherkin
GIVEN el dashboard intentando conectar SSE
WHEN la reconexión falla 5 veces consecutivas
THEN se muestra mensaje de error: "No se pudo conectar. Intenta más tarde"
AND aparece botón "Reintentar"
AND el mapa sigue mostrando vehículos cacheados
```

**Entrada:** Simular fallo persistente de SSE (5+ intentos fallidos)
**Salida observable:** Mensaje de error + botón reintentar + datos cacheados visibles
**Suite:** Rápida (< 2s con timers mock)
**Herramientas:** Vitest + RTL + MSW

---

### Prueba 5: Evento SSE inválido se rechaza sin romper app

**Verifies:** Requisito 7.5

```gherkin
GIVEN el dashboard con SSE conectada
WHEN se emite evento SSE malformado (JSON inválido)
THEN se registra error en console
AND la app sigue funcionando
AND evento válido siguiente se procesa correctamente
```

**Entrada:** Evento SSE con JSON corrupto
**Salida observable:** App no se bloquea, evento siguiente funciona
**Suite:** Rápida (< 2s)
**Herramientas:** Vitest + RTL + MSW
**Verificación:** Spy en console.error, verificar app sigue usable

---

### Prueba 6: Coordenadas inválidas se rechazan

**Verifies:** Requisito 2.6, Requisito 7.6

```gherkin
GIVEN el dashboard con SSE conectada
WHEN se emite evento con coordenadas fuera de rango (lat=91, lon=181)
THEN el evento se rechaza
AND marcador anterior se mantiene en su posición última válida
AND no aparece marcador fuera del mapa
```

**Entrada:** Evento SSE con coordenadas inválidas (latitud > 90)
**Salida observable:** Marcador NO se actualiza + posición anterior se conserva
**Suite:** Rápida (< 2s)
**Herramientas:** Vitest + RTL + MSW
**Propiedad:** El PositionValidator rechaza todas las coordenadas inválidas

---

### Prueba 7: Posición antigua (> 60 min) muestra indicador visual

**Verifies:** Requisito 2.4

```gherkin
GIVEN un vehículo con posición de hace 61 minutos
WHEN el dashboard se carga
THEN el marcador del vehículo tiene clase CSS 'stale'
AND color desvanecido visualmente
AND tooltip muestra: "Última actualización: hace 61 minutos"
```

**Entrada:** Vehicle con timestamp hace 61 minutos (tiempo mock determinista)
**Salida observable:** Marcador con clase 'stale', color visualmente diferente
**Suite:** Rápida (< 2s)
**Herramientas:** Vitest + RTL

---

### Prueba 8: Modo Demo muestra banner solo cuando está activo

**Verifies:** Requisito 4.5

```gherkin
GIVEN VITE_DATA_MODE=mock
WHEN el dashboard se carga
THEN aparece banner: "🎭 MODO DEMOSTRACIÓN"

GIVEN VITE_DATA_MODE=api
WHEN el dashboard se carga
THEN NO aparece banner demo
```

**Entrada:** Variable de entorno VITE_DATA_MODE
**Salida observable:** Banner visible/no visible según configuración
**Suite:** Rápida (< 1s)
**Herramientas:** Vitest + RTL

---

### Prueba 9: Mock genera 20 vehículos válidos y deterministas

**Verifies:** Requisito 4.1, Requisito 4.3

```gherkin
GIVEN MockDataProvider
WHEN se llama loadVehicles()
THEN devuelve exactamente 20 vehículos
AND cada vehiculo tiene: vehicleId único, licensePlate, lat en [-90,90], lon en [-180,180], timestamp
AND llamadas sucesivas devuelven los mismos IDs y posiciones (determinista)
```

**Entrada:** Llamada a MockDataProvider.loadVehicles()
**Salida observable:** 20 vehículos válidos, idénticos entre llamadas
**Suite:** Rápida (< 1s)
**Herramientas:** Vitest
**Propiedad:** Round-trip - serializar y deserializar datos mock preserva integridad

---

### Prueba 10: Cambiar de proveedor (mock → api) sin código

**Verifies:** Requisito 4.6

```gherkin
GIVEN VITE_DATA_MODE=mock
WHEN se cargan vehículos
THEN se usan datos de MockDataProvider

GIVEN VITE_DATA_MODE=api
WHEN se cargan vehículos
THEN se llama GET /api/vehicles
AND se usan datos de ApiDataProvider
```

**Entrada:** Variable de entorno VITE_DATA_MODE + MSW handler para /api/vehicles
**Salida observable:** Provider correcto se selecciona automáticamente
**Suite:** Rápida (< 2s)
**Herramientas:** Vitest + RTL + MSW
**Verificación:** Spy en llamadas HTTP

---

## Matriz de trazabilidad: Requisitos → Pruebas

| Requisito | Criterio | Prueba mínima | Suite | Ruta |
|-----------|----------|---------------|-------|------|
| 1.2 | Mostrar marcadores | Prueba 1 | Rápida | `dashboard.integration.test.ts` |
| 1.3 | ID visible | Prueba 1 | Rápida | `dashboard.integration.test.ts` |
| 2.4 | Posición antigua | Prueba 7 | Rápida | `vehicle-marker.test.ts` |
| 2.6 | Rechazo coords inválidas | Prueba 6 | Rápida | `position-validator.test.ts` |
| 3.1 | SSE conecta | Prueba 2 | Rápida | `dashboard.integration.test.ts` |
| 3.2 | SSE actualiza posición | Prueba 2 | Rápida | `dashboard.integration.test.ts` |
| 3.3 | Renderizado sin recarga | Prueba 2 | Rápida | `dashboard.integration.test.ts` |
| 3.5 | Backoff exponencial | Prueba 3 | Rápida | `sse-connection.test.ts` |
| 4.1 | 20 vehículos mock | Prueba 9 | Rápida | `mock-data-provider.test.ts` |
| 4.2 | Cargar mock en dashboard | Prueba 1 | Rápida | `dashboard.integration.test.ts` |
| 4.3 | Datos válidos mock | Prueba 9 | Rápida | `mock-data-provider.test.ts` |
| 4.5 | Banner Modo Demo | Prueba 8 | Rápida | `demo-mode-banner.test.ts` |
| 4.6 | Cambiar proveedor | Prueba 10 | Rápida | `data-provider-factory.test.ts` |
| 7.1 | Notificación reconectando | Prueba 3 | Rápida | `dashboard.integration.test.ts` |
| 7.2 | Reanuda actualizaciones | Prueba 3 | Rápida | `dashboard.integration.test.ts` |
| 7.3 | Error tras 5 intentos | Prueba 4 | Rápida | `sse-connection.test.ts` |
| 7.5 | Evento inválido ignorado | Prueba 5 | Rápida | `dashboard.integration.test.ts` |
| 7.6 | Posición anterior preservada | Prueba 6 | Rápida | `vehicle-state-manager.test.ts` |
| 8.1 | DTO con campos correctos | (Contrato OpenAPI valida) | - | Integración futura |

---

**Notas de implementación:**

- Las pruebas siguen el estándar `testing.md`: caja negra, flujos de usuario, sin mocks de módulos, interceptación MSW
- Todas las pruebas son **deterministas** (sin Date.now(), con timers mock, seeds fijas)
- Cada prueba tiene **un solo Act**
- Los datos de prueba se construyen con **Object Mother + builders** (VehicleMother, PositionMother)
- Trazabilidad explícita: cada prueba referencia los criterios que verifica
- Tiempo de ejecución: Suite rápida completa < 10 segundos
- Resistencia al refactoring: No verifican detalles de implementación, solo comportamiento observable

