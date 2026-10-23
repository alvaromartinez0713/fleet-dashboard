# Fleet Dashboard - Portal Corporativo de Monitoreo de Flotas

Portal completo para captura móvil de GPS, procesamiento de telemetría en tiempo real y visualización web con agente de IA.

## 🚀 Inicio Rápido - App Móvil

```bash
# Instalar dependencias
cd mobile
npm install

# Ejecutar en emulador Android
make android

# O ejecutar en dispositivo real con Expo Go
npm start
# Escanee el QR con la cámara
```

## 📱 Aplicación Móvil - Funcionalidad

**Captura de GPS cada 30 segundos con sincronización offline-first**

Características implementadas:
- ✅ Captura automática de GPS cada 30 segundos
- ✅ Almacenamiento local en SQLite
- ✅ Funcionamiento offline (offline-first)
- ✅ Envío automático al backend
- ✅ Reintentos con exponential backoff
- ✅ Sincronización manual
- ✅ Indicador de conectividad
- ✅ Listado reactivo con estados
- ✅ Clean Architecture (dominio, aplicación, infraestructura, presentación)
- ✅ 250+ pruebas unitarias e integración

### Cómo usar en la app

1. **Iniciar Captura**: Presiona "▶ Iniciar"
   - Solicita permisos de ubicación
   - Comienza a capturar GPS automáticamente cada 30 segundos
   
2. **Ver Posiciones**: El listado se actualiza automáticamente
   - Muestra status de cada registro: Pendiente, Enviado, Error, Error de validación
   - Botón "Reintentar" para registros específicos (solo si está online)
   
3. **Sincronizar**: Presiona "↻ Sincronizar"
   - Reenvía todos los registros pendientes al backend
   - Disponible solo si está conectado
   
4. **Pausar**: Presiona "⏸ Pausar"
   - Detiene la captura (no pierden los datos locales)

### Estados de un Registro

| Status | Color | Descripción |
|--------|-------|-------------|
| pending | Azul | Esperando envío al backend |
| sent | Verde | Enviado exitosamente (202 Accepted) |
| error | Rojo | Error transitorio (se reintenta automáticamente) |
| validation_error | Naranja | Error permanente (no se reintenta) |

## 🏗 Arquitectura

### Backend (Go + Gin)
- API de ingesta HTTP
- Worker de procesamiento asíncrono
- Persistencia en TimescaleDB
- Mensajería con Kafka

### Frontend Web (React + TypeScript)
- Dashboard con mapa de vehículos
- Alertas en tiempo real (SSE)
- Chat integrado con agente de IA

### Aplicación Móvil (React Native + Expo)
- Captura de GPS con permisos (expo-location)
- Persistencia local con SQLite
- Sincronización por lotes con reintentos inteligentes
- Clean Architecture implementada

## 📋 Requisitos

Para la app móvil:
- Node.js 18+
- npm o yarn
- Android Studio (para emulador) u Expo Go en dispositivo
- Permisos de ubicación (solicitado por la app)

## 🧪 Pruebas

```bash
cd mobile

# Todas las pruebas
npm test

# Con cobertura
npm test -- --coverage

# Watch mode
npm test -- --watch

# Type checking
npm run type-check
```

## 📚 Documentación

- **`mobile/IMPLEMENTATION.md`** - Detalles técnicos de la arquitectura móvil y flujo de ejecución
- **`.kiro/specs/mobile-gps-capture/`** - Especificación completa con requisitos funcionales
- **`.kiro/steering/`** - Documentación de arquitectura, stack, testing, seguridad

## 🔧 Configuración Backend

Para conectar con un backend local, edita `mobile/src/infrastructure/api/RealApiClient.ts`:

```typescript
private baseURL = 'http://localhost:8080'; // Cambiar si es necesario
// En emulador Android: usar 'http://10.0.2.2:8080'
```

El backend debe estar en:
```
POST /api/positions
```

Respuestas esperadas:
- **202 Accepted** - Evento recibido (status = 'sent')
- **4xx / 422** - Error de validación (status = 'validation_error', no reintenta)
- **5xx** - Error transitorio (status = 'error', reintenta con exponential backoff)

## 🐛 Debugging

Ver logs en tiempo real:
```bash
npm start
```

Buscar en la consola mensajes con etiqueta `[App]`:
```
[App] Initializing dependencies...
[App] Capturing position...
[App] Sync complete
```

## 🚨 Solución de Problemas

### "Permiso de ubicación denegado"
- Android: Ajustes > Apps > Fleet Dashboard > Permissions > Location > Allow
- iOS: Settings > Privacy > Location > Fleet Dashboard > Allow

### "No se conecta al backend"
- Verificar que el backend está corriendo
- Revisar URL en `mobile/src/infrastructure/api/RealApiClient.ts`
- En emulador: usar `10.0.2.2` en lugar de `localhost`

---

**Estado del Proyecto:**
- ✅ App móvil: **FUNCIONAL** (captura GPS real, SQLite, sync automática, reintentos)
- ✅ 250+ pruebas unitarias e integración
- ✅ Clean Architecture implementada
- 🔄 Backend: En desarrollo
- 🔄 Frontend web: En desarrollo
