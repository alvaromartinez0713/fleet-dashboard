---
inclusion: always
---

# Guía de estilo y lenguaje

## 1. Propósito

Establecer las convenciones de lenguaje para la documentación del proyecto, incluyendo specs, documentación técnica, comentarios en código, mensajes de usuario y respuestas de API.

El proyecto sigue una política de **bilingüismo controlado** con tres categorías principales:
- **Documentación técnica** (specs, steering, README, documentación): en español
- **Respuestas de API** (headers HTTP, JSON bodies, fields, error responses): en inglés
- **Mensajes de interfaz de usuario** (UI displayed to end users): en español
- **Código** (nombres, variables, funciones, comentarios): en inglés

Esta distinción clara asegura que los contratos de API sean consistentes con los estándares industriales mientras que la documentación y la experiencia de usuario estén accesibles en español.

## 2. Lenguaje general

### Documentación y specs

- **Todos los documentos de spec, documentación técnica y reglas de steering deben escribirse en español**
- Los términos técnicos establecidos (API, HTTP, JSON, etc.) y acrónimos deben mantenerse en inglés
- Los nombres propios de tecnologías (Go, React, PostgreSQL, Kafka) deben mantenerse en inglés

### Respuestas de API

- **Todas las respuestas HTTP deben estar en inglés** (cabeceras, JSON fields, RFC 9457 Problem Details)
- Los nombres de campos en JSON (type, title, status, detail, instance, etc.) deben mantenerse en inglés
- Los códigos de estado HTTP y sus mensajes estándar deben mantenerse en inglés
- Los detalles técnicos internos (stack traces, errores de base de datos, trazas de ejecución) deben mantenerse en inglés

### Mensajes de usuario (UI)

- **Mensajes de error y notificaciones mostrados en la interfaz de usuario**: En español
- **Los mensajes que se muestran directamente al usuario en pantallas, alertas y componentes visuales**

### Código

- **Nombres de variables, funciones, métodos, structs, interfaces, tipos y campos deben escribirse en inglés** (siguiendo convenciones de Go, TypeScript, etc.)
- Los nombres de archivos y directorios deben escribirse en inglés
- Los comentarios en código deben escribirse en inglés (estándar en desarrollo Go)
- Los mensajes de log en producción deben escribirse en inglés para compatibilidad con herramientas de monitoreo

## 3. Términos técnicos que NO deben traducirse

### Lenguajes y frameworks

- Go, Gin, React, React Native, TypeScript, JavaScript, Node.js, Kotlin, Swift, Java, C#, Python

### Base de datos y persistencia

- PostgreSQL, TimescaleDB, SQLite, MySQL, MongoDB, Cassandra, Redis, Kafka, RabbitMQ
- SQL, NoSQL, ORM, Migrations, Schema, Table, Column, Index, Query, Database, DB

### Mensajería y arquitectura

- Kafka, RabbitMQ, Message Queue, Topic, Partition, Consumer, Producer, Offset
- Clean Architecture, Domain, Application, Infrastructure, Ports, Adapters
- Use Cases, Entities, Value Objects, Aggregates, Repositories, Services

### Web y protocolos

- HTTP, HTTPS, URL, URI, JSON, XML, YAML, HTML, CSS, REST, GraphQL, SSE, WebSocket
- API, Middleware, Middleware, Request, Response, Header, Body, Status Code
- GET, POST, PUT, PATCH, DELETE, OPTIONS, TRACE

### DevOps y infraestructura

- Docker, Kubernetes, Docker Compose, Kubernetes, Helm, Terraform, CloudFormation
- CI/CD, Jenkins, GitHub Actions, GitLab CI, Pipeline, Deploy, Release, Build
- AWS, Azure, GCP, EC2, S3, Lambda, CloudWatch, VPC, IAM, RBAC

### Monitoreo y observabilidad

- Metrics, Logs, Traces, APM, Monitoring, Alerting, Dashboard, SLA, SLO, SLI
- Circuit Breaker, Retry, Timeout, Rate Limit, Throttling, Caching, CDN

### Identificación y seguridad

- UUID, GUID, Token, JWT, OAuth, OAuth2, SSO, MFA, 2FA, Hash, Encrypt, Decrypt
- SSL, TLS, Certificate, CA, PKI, RSA, AES, HMAC, Signature, Verify

## 4. Abreviaturas y acrónimos que deben mantenerse en inglés

| Abreviatura | Significado |
|-------------|-------------|
| API | Application Programming Interface |
| URL | Uniform Resource Locator |
| URI | Uniform Resource Identifier |
| UUID | Universal Unique Identifier |
| HTTP | HyperText Transfer Protocol |
| HTTPS | HTTP Secure |
| JSON | JavaScript Object Notation |
| YAML | YAML Ain't Markup Language |
| XML | eXtensible Markup Language |
| HTML | HyperText Markup Language |
| CSS | Cascading Style Sheets |
| CPU | Central Processing Unit |
| RAM | Random Access Memory |
| GPU | Graphics Processing Unit |
| DB | Database |
| SQL | Structured Query Language |
| NoSQL | Not Only SQL |
| CDN | Content Delivery Network |
| IAM | Identity and Access Management |
| RBAC | Role-Based Access Control |
| ACL | Access Control List |
| JWT | JSON Web Token |
| OAuth | Open Authorization |
| TLS | Transport Layer Security |
| SSL | Secure Sockets Layer |
| SSE | Server-Sent Events |
| CIDR | Classless Inter-Domain Routing |

## 5. Convenciones específicas por componente

### Backend (Go)

- **Código**: Todo en inglés (nombres, comentarios, logs)
- **Documentación**: Specs y documentación técnica en español
- **Respuestas de API**: En inglés (cabeceras HTTP, JSON fields, RFC 9457 problem details)
- **Logs de producción**: En inglés (para compatibilidad con herramientas)

### Frontend web (React/TypeScript)

- **Código**: Todo en inglés (nombres, interfaces, componentes, hooks)
- **Mensajes de error al usuario**: En español
- **Documentación**: En español

### Aplicación móvil (React Native)

- **Código**: Todo en inglés (nombres, interfaces, componentes, hooks)
- **Mensajes de error al usuario (UI)**: En español
- **Documentación**: En español

### Infraestructura (Terraform)

- **Código**: Todo en inglés (nombres de recursos, variables, outputs)
- **Documentación**: En español
- **Logs**: En inglés

## 6. Ejemplos de mensajes de API en inglés

### Errores de validación (RFC 9457 Problem Details)

```http
HTTP/1.1 422 Unprocessable Content
Content-Type: application/problem+json
```

```json
{
  "type": "https://api.example.com/problems/validation-error",
  "title": "The request contains invalid data",
  "status": 422,
  "detail": "One or more fields do not meet the requirements.",
  "instance": "/problems/occurrences/abc123"
}
```

### Mensajes de respuesta estándar

- "The record has been created successfully"
- "No results were found for the search"
- "The authentication token has expired"
- "The rate limit for requests has been exceeded"

## 7. Convenciones de cabeceras HTTP y campos JSON

Las cabeceras HTTP y nombres de campos deben seguir estándares industriales (en inglés):

| Cabecera | Uso |
|----------|-----|
| `X-Trace-ID` | Identificador único para rastreo de requests |
| `X-Request-ID` | Identificador de request para auditoría |
| `X-RateLimit-Limit` | Límite de peticiones permitidas |
| `X-RateLimit-Remaining` | Peticiones restantes en el período |
| `X-RateLimit-Reset` | Timestamp de reinicio del límite |
| `Content-Type` | Tipo MIME de la respuesta (RFC estándar) |
| `Accept` | Tipos de contenido aceptados por el cliente |

### RFC 9457 Problem Details

Los campos JSON para respuestas de error deben mantenerse en inglés:

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `type` | URI | Identificador del tipo de problema |
| `title` | String | Resumen breve del problema |
| `status` | Integer | Código de estado HTTP |
| `detail` | String | Descripción específica de la ocurrencia |
| `instance` | URI | Referencia a esta ocurrencia específica |

**Nota**: Estos campos deben usarse en inglés, sin traducir a Spanish (ej. `type`, NO `tipo`; `title`, NO `título`).

## 8. Convenciones de logs

### Niveles de log (en inglés)

- `DEBUG`: Mensajes detallados para diagnóstico
- `INFO`: Mensajes de información general
- `WARN`: Mensajes de advertencia
- `ERROR`: Mensajes de error
- `FATAL`: Errores fatales

### Formato de mensajes (en inglés)

```
time="2024-01-15T10:30:00Z" level=info msg="request completed" trace_id=abc123 duration_ms=45
time="2024-01-15T10:30:01Z" level=error msg="failed to connect to database" error="connection refused"
```

## 9. Traducciones a evitar

### Términos técnicos generales

| Término en español | Término correcto (en inglés) | Nota |
|-------------------|------------------------------|------|
| Interfaz de programación de aplicaciones | API | Término técnico establecido |
| Lenguaje de consulta estructurado | SQL | Acrónimo universal |
| Mensaje de error | Error message | Mantener en inglés |
| Identificador único universal | UUID | Acrónimo universal |
| Llave primaria | Primary key | Término de base de datos |
| Llave foránea | Foreign key | Término de base de datos |
| Tabla | Table | Término de base de datos |
| Base de datos | Database | Término estándar |
| Servidor | Server | Término estándar |
| Cliente | Client | Término estándar |
| Petición | Request | Término HTTP estándar |
| Respuesta | Response | Término HTTP estándar |

### Términos de API y Problem Details (NO traducir)

| Término en español | Término correcto (en inglés) | Nota |
|-------------------|------------------------------|------|
| tipo | type | Campo RFC 9457 Problem Details |
| título | title | Campo RFC 9457 Problem Details |
| estatus | status | Campo RFC 9457 Problem Details |
| detalle | detail | Campo RFC 9457 Problem Details |
| instancia | instance | Campo RFC 9457 Problem Details |
| cabecera | header | Término HTTP |
| campo | field | Término JSON/HTTP |
| cuerpo | body | Término HTTP |
| estado | status | Código de estado HTTP |
| encabezado | header | Término HTTP |

## 10. Convenciones de nombres en código

### Go

```go
// Correcto
type Vehicle struct {
    ID        uuid.UUID
    LicensePLate string
    Status    VehicleStatus
    Coordinates []Coordinate
}

func (v *Vehicle) IsValid() bool {
    return v.ID != uuid.Nil && v.LicensePlate != ""
}

// Incorrecto
type Vehículo struct {
    ID        uuid.UUID
    Placa     string
    Estado    EstadoVehículo
    Coordenadas []Coordenada
}

func (v *Vehículo) EsVálido() bool {
    return v.ID != uuid.Nil && v.Placa != ""
}
```

### TypeScript/React

```typescript
// Correcto
interface Vehicle {
  id: string;
  licensePlate: string;
  status: VehicleStatus;
  coordinates: Coordinate[];
}

const VehicleList: React.FC<VehicleListProps> = ({ vehicles }) => {
  // ...
};

// Incorrecto
interface Vehículo {
  id: string;
  placa: string;
  estado: EstadoVehículo;
  coordenadas: Coordenada[];
}

const ListaVehículos: React.FC<ListaVehículosProps> = ({ vehículos }) => {
  // ...
};
```

## 11. Recomendaciones adicionales

1. **Consistencia**: Mantener el mismo término en todo el proyecto (no usar "vehículo" y "auto" indistintamente)
2. **Especificidad**: Preferir términos técnicos precisos sobre términos generales
3. **Estándares**: Seguir convenciones establecidas por los frameworks y tecnologías utilizadas
4. **Revisión**: Revisar regularmente el uso de términos en new PRs para mantener consistencia

## 12. Recursos relacionados

- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments) - Convenciones de nombres en Go
- [TypeScript Coding Guidelines](https://www.typescriptlang.org/docs/handbook/coding-guidelines.html) - Convenciones de nombres en TypeScript
- [AWS Naming Conventions](https://docs.aws.amazon.com/prescriptive-guidance/latest/cloud-native-applications/naming-conventions.html) - Convenciones para nombres en AWS