---
inclusion: always
---

# Estándar de pruebas

## 1. Propósito

Definir cómo se diseñan las pruebas del backend, del frontend web, de la aplicación móvil y las pruebas de carga.

El objetivo es tener pocas pruebas, valiosas y difíciles de romper, que verifiquen el comportamiento importante del sistema como caja negra. No se busca cubrir capa por capa ni clase por clase.

## 2. Principios

Cada prueba se evalúa con cuatro pilares:

| Pilar | Pregunta |
|---|---|
| Protección contra regresiones | ¿Detectaría un bug real en una regla importante? |
| Resistencia al refactoring | ¿Sigue pasando si cambio la implementación sin cambiar el comportamiento? |
| Feedback rápido | ¿Se ejecuta en segundos? |
| Mantenibilidad | ¿Se entiende y se prepara con poco código? |

Reglas:

- **La resistencia al refactoring no se negocia.** Una prueba que falla sin que haya cambiado el comportamiento (falso positivo) es un defecto de la prueba.
- Los otros tres pilares se equilibran según el caso.
- La unidad de prueba es una **unidad de comportamiento**, no una clase, función o componente.
- Se verifica lo **observable**: la respuesta, el estado final o el efecto en un sistema externo. No se verifica cómo se llegó a ese resultado.
- La cobertura de código no es un objetivo. Es un indicador para encontrar zonas sin probar.

## 3. Qué se prueba

| Tipo de código | Decisión |
|---|---|
| Reglas de negocio y algoritmos (mucha lógica, pocas dependencias) | Probar a fondo |
| Coordinadores: controladores, adaptadores, wiring (poca lógica, muchas dependencias) | Pocas pruebas de integración por los flujos principales |
| Código trivial: DTOs, getters, mapeos directos, configuración | No probar de forma dedicada |
| Código con mucha lógica y muchas dependencias | Refactorizar primero: extraer la lógica al dominio y luego probarla |

## 4. La caja negra: entrar y salir por los bordes

Cada prueba entra por un **punto de entrada público** y observa un **punto de salida público**. Lo que hay en medio es implementación y la prueba no lo conoce.

| Aplicación | Entrada | Salida observable |
|---|---|---|
| Backend: núcleo | Caso de uso (puerto de entrada) | Valor devuelto y estado en los puertos de salida |
| Backend: servicio | HTTP request o mensaje de Kafka | HTTP response, filas en TimescaleDB, mensajes publicados |
| Frontend | Acción del usuario (click, texto, navegación) o evento SSE | Lo que el usuario ve en pantalla |
| Móvil | Acción del usuario, ubicación capturada, cambio de conectividad | Pantalla, registros pendientes, requests enviados a la API |
| Carga | Tráfico HTTP simulado | Respuestas, latencias y datos finales persistidos |

## 5. Dobles de prueba

Se distinguen dos tipos de dependencias externas al proceso:

- **Gestionadas**: solo las usa nuestra aplicación, como TimescaleDB o el SQLite del móvil. Se usa **el mismo motor en una instancia efímera y aislada** (ver 5.1) y se verifica el estado final, no las llamadas.
- **No gestionadas**: las observan terceros o no las controlamos, como Kafka hacia otros consumidores, el LLM, la API del backend vista desde el móvil o el frontend, el GPS o el reloj. **Se sustituyen en el borde** con fakes, mocks o interceptación a nivel de red (sección 8).

Reglas:

- Nunca se mockean colaboradores internos (entidades, servicios de dominio, otros casos de uso, hooks propios).
- Preferir **fakes** con estado (repositorio en memoria, API falsa) a mocks que verifican llamadas.
- Usar mocks que verifican interacción solo cuando esa interacción **es** el comportamiento. Ejemplo: "se publicó exactamente un evento en Kafka".
- El reloj, los generadores de IDs y la aleatoriedad se inyectan para que las pruebas sean deterministas.

### 5.1 Qué significa "real" en una prueba

"Real" significa **el mismo motor, levantado solo para la prueba y destruido al terminar**. Nunca significa producción ni un servidor compartido.

- **TimescaleDB y Kafka:** contenedores efímeros con Testcontainers, en la máquina local o en el runner de CI. Cada ejecución arranca vacía, aplica las migraciones y no contiene datos reales.
- **SQLite (móvil):** base en memoria (`:memory:`) o archivo temporal, eliminado al terminar.
- **Producción:** las pruebas automatizadas nunca acceden a ella, ni a sus credenciales ni a sus datos.

Motivo: un fake de la base de datos no detecta SQL incorrecto, errores de tipos o zonas horarias, `ON CONFLICT` y unique constraints (la base de la idempotencia), migraciones rotas ni hypertables mal definidas.

### 5.2 Suites y entornos

| Suite | Contenido | Requiere | Dónde se ejecuta |
|---|---|---|---|
| Rápida | Núcleo: dominio y casos de uso con fakes, property-based (sección 13), flujos de UI con interceptación de red | Nada externo | Local en cada cambio y en CI |
| Integración | Adaptadores, worker, contrato del proveedor (sección 9) y fallos con Toxiproxy (sección 14) contra contenedores efímeros | Docker | CI y local bajo demanda |
| Mutación | Mutation testing del núcleo (sección 17) | Nada externo | CI programado o bajo demanda |
| E2E y regresión | Recorridos completos | Entorno levantado con Docker Compose o un servidor de pruebas dedicado | CI o entorno de pruebas |
| Carga y caos | Scripts de k6 | Entorno de pruebas dedicado | Bajo demanda, nunca contra producción |

- En Go, las pruebas de integración se separan con build tags: `go test ./...` ejecuta la suite rápida y `go test -tags=integration ./...` la de integración.
- En frontend y móvil, la separación se hace con scripts del gestor de paquetes (por ejemplo `test` y `test:integration`), definidos en la Spec correspondiente.

## 6. Aislar el acoplamiento inevitable

Algo de acoplamiento a frameworks es inevitable. Se concentra en una sola capa de **test harness**, así un cambio de framework solo toca el harness y no las pruebas.

- Cada aplicación tiene helpers de prueba (`tests/support/` o equivalente) que exponen operaciones del dominio: `sendBatch(...)`, `givenVehicleStopped(...)`, `openDashboard()`, `goOffline()`.
- Las pruebas usan esos helpers y no llaman directamente a Gin, Kafka, React Testing Library ni a APIs nativas.
- Los selectores de UI se basan en roles y textos accesibles, no en clases CSS ni en la estructura del DOM.
- Los datos de prueba se construyen según la sección 7.

## 7. Datos de prueba: Object Mother + Test Data Builder

Los datos de prueba se crean combinando dos patrones:

- **Object Mother**: punto único que ofrece escenarios con nombre del dominio (`valid()`, `stoppedInCriticalZone()`, `outOfRange()`).
- **Test Data Builder**: el Mother devuelve un builder con valores por defecto válidos. Cada prueba sobrescribe solo los datos relevantes para su comportamiento.

Reglas:

- Los valores por defecto son **válidos y deterministas**. Nada de aleatoriedad sin seed ni de hora real.
- Cada llamada crea una **instancia nueva**. No se comparten objetos mutables entre pruebas.
- El builder construye a través del **constructor o factory del dominio**, para respetar las invariantes. Si cambia la estructura interna, solo se modifica el builder.
- El Mother solo contiene escenarios reutilizados en varias pruebas. Las variantes puntuales se expresan con el builder dentro de la prueba, para evitar una explosión de métodos.
- Los mismos Mothers alimentan los fakes, los handlers de interceptación de red (sección 8) y los datos semilla de las pruebas de integración.
- Ubicación: `tests/support/mothers/` o equivalente en cada aplicación.

Ejemplo (TypeScript):

```typescript
// tests/support/mothers/telemetryEventMother.ts
export const TelemetryEventMother = {
  valid: () => new TelemetryEventBuilder(),
  outOfRange: () => new TelemetryEventBuilder().withLatitude(120),
};

// in a test: only the relevant data is visible
const event = TelemetryEventMother.valid().withVehicleId('VH-001').build();
```

En Go se aplica el mismo patrón con functional options:

```go
// tests/support/mothers/telemetry_event.go
func NewTelemetryEvent(opts ...TelemetryEventOption) domain.TelemetryEvent
func WithVehicleID(id string) TelemetryEventOption
```

## 8. Interceptación a nivel de red (frontend y móvil)

El frontend y la app móvil se prueban **interceptando las requests a nivel de red**: la aplicación hace requests reales y la herramienta de prueba las captura y responde. La aplicación no sabe que está en una prueba.

- Herramienta de referencia: **MSW (Mock Service Worker)**.
  - Web en navegador: Service Worker.
  - Pruebas de frontend con Vitest/Jest: `msw/node`.
  - React Native: paquete oficial `@msw/react-native`.
- E2E web: interceptación con Playwright (`page.route()`) o un mock server. E2E móvil: la app apunta a un mock server (MSW o WireMock) mediante configuración.
- El soporte de SSE en la herramienta elegida se verifica en su documentación antes de adoptarla para el dashboard. Si no lo soporta, se usa un servidor SSE de prueba en el harness.

Distinción obligatoria:

- **Interceptors de Axios o fetch** (headers, reintentos, logging) son código de producción. Se prueban por su efecto observable en la red y **no se usan como mecanismo de mock**.
- **No se mockean** módulos de cliente HTTP (`axios`, `fetch`, servicios propios de API) con `jest.mock` o equivalentes.

Fidelidad al contrato:

- Los handlers se definen en `tests/support/network/` y se construyen con los Object Mothers (sección 7).
- Los handlers de error devuelven `application/problem+json` con la misma estructura que el backend (`api-standards.md`).
- Cada prueba declara los handlers de su escenario. Las requests no manejadas hacen fallar la prueba (`onUnhandledRequest: 'error'`).
- Los handlers se validan contra el contrato OpenAPI (sección 9).

## 9. Contrato con OpenAPI

El contrato OpenAPI del backend es la **fuente de verdad** del contrato HTTP. Garantiza que los handlers de interceptación de red (sección 8) describen el backend real.

- La ubicación del archivo OpenAPI y las herramientas de validación se definen en la Spec de la API.
- **Lado proveedor (backend):** unas pocas pruebas de servicio validan que las respuestas reales cumplen el esquema, incluidos los éxitos y los Problem Details. Basta una por endpoint y status code relevante. En Go puede usarse un validador como `kin-openapi`.
- **Lado consumidor (frontend y móvil):** las respuestas de los handlers de MSW se validan contra el mismo contrato. Un handler que no cumple el esquema hace fallar la prueba. Los tipos o los handlers pueden generarse desde el contrato.
- Los consumidores tienen pruebas que verifican cómo manejan cada status code y tipo de problema documentado que les afecta.
- Un cambio incompatible en el contrato debe romper una prueba en CI antes del merge.
- Los cambios del contrato siguen las reglas de versionado de `api-standards.md`.
- Consumer-driven contracts (Pact) no se adoptan en el MVP. Se reconsiderarán mediante ADR si aparecen varios consumidores con ciclos de versión independientes.

## 10. Backend (Go)

**Pruebas del núcleo** (la mayoría, en milisegundos):
- Entran por los casos de uso de `internal/application/usecases/`.
- Usan fakes en memoria para los `ports/` de salida.
- Cubren reglas de dominio, validación de eventos, idempotencia, deduplicación, estado del vehículo, alertas y las reglas de las herramientas del agente.
- Las reglas complejas del dominio pueden probarse directamente desde `internal/domain/` si su API es pública y estable.

**Pruebas de servicio** (pocas, con contenedores efímeros según 5.1):
- API de ingesta: enviar HTTP real al servidor levantado y verificar el status code, `application/problem+json` y que el evento llegó a Kafka.
- Worker: publicar en Kafka y verificar filas en TimescaleDB. Incluir duplicados, eventos inválidos y un reinicio a mitad de procesamiento.
- Adaptador de persistencia: verificar las consultas del dashboard y del agente contra TimescaleDB.
- Circuit Breaker y timeouts: inyectar fallos con Toxiproxy (sección 14) y verificar el comportamiento observable (respuesta degradada, recuperación).
- Contrato: validar las respuestas reales contra OpenAPI (sección 9).

**El agente de IA**: el LLM se sustituye por un fake determinista. Se prueban las herramientas del backend (validación, permisos, resultados, ausencia de datos), no la calidad de las respuestas del modelo.

## 11. Frontend (React + TypeScript)

- La unidad es un **flujo del usuario** sobre una página o feature, no un componente aislado.
- HTTP y SSE se sustituyen mediante interceptación a nivel de red (sección 8). No se mockean módulos ni hooks propios.
- Las verificaciones se hacen sobre lo visible: textos, roles, elementos del mapa y alertas.
- Escenarios obligatorios por feature: carga, datos, sin datos, error y desconexión/reconexión SSE.
- Chat del agente: enviar pregunta, estado de espera, respuesta y error.
- Lógica pura de `domain/` y `application/` (formateo de estado, reglas de alertas del cliente): pruebas directas.
- No usar snapshots, no verificar props ni estado interno, no depender del orden de renders.

## 12. Móvil (React Native)

- El núcleo de la sincronización se prueba entrando por los casos de uso de `application/`, con fakes de ubicación y conectividad, **SQLite en memoria** (5.1) y la API del backend sustituida por interceptación a nivel de red (sección 8).
- Escenarios obligatorios:
  - Captura sin conexión: los registros quedan pendientes.
  - Recuperación de la conectividad: se envía el lote.
  - Resultado parcial: solo se eliminan los registros confirmados.
  - Fallo o timeout: no se pierde nada y se reintenta.
  - Reenvío de un lote ya enviado: no se generan duplicados.
  - Permiso de ubicación denegado.
- Pocas pruebas de pantalla para los flujos principales, siguiendo los mismos criterios que el frontend.
- Las APIs nativas (GPS, NetInfo, SQLite nativo) solo se tocan a través de los adaptadores de `infrastructure/`. Las pruebas nunca las mockean directamente.

## 13. Property-based testing para la idempotencia

Se aplica **solo** a la idempotencia y la deduplicación (RNF-04): el procesamiento del worker y la sincronización móvil. No se usa de forma general.

Propiedades mínimas:

- Procesar el mismo evento N veces deja el mismo estado que procesarlo una vez.
- Procesar un lote con duplicados produce exactamente un registro por identificador de evento único.
- Si la Spec define el estado del vehículo por el timestamp del evento, el estado final no depende del orden de llegada.
- Móvil: ante cualquier secuencia de respuestas (éxito, resultado parcial, fallo, timeout), un registro local solo se elimina si fue confirmado y ningún registro no confirmado se pierde.

Reglas:

- Entran por los casos de uso del núcleo, con fakes, y forman parte de la suite rápida.
- Los generadores se construyen a partir de los Object Mothers (sección 7), así solo producen datos válidos salvo que la propiedad pida lo contrario.
- La seed se fija o se registra en cada ejecución para poder reproducir el fallo.
- Cuando una propiedad falla, el caso mínimo encontrado (shrinking) se añade como prueba de ejemplo permanente.
- Herramientas de referencia: `rapid` (Go) y `fast-check` (TypeScript).

## 14. Resiliencia entre servicios con Toxiproxy

Toxiproxy se coloca entre un servicio y sus dependencias (Kafka, TimescaleDB, el servicio del agente o el LLM) para inyectar latencia, cortes, timeouts y cierres de conexión.

- Se usa en la suite de integración, junto con los contenedores efímeros (5.1), y en el entorno de caos. Nunca en producción.
- Escenarios mínimos:
  - Dependencia lenta: se respeta el timeout y se devuelve un error controlado (Problem Details con `503` cuando corresponda).
  - Fallos repetidos: el Circuit Breaker se abre y deja de llamar a la dependencia; la respuesta es degradada y controlada.
  - Recuperación: al volver la dependencia, el Circuit Breaker se recupera y el servicio vuelve a responder con normalidad.
  - Worker: corte de TimescaleDB durante el procesamiento. Tras la recuperación no hay eventos perdidos ni duplicados, y el offset no avanzó antes de persistir.
- Se verifica el comportamiento observable (respuestas, filas persistidas, eventos reprocesados), no el estado interno del Circuit Breaker.
- Los umbrales de apertura, los timeouts y los tiempos de recuperación se toman de la Spec de Circuit Breaker. Las pruebas no inventan valores.
- Complementa a k6: k6 genera la carga y Toxiproxy degrada las dependencias. Pueden combinarse en un mismo escenario de caos.

## 15. End-to-end

- Uno o dos recorridos completos con Docker Compose: lote desde un cliente simulado → API → Kafka → worker → TimescaleDB → dashboard actualizado.
- Se ejecutan en un entorno de pruebas (local, CI o servidor dedicado), nunca contra producción.
- Verifican que las piezas encajan. No repiten casos que ya cubren las pruebas del núcleo.

## 16. Carga y caos (k6)

- Los scripts viven en `load-tests/` y simulan cientos de vehículos con un 10 % de requests duplicados y un 5 % de requests con errores inyectados.
- Se ejecutan contra un entorno de pruebas dedicado, nunca contra producción.
- Se trata el sistema como caja negra: solo HTTP hacia la API.
- Los payloads se generan con funciones equivalentes a los Object Mothers, con seed fija para que sean reproducibles.
- Verificaciones:
  - Los requests válidos reciben el status code esperado.
  - Los requests con errores reciben el Problem Details correcto.
  - Al terminar, el número de eventos persistidos coincide con los eventos únicos válidos enviados (sin pérdidas ni duplicados).
- Los `thresholds` de k6 se declaran como **objetivos**. Los resultados se reportan solo a partir de ejecuciones reales, indicando el entorno.
- Cada escenario documenta cómo ejecutarlo, sus parámetros y cómo interpretar los resultados.

## 17. Mutation testing en el núcleo

Mide el pilar de protección contra regresiones: comprueba si las pruebas detectan cambios en la lógica, algo que la cobertura no mide.

- Alcance: solo el núcleo. En el backend, `internal/domain/` e `internal/application/usecases/`. En frontend y móvil, `domain/` y `application/`. No se aplica a adaptadores, UI ni wiring.
- Herramientas de referencia: `gremlins` (Go) y Stryker (TypeScript).
- Cada mutante superviviente se analiza:
  - Si revela un comportamiento sin probar, se añade una prueba de comportamiento.
  - Si es equivalente o irrelevante, se ignora y se documenta el motivo.
- No se escriben pruebas acopladas a la implementación solo para matar mutantes. La resistencia al refactoring sigue sin negociarse.
- El umbral mínimo de mutation score se fija en la Spec después de una primera ejecución real. Hasta entonces el resultado es informativo y no bloquea el merge.
- El reporte se guarda como artefacto de CI, con fecha y commit.

## 18. Trazabilidad

Cada criterio de aceptación de una Spec (`.kiro/specs/`) se verifica con al menos una prueba, o tiene una justificación explícita de por qué se verifica de otra forma (demostración o revisión manual).

- Los criterios de aceptación tienen un identificador en la Spec. Los requisitos globales usan los de `product.md` (por ejemplo `RF-03`, `RNF-04`).
- Cada prueba que verifica un criterio lo referencia con un comentario encima de la prueba: `// Verifies: RF-03.2`. En k6 se usa un tag con el mismo identificador.
- La Spec mantiene una matriz criterio → prueba, con la ruta de la prueba y su suite.
- Una tarea no se da por terminada si alguno de sus criterios no tiene prueba ni justificación.
- Los criterios verificados solo por E2E o k6 se marcan así en la matriz, y su evidencia procede de ejecuciones reales.
- Las pruebas en cuarentena (sección 19) no cuentan como verificación.

## 19. Bugs y pruebas inestables

**Regresión primero:**

- Todo bug corregido incluye una prueba que lo reproduce.
- La prueba se escribe antes del fix y debe fallar. Después del fix debe pasar.
- Se escribe en la suite más rápida capaz de detectar el bug y referencia el issue: `// Regression: #123`.

**Pruebas inestables (flaky):**

- Están prohibidos los reintentos automáticos para ocultar fallos (`retries` en Playwright o Jest, reruns de jobs en CI).
- Una prueba inestable se pone en cuarentena: se desactiva con un issue asociado, un responsable y un plazo definido en el issue.
- La cuarentena es temporal. La prueba se corrige (la causa suele ser una espera por tiempo, estado compartido o falta de determinismo) o se elimina.

## 20. Pruebas de seguridad

Las pruebas de seguridad aplican los requisitos de `security.md` y siguen los mismos principios de caja negra: entran por los bordes públicos y verifican el comportamiento observable.

- Los criterios de seguridad de cada funcionalidad forman parte de sus criterios de aceptación y de la trazabilidad (sección 18).
- Se prueban la validación de entradas, la autenticación, la autorización y el manejo de errores.
- Autorización: se prueban tanto los accesos permitidos como los denegados, incluidos los accesos horizontales y verticales no autorizados.
- Entradas maliciosas y límites: payloads malformados, valores fuera de rango, campos desconocidos y lotes o requests por encima del tamaño máximo.
- Errores: las respuestas de error usan Problem Details y no exponen trazas, SQL, rutas internas ni secretos.
- Que las pruebas pasen no demuestra que la aplicación sea segura (`security.md`, sección 13).

## 21. Reglas de escritura

- Nombre de la prueba en inglés, describiendo el comportamiento en lenguaje de negocio: `rejects_batch_when_coordinates_are_out_of_range`, no `TestValidateCoords2`.
- Estructura Arrange / Act / Assert, con un solo Act por prueba.
- Sin lógica condicional ni bucles en el cuerpo de la prueba. Para variantes se usan table-driven tests o `it.each`.
- Las pruebas son independientes entre sí y no dependen del orden de ejecución.
- Sin `sleep`. Se espera por condición con timeout.
- No exponer código privado ni añadir código de producción solo para poder probarlo.

## 22. Antipatrones prohibidos

- Mockear clases o funciones internas.
- Mockear módulos de cliente HTTP (`axios`, `fetch`, servicios propios de API) en lugar de interceptar a nivel de red.
- Usar interceptors de producción como mecanismo de mock.
- Verificar el orden o el número de llamadas internas cuando no son el comportamiento.
- Snapshots de UI.
- Probar métodos privados o detalles de implementación.
- Duplicar la lógica de producción dentro de la prueba para calcular el resultado esperado.
- Una prueba por cada clase o archivo "porque sí".
- Pruebas que dependen de datos compartidos, de objetos mutables reutilizados o de la hora real.
- Ejecutar pruebas automatizadas contra producción o contra bases de datos compartidas.
- Handlers de red que no se validan contra el contrato OpenAPI.
- Pruebas acopladas a la implementación escritas solo para matar mutantes.
- Reintentos automáticos de pruebas para ocultar inestabilidad.
- Corregir un bug sin una prueba que lo reproduzca.

## 23. Checklist de revisión

Antes de aceptar una prueba:

1. ¿Entra y sale por bordes públicos?
2. ¿Pasaría si reescribo la implementación con el mismo comportamiento?
3. ¿Falla si rompo la regla que dice proteger?
4. ¿Solo sustituye dependencias no gestionadas, y las dependencias gestionadas son efímeras y aisladas?
5. ¿La red se sustituye por interceptación y no mockeando el cliente HTTP?
6. ¿Los datos salen de Object Mothers y builders, y la prueba solo muestra lo relevante?
7. ¿El acoplamiento a frameworks queda dentro del harness?
8. ¿Es determinista y rápida para su nivel?
9. ¿Referencia el criterio de aceptación que verifica?
10. Si corrige un bug, ¿falló antes del fix?

## 24. Referencias

- [Object Mother — Martin Fowler](https://martinfowler.com/bliki/ObjectMother.html)
- [Combining Object Mother and Fluent Builder — reflectoring.io](https://reflectoring.io/objectmother-fluent-builder/)
- [Mock Service Worker](https://mswjs.io/)
- [MSW: integración con React Native](https://mswjs.io/guides/integrations/react-native)
- [OpenAPI Specification](https://spec.openapis.org/oas/latest.html)
- [Toxiproxy](https://github.com/Shopify/toxiproxy)
- [Stryker Mutator](https://stryker-mutator.io/)
- [Gremlins](https://gremlins.dev/)
- [rapid](https://github.com/flyingmutant/rapid)
- [fast-check](https://fast-check.dev/)
