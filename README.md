# LinkMeQR

**Todo tu negocio en un QR.** Plataforma tipo Linktree para perfiles digitales personalizados, con un código QR permanente por negocio (`/p/:slug`) cuyo contenido se puede editar libremente sin reimprimir el QR.

- **Frontend**: Vue 3 + Vite + TypeScript, Tailwind CSS, Pinia, Vue Router.
- **Backend**: Go (API REST), chi router, sqlx.
- **Base de datos**: SQLite (un solo archivo, driver en Go puro — sin servidor de base de datos).
- **Infraestructura**: un único binario Go que sirve la API, el frontend compilado y los archivos subidos; servicio systemd detrás de Caddy (dominio + HTTPS automático). Sin Docker.

Ver [ARCHITECTURE.md](ARCHITECTURE.md) para el diseño completo (esquema de datos, contrato de API, algoritmo de licencias).

## Requisitos

- Para compilar (en tu máquina): Go 1.25+ y Node 20+.
- En el servidor: Linux con systemd y Caddy. **No necesita Go, Node, Docker ni MySQL.**

## Despliegue (servidor Linux + Caddy)

La app vive en `/opt/linkmeqr`:

```
/opt/linkmeqr/
  linkmeqr      binario (API + frontend + media; migraciones embebidas)
  seed          crea el admin inicial y las plantillas
  dist/         frontend compilado
  data/         linkmeqr.db (SQLite)
  media/        imágenes subidas
  backups/      copias de la base
  backup.sh     respaldo diario (cron)
  .env          configuración (ver .env.example)
```

### Primera vez

1. Sube la app y crea el `.env` en el servidor (`/opt/linkmeqr/.env`, a partir de `.env.example`). Como mínimo:
   - `JWT_SECRET` — `openssl rand -base64 48`.
   - `FRONTEND_ORIGIN` y `PUBLIC_BASE_URL` — el dominio real, ej. `https://linkmeqr.org` (sin slash final). `PUBLIC_BASE_URL` es lo que se codifica en cada QR, así que debe ser el dominio público definitivo.
   - `SEED_ADMIN_EMAIL` / `SEED_ADMIN_PASSWORD`.
2. Crea el admin y las plantillas: `cd /opt/linkmeqr && ./seed` (crea la base si no existe).
3. Instala el servicio, el permiso de reinicio para `deploy.sh` y el bloque de Caddy (única parte que requiere sudo). Con `install.sh`, `linkmeqr.service` y `Caddyfile.snippet` copiados a `/opt/linkmeqr`:
   ```bash
   sudo bash /opt/linkmeqr/install.sh
   ```
4. **Cloudflare** (el dominio va proxied): SSL/TLS → modo **Full (strict)**. Con "Flexible" el sitio entra en un bucle de redirecciones. Ver notas en `deploy/Caddyfile.snippet`.
5. Respaldo diario: `crontab -e` → `30 3 * * * /opt/linkmeqr/backup.sh >/dev/null 2>&1`.

### Actualizar

Desde la raíz del repo, en tu máquina (Git Bash en Windows):

```bash
bash deploy/deploy.sh
```

Compila el backend para Linux y el frontend, sube ambos, respalda la base antes de migrar, cambia los archivos (dejando `linkmeqr.anterior` y `dist.anterior` para volver atrás) y reinicia el servicio. Nunca toca `data/`, `media/` ni `.env`.

Logs: `journalctl -u linkmeqr -f`. Health check: `curl http://127.0.0.1:8090/healthz`.

### Base de datos

- Las migraciones están embebidas en el binario y se aplican solas al arrancar.
- Respaldo manual en caliente: `./linkmeqr -backup /ruta/copia.db` (usa `VACUUM INTO`, seguro con la app corriendo — no copies `linkmeqr.db` con `cp` mientras corre).
- Restaurar: detén el servicio, reemplaza `data/linkmeqr.db` por la copia, borra `data/linkmeqr.db-wal` y `-shm` si existen, y arranca.

## Flujo principal de uso

1. El **administrador** inicia sesión, crea un cliente (`Clientes → + Nuevo cliente`).
2. El administrador genera un **código de activación** (`Licencias → Generar código`, individual o por lote) con la duración deseada (1 mes, 3 meses, 6 meses, 1 año o personalizada).
3. El administrador crea/asigna el **perfil digital** del cliente (`Clientes → Ver perfil / licencia → Crear perfil`), definiendo el `slug` que usará su URL pública y QR.
4. El administrador entrega al cliente sus credenciales y el código de activación.
5. El **cliente** inicia sesión, va a `Licencia` e introduce su código de activación.
6. El cliente personaliza su perfil en `Editor de perfil` (bloques, colores, tipografía, plantilla) con vista previa en tiempo real.
7. La URL pública `/p/:slug` es permanente: cualquier QR que apunte a ella sigue funcionando aunque el contenido se edite.
8. Cualquier persona que escanee el QR llega a la página pública; si la licencia del cliente vence, la página pública muestra automáticamente un aviso de "perfil temporalmente inactivo" hasta que se reactive.

## Desarrollo local (backend y frontend corridos manualmente)

El backend Go y el frontend Vite se corren cada uno en su propia terminal — así ves logs, hot-reload, y puedes probar desde el celular en la misma red Wi-Fi. No hace falta instalar ninguna base de datos: SQLite crea `backend/data/linkmeqr.db` solo.

### 0. Configuración

```bash
cp .env.local.example .env
```

> Nota: `.env.local.example` trae valores de ejemplo ya listos para desarrollo (incluyendo tu IP LAN de referencia `192.168.103.139` en `FRONTEND_ORIGIN` / `PUBLIC_BASE_URL`). Ajusta esa IP a la tuya — revisa con `ipconfig` (Windows) / `ip addr` (Linux) la interfaz Wi-Fi/LAN.

### 1. Terminal A — Backend (Go)

```bash
cd backend
go run ./cmd/api
```

Lee las variables desde `../.env` (vía `godotenv`). Crea la base si no existe, aplica las migraciones automáticamente al arrancar y queda escuchando en `0.0.0.0:8080` (todas las interfaces, así que también responde en tu IP LAN).

La primera vez, en otra terminal, siembra el admin y las plantillas:
```bash
cd backend
go run ./seed
```

Verifica que responde: `curl http://localhost:8080/healthz` → `ok`.

### 2. Terminal B — Frontend (Vite)

```bash
cd frontend
npm install   # solo la primera vez
npm run dev
```

Vite arranca con `host: true`, por lo que además de `http://localhost:5173` queda accesible en tu IP LAN, por ejemplo `http://192.168.103.139:5173` — la terminal de Vite imprime ambas URLs (`Local:` y `Network:`) al iniciar.

### 3. Ver la app desde el celular

1. Conecta el teléfono a la **misma red Wi-Fi** que la computadora.
2. Abre en el navegador del teléfono la URL `Network:` que mostró Vite (algo como `http://192.168.103.139:5173`).
3. Las llamadas a `/api/...` que hace el frontend se resuelven vía el proxy interno de Vite hacia `http://localhost:8080` (en la misma máquina que corre Vite), así que no necesitas exponer el backend por separado ni cambiar nada más — solo asegúrate de que `FRONTEND_ORIGIN` en `.env` incluya esa misma URL LAN (ya viene incluida en `.env.local.example`), porque el backend valida CORS contra ese origen.
4. Si Windows Firewall bloquea la conexión entrante, permite el puerto 5173 (y 8080 si accedes a él directamente) para redes privadas cuando lo solicite, o agrega una regla manual:
   ```powershell
   New-NetFirewallRule -DisplayName "LinkMeQR Vite" -Direction Inbound -LocalPort 5173 -Protocol TCP -Action Allow
   ```

### Dar de alta un negocio de prueba

1. En el navegador (desktop o celular), entra a `/login` y accede como admin (`SEED_ADMIN_EMAIL` / `SEED_ADMIN_PASSWORD` de tu `.env`).
2. `Clientes → + Nuevo cliente` para crear el cliente de prueba.
3. `Licencias → Generar código` (1 mes, por ejemplo).
4. `Clientes → Ver perfil / licencia → Crear perfil` — define el `slug` (ej. `mi-negocio-test`).
5. Cierra sesión, entra como ese cliente, ve a `Licencia` y activa el código generado.
6. Ve a `Editar mi perfil` para personalizar bloques/colores/plantilla con vista previa en vivo.
7. Visita `http://<tu-ip-lan>:5173/p/mi-negocio-test` desde el celular para ver la página pública tal como la vería un cliente que escanea el QR.

## Estructura del proyecto

```
backend/    API REST en Go (ver ARCHITECTURE.md § Estructura de carpetas)
frontend/   SPA en Vue 3 + TypeScript
deploy/     Servicio systemd, bloque de Caddy, scripts de despliegue y respaldo
.env.example
ARCHITECTURE.md   Diseño técnico completo
```

## Licencias y activación de códigos (resumen)

- Cada código tiene una duración fija (1/3/6/12 meses o personalizada en días), estado (`UNUSED`/`USED`/`REVOKED`), y queda ligado al cliente que lo activa.
- Si el cliente **no tiene licencia activa** (o la suya ya venció), la nueva duración cuenta **desde la fecha de activación**.
- Si el cliente **ya tiene una licencia vigente**, la nueva duración se **suma a la fecha de vencimiento existente** (no la reemplaza).
- Cada activación queda registrada en el historial (`license_activations`) con: código usado, días agregados, vencimiento anterior y nuevo vencimiento — visible tanto en el panel del cliente como en el panel del administrador.

## Roadmap (preparado, no incluido en este MVP)

Dominios personalizados, NFC, catálogos y menús interactivos, formularios, promociones/cupones, integración con POS, pagos y suscripciones automáticas, facturación, white-label. El esquema de datos (`templates`, `media`, `profile_blocks.content` como JSON extensible) está diseñado para incorporar estas funcionalidades sin romper la estructura actual — ver [ARCHITECTURE.md § 11](ARCHITECTURE.md).
