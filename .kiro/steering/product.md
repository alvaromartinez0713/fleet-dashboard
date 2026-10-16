---
inclusion: always
---

# Producto: Portal Corporativo de Monitoreo de Flotas y Telemetría

## 1. Propósito del producto

Desarrollar un MVP funcional de un Portal Corporativo para el Monitoreo de Flotas que integre captura móvil de coordenadas, procesamiento asíncrono de telemetría, visualización web reactiva y consultas operativas mediante un agente de inteligencia artificial.

El proyecto busca demostrar la capacidad de diseñar y desarrollar un ecosistema resiliente, aplicando Clean Architecture, principios de diseño orientado al dominio y buenas prácticas de ingeniería con apoyo de herramientas de desarrollo agéntico.

La evaluación contempla tanto el funcionamiento del MVP como la calidad de las decisiones arquitectónicas, la gestión de datos, la resiliencia y la auditoría del código generado mediante IA.

## 2. Alcance y actores

### 2.1 Actores

El enunciado identifica una aplicación móvil para el conductor, un portal corporativo y un agente operativo de IA. No define una matriz detallada de roles ni permisos administrativos.

- **Conductor:** utiliza la aplicación móvil para capturar y enviar coordenadas.
- **Usuario del portal:** consulta información de monitoreo de flotas e interactúa con el agente operativo.
- **Sistema de telemetría:** recibe, transporta, procesa y almacena los eventos generados por los dispositivos.
- **Agente operativo de IA:** consulta el estado de los vehículos y responde preguntas en lenguaje natural.

El sistema de telemetría y el agente de IA son componentes de la solución, no actores humanos.

## 3. Requisitos funcionales

### RF-01. Captura móvil de coordenadas

La aplicación móvil deberá capturar coordenadas de ubicación y permitir su envío al backend.

La frecuencia de captura, los permisos del sistema operativo y las condiciones de ejecución se definirán en las especificaciones técnicas.

### RF-02. Persistencia local y funcionamiento sin conexión

La aplicación móvil deberá conservar localmente las coordenadas cuando no exista conectividad, utilizando una solución apropiada, como SQLite.

La pérdida de conexión no deberá provocar la pérdida inmediata de los registros pendientes.

### RF-03. Sincronización por lotes

Cuando se recupere la conectividad, la aplicación móvil deberá sincronizar en bloque los registros acumulados localmente.

La implementación deberá definir cómo identificar los eventos, gestionar reintentos, procesar resultados parciales y evitar pérdidas o duplicaciones. La eliminación de registros locales deberá depender del mecanismo de confirmación definido para el procesamiento remoto, no únicamente de la aceptación inicial de una petición.

### RF-04. Ingesta asíncrona de telemetría

El backend deberá implementar un flujo de ingesta asíncrona mediante un bus de eventos que desacople la recepción de telemetría de su procesamiento posterior.

Kafka será la tecnología inicial seleccionada para mensajería y procesamiento orientado a eventos.

### RF-05. Procesamiento y persistencia de eventos

El sistema deberá procesar los eventos recibidos y almacenarlos en una solución adecuada para datos de alta frecuencia y series de tiempo.

TimescaleDB será la opción inicial para la persistencia de telemetría. Su selección deberá justificarse considerando las consultas previstas, la integridad de los eventos y las necesidades del MVP.

### RF-06. Consulta del estado de los vehículos

El sistema deberá proporcionar información procesada para consultar el estado actual de los vehículos.

Las reglas concretas para determinar el estado y tratar información desactualizada se definirán en las especificaciones funcionales correspondientes.

### RF-07. Dashboard web reactivo

El portal web deberá presentar información de monitoreo de flotas en un dashboard que incluya:

- Visualización de vehículos en un mapa.
- Presentación de alertas en tiempo real.
- Actualización reactiva de la información mediante Server-Sent Events (SSE) o WebSockets.

La selección del mecanismo de actualización y su estrategia de recuperación ante desconexiones se documentarán como decisiones técnicas.

### RF-08. Agente operativo de inteligencia artificial

El backend deberá integrar un agente operativo capaz de consultar el estado actual de los vehículos y responder preguntas en lenguaje natural.

Un ejemplo planteado en el enunciado es identificar vehículos que llevan detenidos más de veinte minutos en zonas críticas.

El agente deberá acceder a la información mediante capacidades controladas del backend. Las reglas críticas de validación y autorización permanecerán bajo responsabilidad de la aplicación, no del modelo de IA.

LangChain será la biblioteca inicial seleccionada para la integración del agente. Su utilización deberá responder a necesidades concretas del MVP y evitar complejidad innecesaria.

### RF-09. Chat integrado con el portal

El dashboard deberá incluir una interfaz de chat para interactuar con el agente operativo.

La interfaz deberá presentar las consultas y respuestas, además de gestionar los estados de espera y los errores de comunicación.

### RF-10. Resiliencia en la comunicación entre servicios

El backend deberá implementar mecanismos Circuit Breaker en las comunicaciones entre servicios que lo justifiquen.

La solución deberá definir las dependencias protegidas y el comportamiento ante fallos temporales. El Circuit Breaker complementará los mecanismos de reintento, idempotencia y recuperación del procesamiento asíncrono.

### RF-11. Pruebas de carga y escenarios de caos

El proyecto deberá incluir scripts de prueba con k6 capaces de simular cientos de vehículos.

Los escenarios deberán incorporar un 10 % de peticiones duplicadas y un 5 % de peticiones con errores inyectados.

Los resultados deberán basarse en ejecuciones reales y permitir analizar el comportamiento del sistema bajo carga y ante fallos.

### RF-12. Ejecución local

El proyecto deberá proporcionar una configuración de Docker Compose que permita ejecutar localmente los componentes necesarios para demostrar el MVP.

Cuando las limitaciones del entorno impidan ejecutar todos los servicios, podrán utilizarse mocks ligeros o servicios de nube disponibles, documentando las sustituciones y sus limitaciones.

### RF-13. Infraestructura como código

El repositorio deberá incluir infraestructura como código mediante Terraform para definir la infraestructura objetivo en AWS.

El diseño deberá tener presente que el contexto tecnológico del enunciado menciona AWS Control Tower y una organización multi-cuenta. No se afirmará que estas capacidades están desplegadas si no forman parte de la implementación realizada.

### RF-14. Automatización CI/CD móvil

La aplicación móvil deberá incluir archivos de automatización para integración y despliegue mediante GitHub Actions, Fastlane o una combinación apropiada.

La documentación deberá diferenciar las tareas automatizadas de aquellas que requieran configuración o intervención manual.

## 4. Requisitos no funcionales y principios de calidad

### RNF-01. Clean Architecture

La solución deberá aplicar Clean Architecture en el backend, el frontend web y la aplicación móvil.

Las dependencias deberán orientarse hacia el dominio y los casos de uso, manteniendo los detalles de infraestructura y los frameworks en los límites correspondientes.

- **Backend:** separar dominio, aplicación, adaptadores e infraestructura. La API de ingesta y el worker de telemetría podrán ejecutarse como procesos independientes y compartir módulos internos cuando corresponda.
- **Frontend web:** separar dominio, aplicación, infraestructura y presentación. Los componentes de React no deberán concentrar reglas de negocio que correspondan al dominio o a los casos de uso.
- **Aplicación móvil:** separar dominio, aplicación, infraestructura y presentación. La captura de ubicación, la persistencia local, la conectividad y la comunicación HTTP deberán tratarse como dependencias externas accesibles mediante adaptadores.

La arquitectura deberá mantener límites claros sin introducir abstracciones innecesarias.

### RNF-02. Diseño orientado al dominio

El sistema deberá identificar y modelar los conceptos y reglas relevantes del dominio de monitoreo de flotas y telemetría.

Las entidades, los objetos de valor y los casos de uso concretos se definirán en las especificaciones, evitando introducir conceptos de negocio sin justificación.

### RNF-03. Desacoplamiento del procesamiento

La recepción de telemetría deberá estar desacoplada de su procesamiento y persistencia mediante el bus de eventos.

Los fallos de consumidores y almacenamiento deberán contemplarse en las estrategias de reintento y recuperación.

### RNF-04. Integridad e idempotencia

El sistema deberá contemplar duplicados, reintentos, desconexiones y procesamiento repetido.

Las especificaciones deberán establecer cómo se identifica cada evento, cómo se detectan duplicados y qué garantías proporciona cada etapa del flujo.

No se afirmará que existe entrega exactamente una vez de extremo a extremo si la implementación no garantiza esa propiedad.

### RNF-05. Persistencia especializada

La solución de almacenamiento deberá justificarse teniendo en cuenta la frecuencia de los eventos, las consultas previstas y el alcance del MVP.

Se documentarán las decisiones relevantes sobre tablas, índices, consultas e idempotencia, incluyendo las limitaciones conocidas.

### RNF-06. Resiliencia

Las comunicaciones remotas deberán gestionar explícitamente errores y tiempos de espera.

Los mecanismos de resiliencia deberán aplicarse según el tipo de dependencia y evitar reintentos que provoquen duplicaciones incontroladas o pérdida silenciosa de eventos.

### RNF-07. Operación offline-first

La aplicación móvil deberá conservar las coordenadas pendientes cuando no haya conectividad y sincronizarlas al recuperarla.

La sincronización deberá definir el tratamiento de confirmaciones, reintentos y fallos parciales de los lotes.

### RNF-08. Actualización reactiva

El portal deberá recibir actualizaciones mediante SSE o WebSockets, según la decisión técnica adoptada.

Deberá definirse cómo se recupera la información que no se haya recibido durante una desconexión.

### RNF-09. Uso controlado de IA

El agente deberá acceder a la información operativa mediante capacidades definidas por el backend.

Las validaciones críticas y el control de acceso permanecerán bajo responsabilidad de la aplicación. Se deberán contemplar errores, respuestas incompletas y limitaciones del agente.

### RNF-10. Calidad verificable

Las afirmaciones sobre rendimiento, escalabilidad y resiliencia deberán respaldarse mediante pruebas o identificarse como objetivos pendientes de validación.

La calidad deberá demostrarse mediante código revisado, pruebas ejecutables, documentación y evidencia de los resultados.

### RNF-11. Trazabilidad de decisiones

Las decisiones arquitectónicas relevantes deberán documentarse mediante ADR cuando corresponda, relacionándolas con los requisitos que las motivan.

Las propuestas generadas por IA deberán revisarse antes de integrarse, prestando atención a seguridad, complejidad, mantenibilidad y coherencia arquitectónica.

## 5. Entregables de la prueba

Además del MVP funcional, la evaluación contempla los siguientes entregables:

### EP-01. Documentación y auditoría del desarrollo agéntico

El README deberá incluir instrucciones de ejecución y uso de la infraestructura como código.

También deberá documentar al menos dos casos reales en los que una herramienta agéntica haya propuesto una solución deficiente, insegura o no escalable, y explicar cómo se corrigió mediante revisión técnica y refactorización.

Los casos de auditoría deberán proceder del trabajo efectivamente realizado y no de ejemplos inventados.

### EP-02. Sustentación del proyecto

Se deberá preparar una demostración y sustentación del proyecto que explique el funcionamiento del MVP y las decisiones técnicas adoptadas.

La duración y el formato de la sustentación deberán ajustarse a las indicaciones del enunciado de la prueba.

## 6. Stack de referencia del MVP

El enunciado permite elegir las tecnologías. Para este proyecto se propone el siguiente stack inicial:

| Componente | Tecnología seleccionada |
|---|---|
| Backend | Go con Gin |
| Frontend web | React con TypeScript |
| Aplicación móvil | React Native |
| Mensajería y eventos | Kafka |
| Persistencia de telemetría | TimescaleDB |
| Agente de inteligencia artificial | LangChain |
| Actualizaciones del dashboard | SSE, sujeto a validación |
| Entorno local | Docker Compose |
| Infraestructura como código | Terraform para AWS |
| Pruebas de carga y caos | k6 |
| CI/CD | GitHub Actions y Fastlane, según las necesidades del flujo móvil |

Este stack representa la dirección inicial del MVP. Las tecnologías se seleccionan para mantener una solución coherente y demostrable dentro del tiempo disponible.

El enunciado menciona React y Next.js para web, Go y .NET para backend, TimescaleDB y Cassandra para persistencia, además de Druid para analítica. No exige utilizar todas estas tecnologías simultáneamente. La elección de React con TypeScript, Go con Gin y TimescaleDB deberá justificarse en la documentación técnica.

Las decisiones relevantes podrán revisarse mediante ADR cuando exista una razón técnica concreta para hacerlo.

## 7. Alcance del MVP

El MVP deberá priorizar un recorrido funcional de extremo a extremo:

1. La aplicación móvil captura y conserva coordenadas.
2. La aplicación sincroniza los registros pendientes, incluidos los lotes acumulados sin conexión.
3. El backend recibe telemetría y publica eventos en Kafka.
4. Un consumidor procesa los eventos y los persiste en TimescaleDB.
5. El portal presenta la información de los vehículos mediante un mapa y alertas reactivas.
6. El usuario consulta al agente operativo desde el chat integrado.
7. El proyecto dispone de un entorno de demostración local, scripts de carga, infraestructura como código y automatización móvil.

Las simplificaciones motivadas por tiempo o recursos deberán documentarse, diferenciando capacidades implementadas, componentes simulados y trabajo pendiente.

## 8. Criterios globales de aceptación

El MVP deberá permitir demostrar que:

- La aplicación móvil captura coordenadas y las conserva localmente cuando no hay conexión.
- Los registros pendientes se sincronizan en bloque al recuperar la conectividad.
- La ingesta utiliza un bus de eventos para desacoplar recepción y procesamiento.
- Los eventos procesados se almacenan en la base de datos seleccionada.
- El procesamiento contempla duplicados y reintentos mediante una política documentada.
- El portal presenta un mapa, alertas y actualizaciones reactivas.
- El chat consulta al agente operativo y presenta respuestas basadas en información disponible.
- La comunicación entre servicios incorpora Circuit Breaker donde esté justificado.
- Los scripts de carga simulan cientos de vehículos, con un 10 % de peticiones duplicadas y un 5 % de errores inyectados.
- Docker Compose permite ejecutar el entorno de demostración, o se documentan las dependencias sustituidas.
- El repositorio contiene infraestructura como código, automatización CI/CD móvil y documentación de ejecución.
- El README incluye la auditoría de IA con al menos dos casos reales.
- Existe una demostración funcional y una sustentación que explica las decisiones técnicas.

Los criterios detallados de cada funcionalidad se definirán en las Specs correspondientes. Los objetivos cuantitativos de latencia, rendimiento, disponibilidad y retención se establecerán cuando sean necesarios para evaluar el comportamiento del sistema.

## 9. Decisiones pendientes de especificación

Las siguientes decisiones deberán resolverse progresivamente antes o durante la implementación de las funcionalidades correspondientes:

1. **Modelo de eventos:** identificadores, marcas temporales, validación y versionado.
2. **Ingesta y persistencia:** confirmación de publicación, tratamiento de errores y confirmación de procesamiento.
3. **Sincronización móvil:** contrato de lotes, resultados parciales y condiciones para eliminar registros locales.
4. **Estado de los vehículos:** información requerida y reglas para determinarlo.
5. **Actualización del portal:** elección definitiva entre SSE y WebSockets y estrategia de recuperación.
6. **Agente de IA:** herramientas disponibles, acceso a información operativa y tratamiento de errores.
7. **Circuit Breaker:** dependencias protegidas, condiciones de apertura y recuperación.
8. **Pruebas de carga y caos:** escenarios, métricas y criterios para interpretar resultados.
9. **Infraestructura y CI/CD:** componentes desplegados, secretos, configuración y alcance de la automatización.
