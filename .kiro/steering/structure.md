---
inclusion: always
---

# Estructura y arquitectura del proyecto

## 1. Propósito

Este documento define la organización lógica del repositorio y las reglas de dependencia entre componentes del Portal Corporativo de Monitoreo de Flotas y Telemetría.

La solución deberá aplicar Clean Architecture en backend, frontend web y aplicación móvil, adaptando sus principios a las responsabilidades de cada aplicación.

La estructura propuesta es una guía para organizar el desarrollo. Los directorios definitivos podrán ajustarse durante la implementación, siempre que se mantengan los límites arquitectónicos establecidos.

## 2. Organización general del repositorio

La estructura inicial propuesta es:

```text
fleet-platform/
├── .github/
│   └── workflows/
├── .kiro/
│   ├── steering/
│   │   ├── product.md
│   │   ├── tech.md
│   │   ├── structure.md
│   │   └── git-workflow.md
│   └── specs/
├── backend/
│   ├── cmd/
│   ├── internal/
│   ├── migrations/
│   └── tests/
├── frontend/
│   ├── src/
│   └── tests/
├── mobile/
│   ├── src/
│   └── tests/
├── infrastructure/
│   └── terraform/
├── load-tests/
├── docs/
│   └── adr/
├── infrastructure/docker/docker-compose.yml
├── .gitignore
└── README.md
```

Esta estructura es una propuesta inicial, no una afirmación de que todos los directorios ya existan.

Los directorios y archivos se crearán progresivamente cuando sean necesarios para una funcionalidad o una configuración concreta. No se deberán generar carpetas vacías ni capas sin una responsabilidad definida.

## 3. Backend

El backend podrá ejecutarse como dos procesos independientes: la API HTTP y el worker de telemetría.

Ambos podrán compartir dominio, casos de uso y adaptadores comunes sin acoplar el dominio a los procesos de ejecución.

### 3.1 Estructura propuesta

```text
backend/                  # directorio padre para los servicios de backend
└── go/                   # implementación del servicio en Go
    ├── cmd/
    │   ├── api/
    │   │   └── main.go
    │   └── worker/
    │       └── main.go
    ├── internal/
    │   ├── domain/
    │   ├── application/
    │   │   ├── ports/
    │   │   └── usecases/
    │   ├── adapters/
    │   │   ├── inbound/
    │   │   │   └── http/
    │   │   └── outbound/
    │   │       ├── messaging/
    │   │       └── persistence/
    │   └── infrastructure/
    │       └── config/
    ├── migrations/
    └── tests/
```

Los nombres concretos podrán adaptarse si una organización diferente mejora la claridad sin romper las reglas de dependencia.

### 3.2 Dominio

`internal/domain/` contendrá los conceptos y reglas de negocio que no dependan de frameworks ni de infraestructura.

Podrá incluir:

- Entidades.
- Objetos de valor.
- Reglas e invariantes del dominio.
- Errores propios del dominio.

El dominio no deberá importar Gin, clientes de Kafka, controladores SQL ni bibliotecas de infraestructura.

### 3.3 Aplicación

`internal/application/` contendrá los casos de uso y la coordinación de operaciones de negocio.

`internal/application/ports/` definirá las interfaces que los casos de uso necesitan para comunicarse con sistemas externos.

`internal/application/usecases/` contendrá las acciones que el sistema debe ejecutar, por ejemplo, recibir una solicitud de ingesta o procesar un evento de telemetría.

Los casos de uso no deberán depender de implementaciones concretas de bases de datos, brokers o frameworks HTTP.

### 3.4 Adaptadores de entrada

`internal/adapters/inbound/http/` contendrá los controladores HTTP, las rutas, la validación de las solicitudes y la conversión entre los modelos HTTP y los comandos o resultados de aplicación.

Los controladores deberán ser delgados: no deberán implementar reglas de negocio ni acceder directamente a la base de datos.

El worker podrá disponer de adaptadores de entrada propios para consumir mensajes y activar los casos de uso correspondientes.

### 3.5 Adaptadores de salida

`internal/adapters/outbound/messaging/` contendrá las implementaciones de las interfaces de mensajería, incluida la integración con Kafka.

`internal/adapters/outbound/persistence/` contendrá las implementaciones de persistencia, incluida la integración con TimescaleDB.

Los adaptadores traducirán entre los contratos internos de la aplicación y los detalles técnicos externos.

### 3.6 Infraestructura

`internal/infrastructure/config/` podrá contener la carga y validación de configuración, la composición de dependencias y la inicialización de recursos compartidos.

La infraestructura podrá conocer las implementaciones concretas de los adaptadores para conectarlas con los puertos definidos por la aplicación.

La composición de dependencias no deberá trasladar decisiones de negocio a `main.go`.

## 4. Frontend web

### 4.1 Estructura propuesta

```text
frontend/
└── src/
    ├── domain/
    ├── application/
    ├── infrastructure/
    │   ├── http/
    │   └── realtime/
    ├── presentation/
    │   ├── components/
    │   ├── features/
    │   ├── pages/
    │   └── hooks/
    └── main.tsx
```

La estructura exacta dependerá de las necesidades de la interfaz. Se evitará crear módulos sin responsabilidades concretas.

### 4.2 Responsabilidades

- `domain/`: conceptos, reglas y tipos propios del dominio que necesite el cliente web.
- `application/`: coordinación de operaciones del cliente y casos de uso de presentación.
- `infrastructure/`: clientes HTTP, suscripciones SSE y otras integraciones externas.
- `presentation/`: componentes React, páginas, hooks y composición visual.

Los componentes de presentación no deberán acceder directamente a la base de datos ni implementar la lógica de negocio del backend.

Los detalles de HTTP y SSE deberán permanecer encapsulados en adaptadores para facilitar las pruebas y los cambios de infraestructura.

El frontend no deberá convertirse en la autoridad final de las reglas críticas del dominio.

## 5. Aplicación móvil

### 5.1 Estructura propuesta

```text
mobile/
└── src/
    ├── domain/
    ├── application/
    ├── infrastructure/
    │   ├── location/
    │   ├── persistence/
    │   ├── connectivity/
    │   └── api/
    └── presentation/
        ├── components/
        ├── screens/
        └── hooks/
```

### 5.2 Responsabilidades

- `domain/`: conceptos y reglas de negocio de la captura y sincronización de telemetría.
- `application/`: casos de uso para capturar coordenadas, almacenar registros pendientes y sincronizar lotes.
- `infrastructure/location/`: integración con las capacidades de ubicación del dispositivo.
- `infrastructure/persistence/`: implementación de almacenamiento local mediante SQLite.
- `infrastructure/connectivity/`: mecanismos para conocer y gestionar cambios de conectividad.
- `infrastructure/api/`: comunicación HTTP con el backend.
- `presentation/`: pantallas, componentes y hooks de la aplicación.

La lógica de sincronización no deberá quedar dispersa entre componentes de React Native.

La persistencia local y el acceso a la ubicación deberán encapsularse detrás de contratos que permitan sustituir implementaciones y probar los casos de uso sin depender de un dispositivo físico.

La eliminación de registros pendientes deberá respetar las garantías de confirmación definidas para la sincronización.

## 6. Infraestructura y automatización

### 6.1 Infraestructura

`infrastructure/terraform/` contendrá los módulos y configuraciones Terraform que se incorporen para definir la infraestructura objetivo.

Los módulos deberán organizarse según las responsabilidades reales de los recursos desplegados.

No se deberá afirmar que la infraestructura de AWS está desplegada cuando solo existan archivos de configuración.

### 6.2 CI/CD

`.github/workflows/` contendrá los flujos de automatización de GitHub Actions.

Los flujos deberán separar, según corresponda:

- Validaciones de código.
- Ejecución de pruebas.
- Construcción de artefactos.
- Validaciones de infraestructura.
- Automatización móvil.

Fastlane podrá complementar los flujos móviles cuando sea necesario.

Los pipelines deberán reutilizar comandos documentados y evitar duplicar la lógica de construcción y validación.

### 6.3 Pruebas de carga

`load-tests/` contendrá los escenarios y scripts de k6. Su diseño y documentación siguen `testing.md`.

## 7. Specs y documentación

### 7.1 Steering

`.kiro/steering/` contendrá las reglas generales del proyecto que deben orientar el trabajo de Kiro.

- `product.md`: propósito, alcance y requisitos del producto.
- `tech.md`: stack tecnológico y restricciones técnicas.
- `structure.md`: organización y dependencias arquitectónicas.
- `testing.md`: estándar único de pruebas.
- `git-workflow.md`: convenciones de ramas, commits y colaboración.

Estos documentos deberán mantener responsabilidades diferenciadas y evitar duplicaciones extensas.

### 7.2 Specs

`.kiro/specs/` contendrá las especificaciones de funcionalidades concretas.

Cada Spec deberá desarrollar los requisitos, el diseño y las tareas necesarias según el flujo de trabajo de Kiro.

Las Specs deberán referenciar las reglas generales de Steering y evitar redefinirlas de forma contradictoria.

### 7.3 ADR

`docs/adr/` contendrá los Architecture Decision Records.

Cada ADR deberá documentar una decisión arquitectónica relevante, su contexto, las alternativas evaluadas, la decisión adoptada y sus consecuencias.

No será necesario crear un ADR para cada detalle de implementación.

## 8. Reglas de dependencia

La dirección general de dependencias deberá respetar las siguientes reglas:

1. El dominio no dependerá de las capas externas.
2. La aplicación podrá depender del dominio y de los contratos que necesita.
3. Los adaptadores implementarán los contratos de entrada y salida correspondientes.
4. La infraestructura podrá conectar las implementaciones concretas con los contratos internos.
5. Los frameworks se mantendrán en los límites de la arquitectura.
6. Los procesos de ejecución coordinarán dependencias, pero no contendrán reglas de negocio.
7. Las pruebas deberán poder verificar las reglas de dominio y los casos de uso sin requerir siempre todos los servicios externos.

La aplicación web y la aplicación móvil podrán tener estructuras diferentes de la del backend, pero deberán conservar la separación entre reglas de negocio, casos de uso, infraestructura y presentación.

## 9. Criterios para modificar la estructura

Antes de crear una nueva capa, paquete o módulo, deberá existir una responsabilidad concreta que lo justifique.

Los cambios de estructura deberán:

- Mantener límites claros entre dominio, aplicación e infraestructura.
- Evitar dependencias circulares.
- Evitar duplicación de reglas de negocio.
- Facilitar las pruebas automatizadas.
- Mantener una complejidad proporcional a las necesidades del MVP.

Si una modificación afecta significativamente los límites arquitectónicos o introduce un patrón nuevo, deberá justificarse en la Spec correspondiente y documentarse mediante ADR cuando corresponda.

## Lenguaje y estilo

Toda la documentación de este proyecto debe seguir las convenciones establecidas en `lenguaje.md`:

- **Documentación y specs:** escritos en español, con términos técnicos y acrónimos en inglés
- **Código:** nombres en inglés siguiendo convenciones de Go, TypeScript y React
- **Comentarios en código:** en inglés (estándar Go)
- **Mensajes de error al usuario:** en español
- **Logs de producción:** en inglés (compatibilidad con herramientas de monitoreo)

Ver el documento completo en `.kiro/steering/lenguaje.md`.
