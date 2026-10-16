---
inclusion: always
---

# Estándares de diseño e implementación de API

## 1. Propósito y alcance

Definir estándares generales para diseñar, implementar, documentar, probar y mantener API, especialmente API HTTP basadas en REST.

Los contratos específicos de cada producto deben documentarse en la sección final o en la Spec correspondiente.

## 2. Principios generales

- Diseñar contratos explícitos, consistentes y orientados a las necesidades de los consumidores.
- Separar el contrato HTTP de los modelos internos del dominio y de persistencia.
- Mantener consistentes las convenciones de nombres, validación, respuestas, errores y versionado.
- Utilizar los métodos y códigos HTTP según su semántica estándar.
- Preferir cambios compatibles con los clientes existentes.
- Validar las entradas y aplicar autorización en el servidor.
- Documentar los comportamientos relevantes y sus condiciones de fallo.
- Evitar contratos ambiguos, comportamientos implícitos y respuestas que prometan más de lo que el sistema garantiza.

## 3. Diseño de recursos y rutas

- Utilizar sustantivos para identificar recursos.
- Mantener una convención consistente para los nombres y la pluralización de recursos.
- Utilizar rutas predecibles y evitar estructuras excesivamente profundas.
- Utilizar parámetros de ruta para identificar recursos y parámetros de consulta para filtros, paginación y ordenamiento.
- Definir formatos estables para los identificadores.
- Evitar incluir detalles internos de la base de datos o de la arquitectura en las rutas públicas.
- Utilizar una estrategia de versionado explícita cuando el ciclo de vida del producto lo requiera.

## 4. Semántica de métodos HTTP

- Utilizar `GET` para recuperar representaciones sin producir cambios de estado intencionales.
- Utilizar `POST` para crear recursos, enviar comandos o presentar información para procesamiento.
- Utilizar `PUT` para reemplazar una representación cuando corresponda.
- Utilizar `PATCH` para modificar parcialmente un recurso cuando se haya definido esa semántica.
- Utilizar `DELETE` para eliminar recursos cuando la operación esté soportada y autorizada.
- Respetar las propiedades de seguridad e idempotencia definidas por la semántica HTTP.
- No utilizar un método HTTP únicamente por conveniencia si contradice el comportamiento real de la operación.
- Documentar las particularidades de las operaciones que no sigan el patrón habitual de recursos.

## 5. Códigos de estado HTTP

Seleccionar el código que describa correctamente el resultado de la solicitud.

- `200 OK`: solicitud procesada correctamente y se devuelve una representación.
- `201 Created`: se creó un recurso.
- `202 Accepted`: solicitud aceptada para procesamiento, que todavía puede estar pendiente.
- `204 No Content`: operación exitosa sin cuerpo de respuesta.
- `400 Bad Request`: solicitud inválida o malformada.
- `401 Unauthorized`: no se dispone de autenticación válida.
- `403 Forbidden`: la operación no está permitida para el solicitante.
- `404 Not Found`: recurso no encontrado o no disponible según la política de exposición de la API.
- `405 Method Not Allowed`: método no admitido para el recurso.
- `409 Conflict`: conflicto con el estado actual del recurso.
- `412 Precondition Failed`: no se cumple una precondición de la solicitud.
- `413 Content Too Large`: contenido superior al límite permitido.
- `415 Unsupported Media Type`: tipo de contenido no admitido.
- `422 Unprocessable Content`: contenido sintácticamente válido que no cumple las reglas semánticas adoptadas por la API.
- `429 Too Many Requests`: se excedió un límite de solicitudes.
- `500 Internal Server Error`: fallo inesperado del servidor.
- `503 Service Unavailable`: servicio temporalmente no disponible.

No devolver un código de éxito cuando la operación haya fallado. Tampoco utilizar códigos de error únicamente porque resulten cómodos para la implementación.

## 6. Estándar de errores: Problem Details

### Norma adoptada

Las respuestas de error HTTP deben seguir **RFC 9457 — Problem Details for HTTP APIs**, utilizando el tipo de contenido `application/problem+json` para la representación JSON.

[Referencia normativa](https://www.rfc-editor.org/rfc/rfc9457.html)

### Estructura

- Utilizar los miembros estándar definidos por RFC 9457 cuando sean aplicables:
  - `type`: URI que identifica el tipo de problema.
  - `title`: resumen breve y estable del tipo de problema.
  - `status`: código de estado HTTP asociado al problema.
  - `detail`: explicación específica de esta ocurrencia, destinada a ayudar al consumidor.
  - `instance`: URI de referencia que identifica esta ocurrencia concreta del problema.
- Los miembros opcionales deben omitirse cuando no aporten información útil.
- Las extensiones específicas pueden añadirse cuando exista una necesidad concreta y documentada.
- No inventar formatos alternativos de error para cada endpoint.
- No utilizar una estructura genérica propia como sustituto de Problem Details.

### Semántica y seguridad

- El código HTTP real de la respuesta debe ser coherente con el miembro `status` del documento.
- El miembro `type` debe identificar el tipo de problema, no la ocurrencia individual.
- Utilizar `about:blank` cuando no sea necesario definir un tipo de problema más específico.
- Los miembros `title` y `detail` deben ser comprensibles y no revelar información sensible.
- No incluir trazas de ejecución, consultas SQL, secretos, rutas internas ni detalles que faciliten un ataque.
- No exigir a los clientes que analicen el texto de `detail` para tomar decisiones programáticas; utilizar tipos de problema y extensiones documentadas.
- Definir tipos de problema específicos únicamente cuando aporten semántica útil y estable.
- Si se definen tipos propios, documentar su significado, el estado HTTP aplicable y las acciones que puede realizar el consumidor.

### Extensiones

- Añadir extensiones solo cuando aporten valor al consumidor de la API.
- Documentar el nombre, tipo y semántica de cada extensión.
- Evitar duplicar información que ya esté definida por los miembros estándar.
- Utilizar una extensión estructurada para errores de validación cuando el contrato necesite devolver varios errores por campo.
- No introducir extensiones innecesarias como `code`, `errors` o `traceId` por defecto; justificar cada una en el contrato de la API.
- Mantener las extensiones estables y compatibles entre endpoints.

### Ejemplo ilustrativo

```http
HTTP/1.1 422 Unprocessable Content
Content-Type: application/problem+json
```

```json
{
  "type": "https://api.example.com/problems/validation-error",
  "title": "La solicitud contiene datos inválidos",
  "status": 422,
  "detail": "Uno o más campos no cumplen los requisitos.",
  "instance": "/problems/occurrences/abc123"
}
```

El dominio y la ruta del ejemplo son ilustrativos. Los valores definitivos deben corresponder al producto y al diseño de su API.

## 7. Validación y contratos de entrada

- Validar los campos obligatorios, tipos, formatos, rangos, longitudes y valores admitidos.
- Definir el comportamiento ante campos desconocidos.
- Establecer límites para el tamaño de las solicitudes, colecciones y lotes.
- Validar fechas, timestamps, zonas horarias, identificadores y unidades de medida.
- Aplicar validaciones de negocio en la capa correspondiente, sin depender únicamente de validaciones de transporte.
- No confiar en la validación del cliente como mecanismo de seguridad.
- Definir claramente si una solicitud se acepta o rechaza como una unidad o si admite resultados parciales.
- Documentar los errores de validación y utilizar Problem Details para las respuestas de error HTTP.

## 8. Paginación, filtros y ordenamiento

- Definir una estrategia de paginación coherente para los endpoints que devuelvan colecciones.
- Establecer tamaños predeterminados y máximos de página.
- Documentar los parámetros admitidos, los valores predeterminados y el orden de los resultados.
- Validar los campos permitidos para filtrar y ordenar.
- No permitir que los parámetros se conviertan en expresiones de consulta ejecutables sin validación.
- Considerar paginación por cursor cuando el volumen de datos o las modificaciones concurrentes lo justifiquen.
- Definir cómo se representa el final de una colección y cómo se continúa la consulta.

## 9. Idempotencia, concurrencia y reintentos

- Definir la idempotencia según la semántica HTTP y las necesidades de cada operación.
- Para operaciones que puedan repetirse por reintentos, documentar qué ocurre con solicitudes duplicadas.
- Utilizar identificadores de idempotencia cuando el contrato y la naturaleza de la operación lo requieran.
- Definir el alcance, la duración y el comportamiento de los identificadores de idempotencia.
- No prometer ejecución exactamente una vez si la arquitectura no puede garantizarla.
- Distinguir los errores transitorios de los errores permanentes que no deben reintentarse.
- Utilizar mecanismos de concurrencia condicional cuando sea necesario prevenir actualizaciones incompatibles.

## 10. Operaciones asíncronas

- Diferenciar la aceptación de una operación de su finalización.
- Utilizar `202 Accepted` cuando la solicitud haya sido aceptada para procesamiento asíncrono y este no haya finalizado.
- Definir cómo se comunica al consumidor el estado final cuando el flujo lo requiera.
- Proporcionar un identificador de seguimiento o un recurso de estado cuando esté justificado por el contrato.
- Documentar los estados, errores, tiempos de espera y condiciones de reintento.
- No declarar que una operación terminó correctamente si solo se confirmó su recepción o publicación en un sistema de mensajería.
- No imponer un mecanismo universal de seguimiento asíncrono cuando la funcionalidad no lo necesite.

## 11. Seguridad y resiliencia

- Aplicar autenticación y autorización según los requisitos de cada endpoint.
- Limitar el tamaño de solicitudes y las operaciones costosas.
- Configurar límites de tasa cuando corresponda.
- Validar entradas y utilizar acceso seguro a bases de datos.
- No exponer información sensible en respuestas ni en errores.
- Establecer tiempos de espera y políticas de reintento adecuados a las dependencias.
- Evitar reintentos automáticos de operaciones no idempotentes cuando no exista protección frente a duplicados.
- Mantener una separación clara entre errores de validación, autorización, negocio y fallos internos.

## 12. Versionado y compatibilidad

- Definir una estrategia de versionado coherente con el ciclo de vida de la API.
- Evitar cambios incompatibles sin una estrategia de migración.
- Preferir cambios aditivos cuando sean compatibles con el contrato.
- Documentar los cambios que afecten a los clientes.
- Definir un proceso para retirar versiones obsoletas cuando sea necesario.
- Mantener alineados el contrato, la implementación, las pruebas y la documentación.
- No incrementar la versión por cambios internos que no alteren el contrato público.

## 13. Documentación y pruebas de contrato

- Documentar rutas, métodos, parámetros, esquemas, códigos de estado, errores y requisitos de seguridad.
- Utilizar OpenAPI cuando el proyecto lo adopte.
- Representar las respuestas de error con el esquema de Problem Details y el tipo de contenido correspondiente.
- Probar contratos exitosos y fallidos.
- Verificar que los consumidores manejen correctamente los códigos HTTP y los tipos de problema.
- Mantener ejemplos coherentes con la implementación real.
- Evitar que la documentación prometa comportamientos que no estén implementados o probados.

## 14. Referencias generales

- [RFC 9457 — Problem Details for HTTP APIs](https://www.rfc-editor.org/rfc/rfc9457.html)
- [RFC 9110 — HTTP Semantics](https://www.rfc-editor.org/rfc/rfc9110.html)
- [OpenAPI Specification](https://spec.openapis.org/oas/latest.html)
- [OWASP REST Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/REST_Security_Cheat_Sheet.html)

## Lenguaje y estilo

Toda la documentación de este proyecto debe seguir las convenciones establecidas en `lenguaje.md`:

- **Documentación y specs:** escritos en español, con términos técnicos y acrónimos en inglés
- **Código:** nombres en inglés siguiendo convenciones de Go, TypeScript y React
- **Comentarios en código:** en inglés (estándar Go)
- **Mensajes de error al usuario:** en español
- **Logs de producción:** en inglés (compatibilidad con herramientas de monitoreo)

Ver el documento completo en `.kiro/steering/lenguaje.md`.
