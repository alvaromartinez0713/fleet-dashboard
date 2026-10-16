---
inclusion: always
---

# Estándares de seguridad de software

## 1. Propósito y alcance

Establecer principios y prácticas de seguridad aplicables durante el diseño, desarrollo, pruebas, despliegue y mantenimiento de aplicaciones de software.

Este documento define estándares generales reutilizables. Los requisitos específicos de cada producto se documentan en la sección final.

Las decisiones de seguridad deben considerar el contexto de la aplicación, los riesgos, los datos tratados, los usuarios, las integraciones y el entorno de despliegue.

## 2. Principios generales

- Aplicar seguridad desde el diseño y durante todo el ciclo de vida del software.
- Aplicar el principio de mínimo privilegio a usuarios, servicios, procesos y recursos.
- Denegar el acceso por defecto y conceder únicamente los permisos necesarios.
- Validar las entradas en los límites de confianza del sistema.
- Aplicar controles de seguridad en el servidor y no depender de mecanismos exclusivamente implementados en el cliente.
- Reducir la superficie de ataque y evitar funcionalidades, puertos, permisos y dependencias innecesarios.
- Separar responsabilidades para limitar el impacto de una vulnerabilidad.
- Utilizar configuraciones seguras por defecto.
- Mantener las dependencias, herramientas y componentes actualizados según su riesgo.
- Documentar los riesgos conocidos, las decisiones de seguridad y las excepciones justificadas.

## 3. Modelado de amenazas y análisis de riesgos

- Identificar activos, actores, límites de confianza, puntos de entrada y dependencias externas.
- Evaluar amenazas considerando la confidencialidad, integridad y disponibilidad.
- Analizar los riesgos de autenticación, autorización, exposición de datos, inyección, abuso de funcionalidades y denegación de servicio.
- Aplicar un nivel de protección proporcional al impacto y probabilidad de cada riesgo.
- Revisar el modelo de amenazas cuando cambien significativamente la arquitectura, las integraciones o los datos tratados.
- Registrar los riesgos que no puedan mitigarse y justificar su aceptación.

## 4. Autenticación y gestión de identidades

- Utilizar mecanismos de autenticación apropiados al tipo de aplicación y a sus requisitos.
- No implementar algoritmos criptográficos ni mecanismos de autenticación propios cuando existan alternativas estándar y mantenidas.
- Verificar la identidad en el servidor antes de permitir el acceso a operaciones protegidas.
- Proteger las credenciales durante su transmisión, almacenamiento, recuperación y renovación.
- Aplicar políticas adecuadas de expiración, revocación y renovación de sesiones o tokens.
- Evitar exponer credenciales, tokens de acceso, cookies de sesión y otros secretos en logs o mensajes de error.
- Aplicar medidas adicionales de autenticación para operaciones de riesgo cuando corresponda.

## 5. Autorización y control de acceso

- Verificar permisos en el servidor en cada operación protegida.
- Aplicar el principio de mínimo privilegio.
- Separar autenticación de autorización.
- No confiar en roles, identificadores de usuario, propietarios de recursos ni permisos proporcionados por el cliente sin validación.
- Evitar accesos horizontales o verticales no autorizados a recursos.
- Aplicar controles consistentes en todas las rutas y servicios que accedan a información protegida.
- Denegar la operación cuando no sea posible verificar de forma fiable los permisos.
- Probar tanto los accesos permitidos como los denegados.

## 6. Validación de entradas y tratamiento de datos

- Validar tipos, formatos, longitudes, rangos, valores permitidos y campos obligatorios.
- Preferir listas de valores permitidos cuando el dominio de entrada esté definido.
- Normalizar los datos cuando sea necesario y de forma consistente.
- Tratar como no confiables los datos recibidos de usuarios, servicios externos, archivos, colas de mensajes y bases de datos externas.
- Utilizar consultas parametrizadas y mecanismos seguros para acceder a bases de datos.
- Evitar concatenar entradas no confiables en SQL, comandos del sistema, expresiones ejecutables o consultas dinámicas.
- Aplicar codificación de salida contextual para prevenir la interpretación no deseada de datos.
- Validar los datos después de deserializarlos y antes de utilizarlos en operaciones sensibles.
- Establecer límites razonables para tamaños de solicitudes, colecciones, archivos y operaciones costosas.

## 7. Protección de datos y criptografía

- Clasificar los datos según su sensibilidad y los requisitos de protección aplicables.
- Recopilar y conservar únicamente los datos necesarios.
- Utilizar protocolos criptográficos y bibliotecas mantenidas y reconocidas.
- No diseñar algoritmos criptográficos propios.
- Utilizar TLS para proteger las comunicaciones cuando corresponda.
- Proteger los datos sensibles almacenados mediante controles apropiados al riesgo y al entorno.
- Gestionar las claves criptográficas de forma segura, separándolas de los datos que protegen.
- Definir políticas de retención y eliminación de datos.
- Evitar divulgar información sensible en respuestas de API, logs, métricas, trazas, informes de fallos o entornos de prueba.

## 8. Gestión de secretos y configuración

- No incluir secretos, contraseñas, tokens, claves privadas ni credenciales en el código fuente.
- No almacenar secretos en repositorios, imágenes de contenedores, documentación, archivos de ejemplo o paquetes cliente.
- Utilizar mecanismos apropiados de gestión de secretos y configuración por entorno.
- Mantener archivos de ejemplo libres de valores reales.
- Evitar que los secretos aparezcan en logs, argumentos de procesos o mensajes de error.
- Limitar el acceso a los secretos según las responsabilidades de cada componente.
- Establecer mecanismos de rotación y revocación de credenciales.
- Revisar las configuraciones para evitar servicios, permisos o interfaces de administración expuestos innecesariamente.

## 9. Manejo seguro de errores y registros

- Devolver errores controlados y adecuados al consumidor.
- No exponer trazas internas, consultas SQL, rutas del sistema, secretos ni detalles de infraestructura a clientes no confiables.
- Registrar eventos relevantes para diagnóstico y auditoría sin almacenar datos sensibles innecesarios.
- Utilizar identificadores de correlación cuando ayuden a investigar incidentes.
- Proteger los registros frente a accesos y modificaciones no autorizados.
- Evitar registrar tokens, contraseñas, claves privadas y payloads sensibles completos.
- Diferenciar los mensajes destinados al usuario de los detalles técnicos internos.
- No utilizar el contenido de un mensaje de error como sustituto de controles de autorización.

## 10. Seguridad de dependencias y cadena de suministro

- Preferir dependencias mantenidas y de procedencia confiable.
- Revisar vulnerabilidades conocidas y actualizar componentes según su criticidad.
- Mantener manifiestos y archivos de bloqueo de dependencias cuando el ecosistema lo permita.
- Evitar incorporar paquetes innecesarios o de origen dudoso.
- Revisar las acciones de CI/CD, imágenes base y herramientas externas utilizadas por el proyecto.
- Aplicar análisis de dependencias, detección de secretos y análisis estático cuando estén disponibles.
- Evaluar las alertas según su impacto real y documentar las excepciones justificadas.
- No considerar la ausencia de alertas automatizadas como garantía absoluta de seguridad.

## 11. Seguridad de aplicaciones web, móviles y servicios

- Aplicar las prácticas de seguridad pertinentes a cada plataforma y su modelo de amenazas.
- Proteger las comunicaciones entre clientes y servicios.
- No distribuir secretos privilegiados dentro de aplicaciones web o móviles.
- Configurar explícitamente los orígenes, permisos y capacidades que necesita cada aplicación.
- Mantener los controles de autorización en servicios confiables.
- Proteger las sesiones y los datos locales según las capacidades de cada plataforma.
- Considerar los riesgos de falsificación de solicitudes, ejecución de código, redirecciones no validadas, carga de archivos y exposición de recursos.
- No asumir que un cliente o dispositivo está bajo control exclusivo de la organización.

## 12. Seguridad de infraestructura y despliegue

- Aplicar mínimo privilegio a las identidades y permisos de infraestructura.
- Separar los entornos de desarrollo, pruebas y producción.
- Evitar credenciales de larga duración cuando existan alternativas seguras.
- Limitar la exposición de puertos, servicios y paneles de administración.
- Utilizar imágenes y configuraciones reproducibles.
- Revisar los cambios de infraestructura antes de aplicarlos.
- Proteger los secretos de los pipelines de integración y despliegue.
- Evitar que las tareas automatizadas puedan modificar entornos críticos sin los controles de autorización correspondientes.
- Mantener procedimientos de actualización, recuperación y respuesta a incidentes.

## 13. Pruebas y revisión de seguridad

- Incorporar la seguridad en los criterios de aceptación de las funcionalidades relevantes.
- Probar validaciones, autenticación, autorización y manejo de errores.
- Incluir pruebas de accesos no autorizados y entradas maliciosas.
- Realizar revisiones de código enfocadas en los riesgos de seguridad.
- Utilizar análisis estático, análisis de dependencias y otras herramientas adecuadas al contexto.
- Revisar los resultados de las herramientas en lugar de ignorarlos automáticamente.
- Documentar vulnerabilidades conocidas, medidas compensatorias y riesgos pendientes.
- No afirmar que una aplicación es segura únicamente porque sus pruebas automatizadas pasan.

## 14. Referencias generales

- [OWASP Application Security Verification Standard (ASVS)](https://owasp.org/www-project-application-security-verification-standard/)
- [OWASP Cheat Sheet Series](https://cheatsheetseries.owasp.org/)
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [CWE](https://cwe.mitre.org/)

Estas referencias sirven como guías para seleccionar controles aplicables; no implican que todos los requisitos de cada estándar deban adoptarse sin evaluar el contexto del proyecto.

## Lenguaje y estilo

Toda la documentación de este proyecto debe seguir las convenciones establecidas en `lenguaje.md`:

- **Documentación y specs:** escritos en español, con términos técnicos y acrónimos en inglés
- **Código:** nombres en inglés siguiendo convenciones de Go, TypeScript y React
- **Comentarios en código:** en inglés (estándar Go)
- **Mensajes de error al usuario:** en español
- **Logs de producción:** en inglés (compatibilidad con herramientas de monitoreo)

Ver el documento completo en `.kiro/steering/lenguaje.md`.
