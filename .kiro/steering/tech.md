---
inclusion: always
---

# Guía técnica del proyecto

## 1. Propósito

Este documento establece el stack tecnológico de referencia, los criterios de implementación y las restricciones técnicas del Portal Corporativo de Monitoreo de Flotas y Telemetría.

Las decisiones deben ser coherentes con `product.md`, respetar Clean Architecture y favorecer una solución funcional, verificable y mantenible.

Las tecnologías descritas representan la selección inicial del proyecto. Cualquier cambio relevante deberá justificarse técnicamente y, cuando corresponda, documentarse mediante un Architecture Decision Record (ADR).

## 2. Stack tecnológico

| Área | Tecnología | Responsabilidad |
|---|---|---|
| Backend | Go | Lenguaje de implementación de los servicios de backend |
| Framework HTTP | Gin | Exposición de endpoints HTTP |
| Frontend web | React | Construcción de la interfaz del portal |
| Lenguaje web | TypeScript | Tipado estático del frontend |
| Aplicación móvil | React Native | Aplicación móvil para captura y sincronización de coordenadas |
| Persistencia móvil | SQLite | Almacenamiento local de registros pendientes |
| Mensajería | Apache Kafka | Desacoplamiento de la ingesta y el procesamiento |
| Base de datos | PostgreSQL con TimescaleDB | Persistencia y consulta de telemetría |
| Agente de IA | LangChain | Orquestación de consultas y herramientas del agente |
| Actualizaciones web | SSE, inicialmente | Entrega de actualizaciones del servidor al dashboard |
| Entorno local | Docker Compose | Ejecución coordinada de servicios para desarrollo y demostración |
| Infraestructura como código | Terraform | Definición de infraestructura objetivo en AWS |
| Pruebas de carga | k6 | Simulación de vehículos y generación de carga |
| CI/CD | GitHub Actions | Automatización de validaciones y flujos de integración |
| Automatización móvil | Fastlane | Automatización de tareas de construcción y distribución móvil |

La elección de bibliotecas auxiliares, versiones, gestores de paquetes y herramientas de pruebas se concretará durante las Specs de cada funcionalidad.

## 3. Backend

### 3.1 Principios

- Utilizar Go como lenguaje principal.
- Utilizar Gin como adaptador de entrada HTTP.
- Mantener el dominio y los casos de uso independientes de Gin, Kafka y TimescaleDB.
- Separar la recepción de telemetría de su procesamiento asíncrono.
- Validar las entradas en los límites del sistema.
- Gestionar explícitamente errores, cancelación y tiempos de espera.
- Evitar dependencias y abstracciones que no aporten valor al MVP.

### 3.2 API de ingesta

La API deberá validar las solicitudes, ejecutar el caso de uso correspondiente y publicar los eventos utilizando un adaptador de mensajería.

La aceptación HTTP de una solicitud y la confirmación de persistencia final son estados distintos. El contrato de la API deberá expresar con claridad qué se ha confirmado.

No se deberá afirmar que un evento está almacenado definitivamente en TimescaleDB únicamente porque la API haya aceptado la solicitud.

### 3.3 Worker de telemetría

El worker consumirá eventos de Kafka, validará los datos requeridos y coordinará su persistencia.

La implementación deberá contemplar:

- Procesamiento idempotente.
- Reintentos y errores recuperables.
- Manejo de eventos inválidos.
- Confirmación de mensajes según las garantías del procesamiento.
- Recuperación ante interrupciones.
- Observabilidad de fallos y resultados.

La estrategia de reintentos y confirmaciones deberá evitar tanto la pérdida silenciosa de eventos como el avance del offset antes de que se cumplan las garantías de procesamiento definidas.

## 4. Frontend web

- Utilizar React y TypeScript.
- Separar presentación, lógica de aplicación y acceso a servicios.
- Evitar que los componentes visuales concentren reglas de negocio.
- Gestionar explícitamente estados de carga, error, desconexión y ausencia de datos.
- Encapsular la comunicación HTTP y la suscripción a actualizaciones del servidor en adaptadores.
- Mantener las reglas del dominio fuera de los componentes de interfaz.

El mecanismo inicial de actualización será SSE. Si se determina que WebSockets es más adecuado, el cambio deberá justificarse considerando las necesidades reales de comunicación.

## 5. Aplicación móvil

- Utilizar React Native.
- Mantener separadas la presentación, la lógica de aplicación y la infraestructura.
- Utilizar SQLite para conservar los registros pendientes.
- Tratar la conectividad como una condición variable, no como una dependencia garantizada.
- Implementar la sincronización mediante casos de uso y adaptadores.
- Gestionar permisos de ubicación, errores de captura y fallos de sincronización.

La sincronización deberá conservar los registros locales hasta recibir la confirmación exigida por el contrato de procesamiento remoto.

Una respuesta HTTP exitosa o de aceptación no deberá interpretarse automáticamente como confirmación de persistencia definitiva si el contrato no ofrece esa garantía.

## 6. Mensajería y persistencia

### 6.1 Kafka

Kafka será el mecanismo inicial para desacoplar la ingesta y el procesamiento.

La implementación deberá definir los tópicos necesarios, la clave de particionamiento, el formato del evento y las políticas de reintento según los requisitos reales.

No se deberá asumir que Kafka garantiza por sí solo la idempotencia de las escrituras en la base de datos.

### 6.2 TimescaleDB

TimescaleDB será la base de datos inicial para almacenar telemetría.

El diseño deberá considerar:

- Esquema y tipos de datos.
- Marcas temporales.
- Identificación única de los eventos.
- Índices para las consultas previstas.
- Estrategia de inserción y tratamiento de duplicados.
- Consultas necesarias para el dashboard y el agente.
- Gestión de migraciones.

La optimización prematura y la incorporación de sistemas adicionales de almacenamiento deberán evitarse mientras no exista una necesidad demostrada.

## 7. Integración del agente de IA

LangChain será la opción inicial para orquestar el agente operativo.

El agente accederá a la información mediante herramientas controladas que exponga el backend. No deberá disponer de acceso indiscriminado a la base de datos ni ejecutar SQL arbitrario generado por el modelo.

Las herramientas deberán:

- Tener responsabilidades concretas y acotadas.
- Validar sus parámetros.
- Aplicar las restricciones de acceso correspondientes.
- Devolver resultados estructurados cuando sea posible.
- Gestionar errores y respuestas sin datos.
- Limitar las operaciones disponibles a las necesarias para el MVP.

Las reglas críticas de negocio deberán ejecutarse en el backend y no depender exclusivamente de las instrucciones del modelo.

## 8. Resiliencia y manejo de errores

Los mecanismos de resiliencia se aplicarán de acuerdo con el tipo de dependencia.

- Utilizar tiempos de espera en operaciones remotas.
- Aplicar reintentos limitados cuando la operación sea segura para repetir.
- Implementar Circuit Breaker en dependencias remotas donde aporte valor.
- Evitar reintentos ilimitados y ciclos de llamadas.
- Propagar errores con suficiente contexto para facilitar su diagnóstico.
- Evitar registrar secretos o información sensible.

El Circuit Breaker no reemplaza la idempotencia, la recuperación de consumidores ni las políticas de confirmación de Kafka.

## 9. Pruebas

La solución deberá incorporar pruebas en los niveles apropiados:

- **Unitarias:** reglas de dominio y casos de uso.
- **Integración:** adaptadores, persistencia y mensajería.
- **Contrato o API:** validación de entradas y respuestas.
- **Extremo a extremo:** recorridos prioritarios del MVP.
- **Carga y resiliencia:** scripts de k6 con los escenarios definidos en `product.md`.

Las pruebas deberán poder ejecutarse de forma reproducible y documentar sus dependencias.

Los resultados de rendimiento deberán proceder de ejecuciones reales. No se deberán inventar métricas ni presentar objetivos como resultados alcanzados.

## 10. Contenedores e infraestructura

Docker Compose será el mecanismo inicial para ejecutar localmente los servicios requeridos.

Terraform definirá la infraestructura objetivo en AWS. La configuración local y la infraestructura de nube deberán distinguirse claramente.

No se deberán incluir credenciales, tokens ni secretos en el repositorio. La configuración sensible deberá inyectarse mediante mecanismos apropiados para cada entorno.

## 11. Versiones y dependencias

- Fijar versiones compatibles de las herramientas y dependencias principales.
- Mantener los archivos de bloqueo correspondientes al gestor de paquetes utilizado.
- Evitar actualizaciones masivas sin necesidad.
- Revisar las dependencias antes de incorporarlas.
- No agregar una biblioteca cuando la funcionalidad pueda resolverse de forma más simple y mantenible sin ella.
- Documentar cambios relevantes del stack mediante ADR cuando corresponda.

Las versiones concretas se definirán durante la configuración inicial del proyecto y deberán verificarse en el entorno de desarrollo.

## 12. Seguridad y observabilidad

- Validar los datos recibidos en los límites del sistema.
- No confiar en datos proporcionados por clientes o modelos de IA.
- Gestionar adecuadamente los permisos de ubicación de la aplicación móvil.
- Proteger las credenciales mediante variables de entorno o mecanismos de secretos.
- Registrar errores y eventos operativos útiles sin exponer información sensible.
- Mantener separados los mensajes destinados al usuario y los detalles internos de diagnóstico.

La instrumentación adicional deberá incorporarse en función de las necesidades de diagnóstico y los criterios de aceptación del MVP.

## 13. Gestión de decisiones técnicas

Cuando una decisión afecte significativamente la arquitectura, la seguridad, el coste operativo o la mantenibilidad, deberá evaluarse y documentarse mediante un ADR.

Cada ADR deberá explicar el contexto, las opciones consideradas, la decisión adoptada y sus consecuencias.

Los cambios deberán mantener coherencia con los requisitos definidos en `product.md` y con los límites arquitectónicos de `structure.md`.

## Lenguaje y estilo

Toda la documentación de este proyecto debe seguir las convenciones establecidas en `lenguaje.md`:

- **Documentación y specs:** escritos en español, con términos técnicos y acrónimos en inglés
- **Código:** nombres en inglés siguiendo convenciones de Go, TypeScript y React
- **Comentarios en código:** en inglés (estándar Go)
- **Mensajes de error al usuario:** en español
- **Logs de producción:** en inglés (compatibilidad con herramientas de monitoreo)

Ver el documento completo en `.kiro/steering/lenguaje.md`.
