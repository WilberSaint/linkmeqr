# LinkMeQR — Arquitectura y Plan de Construcción (MVP)

> Documento de referencia del proyecto. "Todo tu negocio en un QR."

## 1. Stack (obligatorio)

- **Frontend**: Vue 3 + Vite + TypeScript, Tailwind CSS, Pinia, Vue Router.
- **Backend**: Go, API REST.
- **DB**: MySQL 8 + phpMyAdmin.
- **Auth**: JWT (access + refresh), Argon2id, roles ADMIN/CLIENT.
- **Infra**: Docker Compose + Nginx (reverse proxy), pensado para un droplet de DigitalOcean.

## 2. Diagrama de arquitectura

```
                         ┌────────────────────┐
                         │       Nginx         │  :80 / :443
                         │  reverse proxy      │
                         └─────────┬───────────┘
              ┌──────────────────┼───────────────────┬───────────────┐
              │                  │                   │               │
        /  (estático)      /api/*  (proxy)      /phpmyadmin/*   (futuro: /p/* SSR opcional)
              │                  │                   │
    ┌─────────▼────────┐ ┌───────▼────────┐ ┌────────▼─────────┐
    │ frontend (Vue3)   │ │ backend (Go)    │ │ phpMyAdmin        │
    │ build estático     │ │ :8080 REST API  │ │ :80 (contenedor)  │
    │ servido por Nginx  │ │                 │ │                   │
    └────────────────────┘ └───────┬─────────┘ └─────────┬─────────┘
                                    │                     │
                              ┌─────▼─────────────────────▼─────┐
                              │            MySQL 8               │
                              │  volumen persistente `db_data`   │
                              └───────────────────────────────────┘
```

El frontend es una SPA compilada a estáticos sin SSR: la ruta pública `/p/:slug` se resuelve en el cliente vía Vue Router, y el backend expone `GET /api/public/profiles/:slug` sin autenticación. El QR siempre codifica `https://linkmeqr.com/p/:slug` — nunca cambia aunque el contenido del perfil se edite.

## 3. Estructura de carpetas

### Backend (Go) — `backend/`
```
backend/
  cmd/api/main.go                 # entrypoint, wiring de dependencias, arranque HTTP
  internal/
    config/                       # carga de .env / variables de entorno
    database/                     # conexión MySQL, migrator
    models/                       # structs de dominio (User, Profile, License, ActivationCode, ...)
    repository/                   # acceso a datos (SQL puro con database/sql + sqlx)
    services/                     # lógica de negocio (auth, licensing, profiles, qr, analytics)
    handlers/                     # HTTP handlers (chi router), por dominio
    middleware/                   # JWT auth, roles, rate limit, CORS, logging, recover
    validator/                    # validación de payloads (go-playground/validator)
    utils/                        # helpers (uuid, hashing, jwt, qr render, response envelope)
  migrations/                     # SQL versionado (0001_init.sql, ...)
  seed/                           # seed del admin inicial
  go.mod / go.sum
  Dockerfile
```

Decisiones de librerías Go:
- Router: `chi` (ligero, idiomático, buen soporte de middleware).
- DB: `database/sql` + driver `go-sql-driver/mysql`, con `sqlx` para queries ergonómicas (sin ORM pesado — control total sobre SQL, más fácil de razonar sobre el acumulado de licencias).
- Migraciones: `golang-migrate/migrate` (CLI + librería), archivos en `migrations/`.
- JWT: `golang-jwt/jwt/v5`.
- Password hashing: `golang.org/x/crypto/argon2` (Argon2id).
- Validación: `go-playground/validator/v10`.
- Rate limiting: middleware propio basado en `golang.org/x/time/rate` (in-memory, por IP+ruta, suficiente para MVP single-node).
- QR: `github.com/skip2/go-qrcode` como base para matriz QR (control de nivel de corrección de errores L/M/Q/H); render final de PNG/SVG con lógica propia de dibujo (permite personalizar módulos/ojos/logo y mantener quiet zone), usando `image/png` de la stdlib y generación manual de SVG (string building) para exportación vectorial.
- UUID: `google/uuid`.

### Frontend (Vue 3) — `frontend/`
```
frontend/
  src/
    main.ts
    App.vue
    router/
      index.ts                    # rutas públicas, cliente (guard CLIENT), admin (guard ADMIN)
    stores/                       # Pinia: auth.ts, profile.ts, license.ts, editor.ts
    api/                          # cliente axios + módulos por dominio (auth.ts, profiles.ts, ...)
    types/                        # tipos TS compartidos (User, Profile, Block, License, ...)
    composables/                  # useAuth, useAnalyticsTracker, useQrPreview, useDragReorder
    components/
      common/                     # UI genérica (Button, Modal, Input, Badge, Spinner)
      blocks/                     # un componente de render por tipo de bloque + BlockEditorForm
      editor/                     # BlockList (reorder), ThemeEditor, QrCustomizer, LivePreview
      admin/                      # tablas y forms de administración
      client/                     # widgets del panel cliente (LicenseStatusCard, etc.)
      public/                     # render de la página pública (ProfileHeader, BlockRenderer)
    views/
      public/ProfilePublicView.vue
      public/ProfileInactiveView.vue
      auth/LoginView.vue
      client/DashboardView.vue, ProfileEditorView.vue, LicenseView.vue
      admin/DashboardView.vue, ClientsView.vue, LicensesView.vue, ActivationCodesView.vue, TemplatesView.vue, StatsView.vue
  index.html, vite.config.ts, tailwind.config.js, tsconfig.json
  Dockerfile (multi-stage build → artefactos estáticos)
```

Librerías frontend:
- HTTP: `axios` con interceptor de refresh token.
- Drag & drop de bloques (MVP): `vuedraggable@next` (wrapper Vue3 de SortableJS) — simple y confiable, evita reinventar drag&drop.
- QR (vista previa en editor): renderizado vía llamada al backend (`GET /api/qr/preview`) que devuelve PNG/SVG ya compuesto, para que preview y export sean siempre idénticos — no se duplica lógica de dibujo en JS.
- Iconos: `lucide-vue-next` (set consistente para bloques sociales/contacto).

## 4. Esquema de base de datos

Ya creado en [migrations/0001_init.sql](backend/migrations/0001_init.sql): `users`, `refresh_tokens`, `templates`, `profiles`, `profile_themes`, `profile_blocks`, `media`, `licenses`, `activation_codes`, `license_activations`, `qr_codes`, `analytics_events`, `audit_logs`. UUIDs como `CHAR(36)` generados en Go (`google/uuid`). Índices en FKs y en columnas de filtrado frecuente (slug, status+expires_at, profile_id+created_at para analytics).

## 5. Lógica de licencias (núcleo del negocio)

Tabla `licenses`: una fila por usuario CLIENT, con `status` (`INACTIVE`/`ACTIVE`/`EXPIRED`) y `expires_at`.

**Algoritmo de activación de código** (transacción SQL):
```
func ActivateCode(userID, code string) error:
    BEGIN TRANSACTION
    ac := SELECT activation_codes WHERE code = ? FOR UPDATE
    if ac == nil or ac.status != 'UNUSED': return ErrInvalidCode

    license := SELECT licenses WHERE user_id = ? FOR UPDATE
    if license == nil:
        license = new License{user_id, status: INACTIVE, expires_at: NULL}
        INSERT license

    now := time.Now().UTC()
    previousExpiresAt := license.expires_at

    if license.status == 'ACTIVE' AND license.expires_at > now:
        // Acumular sobre la fecha de vencimiento vigente
        newExpiresAt := license.expires_at + ac.duration_days days
    else:
        // Sin licencia previa o vencida: cuenta desde hoy
        newExpiresAt := now + ac.duration_days days
        if license.activated_at == NULL:
            license.activated_at = now

    UPDATE licenses SET status='ACTIVE', expires_at=newExpiresAt, activated_at=COALESCE(activated_at, now)
    UPDATE activation_codes SET status='USED', used_by_user_id=userID, activated_at=now, expires_at=newExpiresAt

    INSERT INTO license_activations
        (license_id, activation_code_id, user_id, duration_days_added,
         previous_expires_at, new_expires_at, activated_at)
        VALUES (...)

    COMMIT
```

Puntos clave:
- `FOR UPDATE` en ambas filas evita condiciones de carrera si el usuario reintenta el submit.
- La condición de "vencida" es `expires_at <= now` (no solo el campo `status`, que se corrige de forma perezosa en este mismo flujo y también vía un job/consulta que marca `EXPIRED` cuando se lee el estado).
- `license_activations` guarda el historial completo pedido: código usado, días agregados, fecha anterior, fecha nueva.
- Un cron/goroutine ligero (o simplemente el cálculo en cada `GetLicenseStatus`) determina "activo" comparando `expires_at` contra `now`, así no depende de un job batch para el MVP.

**Página pública cuando la licencia expiró**: `GET /api/public/profiles/:slug` verifica el estado de licencia del dueño del perfil; si no está `ACTIVE` (según `expires_at > now`), devuelve un flag `inactive: true` y el frontend renderiza `ProfileInactiveView.vue` en lugar del perfil.

## 6. Auth

- `POST /api/auth/login` → valida credenciales (Argon2id), emite `access_token` (JWT, 15 min) + `refresh_token` (opaco, hash guardado en `refresh_tokens`, 30 días).
- `POST /api/auth/refresh` → valida refresh token contra su hash, rota el token (revoca el viejo, emite uno nuevo).
- `POST /api/auth/logout` → revoca el refresh token actual.
- Middleware `RequireAuth` parsea el JWT del header `Authorization: Bearer`; middleware `RequireRole(role)` verifica el claim `role`.
- Rate limiting más estricto en `/api/auth/login` (por IP) para mitigar fuerza bruta.
- CORS: origen configurable por env (`FRONTEND_ORIGIN`), credenciales habilitadas solo para ese origen.
- Toda mutación desde rutas `/api/admin/*` escribe una fila en `audit_logs` (acción, entidad, actor, metadata) vía un helper de servicio, no vía middleware genérico (para poder registrar metadata específica de cada acción).

## 7. Contrato de API (resumen por dominio)

**Auth** — público
- `POST /api/auth/login`
- `POST /api/auth/refresh`
- `POST /api/auth/logout`

**Cliente — perfil propio** (rol CLIENT)
- `GET /api/me` — datos de usuario + estado de licencia (status, expires_at, days_remaining)
- `GET /api/me/profile` · `PATCH /api/me/profile` (nombre, descripción, logo, template)
- `GET /api/me/theme` · `PATCH /api/me/theme`
- `GET /api/me/blocks` · `POST /api/me/blocks` · `PATCH /api/me/blocks/:id` · `DELETE /api/me/blocks/:id`
- `POST /api/me/blocks/:id/duplicate`
- `PATCH /api/me/blocks/reorder` (array de `{id, sort_order}`)
- `POST /api/me/license/activate` (body: `{code}`)
- `GET /api/me/license/history`
- `GET /api/me/qr` · `PATCH /api/me/qr` · `GET /api/me/qr/export?format=png|svg`
- `GET /api/me/stats/summary` · `GET /api/me/stats/timeseries?range=7d|30d`
- `POST /api/media/upload` (logo, fondo, imágenes de bloques)

**Admin** (rol ADMIN)
- `GET /api/admin/clients` · `POST /api/admin/clients` · `GET/PATCH /api/admin/clients/:id` · `POST /api/admin/clients/:id/activate` · `POST /api/admin/clients/:id/deactivate`
- `GET /api/admin/clients/:id/profile` (ver/asignar perfil de un cliente)
- `POST /api/admin/licenses/codes` (individual: `{duration_type, duration_days?}`)
- `POST /api/admin/licenses/codes/batch` (`{duration_type, quantity}`)
- `GET /api/admin/licenses/codes` (filtros: status, batch_id)
- `POST /api/admin/licenses/codes/:id/revoke`
- `GET /api/admin/licenses/:userId/history`
- `GET /api/admin/templates` · `POST/PATCH/DELETE /api/admin/templates/:id`
- `GET /api/admin/stats/overview` (totales de clientes, licencias activas/vencidas, visitas globales)
- `GET /api/admin/audit-logs` (filtros por entidad/actor)

**QR** (rol CLIENT, propio perfil)
- `GET /api/qr/preview` (query params de personalización → PNG/SVG on-the-fly, para el editor)
- Devuelve junto al binario/])SVG un header o endpoint hermano `GET /api/qr/validate` que responde `{ warnings: string[] }` cuando la combinación de colores/logo compromete el contraste o el error-correction disponible.

**Público** (sin auth)
- `GET /api/public/profiles/:slug` → perfil completo + bloques + tema, o `{inactive: true}`
- `POST /api/public/profiles/:slug/events` (`{type: VIEW|BLOCK_CLICK, block_id?, ...client hints}`)

## 8. Generador de QR — escaneabilidad garantizada

- Nivel de corrección de error mínimo `M`; se fuerza `Q` o `H` automáticamente cuando el usuario agrega un logo central (el logo puede tapar hasta ~30% con H).
- Quiet zone: siempre se reserva el margen mínimo de 4 módulos alrededor del código, no configurable por el usuario (se documenta como restricción, no como opción).
- El backend calcula el contraste entre `foreground_color` y `background_color` (fórmula de luminancia relativa); si el contraste es insuficiente devuelve warning y sugiere no continuar, pero permite forzar la descarga (decisión informada del usuario, como pide el enunciado: "mostrando advertencias").
- Export: `PNG` (raster con `image/png`) y `SVG` (paths generados en servidor) para impresión en alta resolución en tarjetas físicas.

## 9. Docker Compose

Servicios: `mysql`, `phpmyadmin`, `backend` (Go, expone 8080 internamente), `frontend` (build multi-stage, sirve estáticos vía su propio Nginx interno o copiado al Nginx principal), `nginx` (reverse proxy, único puerto expuesto 80/443).

Nginx rutea:
- `/` → estáticos del frontend
- `/api/` → `backend:8080`
- `/phpmyadmin/` → `phpmyadmin:80` (protegido, solo para admin de infraestructura, no para usuarios finales)

Variables de entorno vía `.env` (no versionado) + `.env.example` documentado: credenciales MySQL, `JWT_SECRET`, `JWT_REFRESH_SECRET`, `FRONTEND_ORIGIN`, `PUBLIC_BASE_URL` (para que el QR siempre apunte a `PUBLIC_BASE_URL/p/:slug`).

Volumen persistente `db_data` para MySQL; volumen `media_data` compartido por el backend para uploads.

## 10. Fases de construcción incremental

1. **Fase 0 — Scaffolding**: estructura de carpetas (ya creada), `go.mod`, Vite+Vue3+TS+Tailwind+Pinia+Router boilerplate, `.env.example`.
2. **Fase 1 — DB**: migraciones completas (ya creada `0001_init.sql`), seed del admin inicial, conexión y migrator en Go.
3. **Fase 2 — Auth**: modelos User, hashing Argon2id, JWT+refresh, middleware, endpoints login/refresh/logout, store `auth` en Pinia, vista de login, guards de router.
4. **Fase 3 — Admin core**: CRUD de clientes, generación de códigos (individual/batch), activación/desactivación, historial de licencias — endpoints + vistas admin.
5. **Fase 4 — Cliente: perfil y licencia**: activación de código desde el panel cliente, vista de estado de licencia (días restantes), CRUD de perfil básico + tema.
6. **Fase 5 — Bloques y editor visual**: CRUD+reorder+duplicate de bloques, editor con preview en tiempo real, plantillas predefinidas.
7. **Fase 6 — Página pública + QR**: render público por slug, página de "perfil inactivo", generador de QR (preview+export), tracking de eventos (`VIEW`/`BLOCK_CLICK`).
8. **Fase 7 — Analíticas**: agregación de eventos (totales, por día, 7/30 días, dispositivo/OS/browser) y vistas de stats en ambos paneles.
9. **Fase 8 — Docker/Nginx/Deploy**: Dockerfiles, docker-compose.yml, config Nginx, README de despliegue en Linux/DigitalOcean.

Cada fase se implementa y se verifica (compilación, endpoints probados con curl o similar, flujo en navegador) antes de pasar a la siguiente.

## 11. Extensibilidad futura (preparada, no implementada)

- `templates` desacoplado de `profiles` permite agregar plantillas sin tocar esquema.
- `profile_blocks.block_type` como ENUM ampliable + columna `content JSON` genérica permite agregar tipos (menú interactivo, catálogo, formularios, cupones) sin nueva tabla por tipo.
- `media` centralizado sirve para futuros catálogos/menús con múltiples imágenes.
- Estructura de `profiles` admite añadir `custom_domain VARCHAR` y `nfc_tag_id` más adelante sin romper nada.
- `licenses`/`activation_codes` ya modelan duración/estado de forma genérica — pasar a cobros recurrentes (Stripe/pagos) solo añade una tabla `subscriptions` que alimente `licenses` en vez de reemplazar el diseño.
