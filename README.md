# LinkMeQR

**Todo tu negocio en un QR.** Plataforma tipo Linktree para perfiles digitales personalizados, con un código QR permanente por negocio (`/p/:slug`) cuyo contenido se puede editar libremente sin reimprimir el QR.

- **Frontend**: Vue 3 + Vite + TypeScript, Tailwind CSS, Pinia, Vue Router.
- **Backend**: Go (API REST), chi router, sqlx.
- **Base de datos**: MySQL 8 + phpMyAdmin.
- **Infraestructura**: Docker Compose + Nginx (reverse proxy), pensado para un droplet de DigitalOcean.

Ver [ARCHITECTURE.md](ARCHITECTURE.md) para el diseño completo (esquema de datos, contrato de API, algoritmo de licencias, generación de QR).

## Requisitos

- Docker y Docker Compose (v2)
- Un dominio apuntando al droplet (opcional para desarrollo local)

## Despliegue rápido (Linux / DigitalOcean droplet)

1. **Clonar el repositorio en el servidor:**
   ```bash
   git clone <repo-url> linkmeqr
   cd linkmeqr
   ```

2. **Configurar variables de entorno:**
   ```bash
   cp .env.example .env
   nano .env
   ```
   Como mínimo, cambia:
   - `DB_PASSWORD`, `MYSQL_ROOT_PASSWORD` — contraseñas de MySQL.
   - `JWT_SECRET` — genera uno con `openssl rand -base64 48`.
   - `FRONTEND_ORIGIN` y `PUBLIC_BASE_URL` — el dominio real, ej. `https://linkmeqr.com` (sin slash final). `PUBLIC_BASE_URL` es la URL que se codifica dentro de cada QR (`{PUBLIC_BASE_URL}/p/{slug}`), así que debe ser el dominio público final.
   - `SEED_ADMIN_EMAIL`, `SEED_ADMIN_PASSWORD` — credenciales del administrador inicial.

3. **Levantar el stack completo:**
   ```bash
   docker compose -f docker-compose.prod.yml up -d --build
   ```
   Esto construye e inicia: `mysql`, `phpmyadmin`, `backend` (Go), `frontend` (Vue compilado, servido por Nginx interno), y `nginx` (reverse proxy en el puerto 80).

   > `docker-compose.yml` (sin `-f`) es el usado para desarrollo local — solo levanta `mysql` y `phpmyadmin`, ya que ahí el backend y frontend se corren manualmente en terminales separadas (ver "Desarrollo local" más abajo). En el servidor de producción usa siempre `-f docker-compose.prod.yml`.

   El backend aplica las migraciones de `backend/migrations/` automáticamente al arrancar.

4. **Crear el administrador inicial y las plantillas por defecto:**
   ```bash
   docker compose -f docker-compose.prod.yml exec backend ./seed
   ```
   Esto crea el usuario ADMIN con `SEED_ADMIN_EMAIL` / `SEED_ADMIN_PASSWORD` (si no existe ya) y las 7 plantillas predefinidas (Minimal, Business, Restaurant, Modern, Elegant, Dark, Colorful).

5. **Verificar:**
   - Frontend: `http://tu-dominio/` → pantalla de login.
   - API health check: `http://tu-dominio/healthz` → `ok`.
   - phpMyAdmin: `http://tu-dominio/phpmyadmin/` (restringir acceso a nivel de firewall/VPN en producción; no está pensado para exponerse públicamente).

6. **HTTPS (recomendado en producción):** coloca un proxy TLS delante (Certbot + Nginx, o un load balancer de DigitalOcean) apuntando al puerto 80 del contenedor `nginx`, o añade un `server { listen 443 ssl; ... }` a `nginx/nginx.conf` con tus certificados montados como volumen.

## Flujo principal de uso

1. El **administrador** inicia sesión, crea un cliente (`Clientes → + Nuevo cliente`).
2. El administrador genera un **código de activación** (`Licencias → Generar código`, individual o por lote) con la duración deseada (1 mes, 3 meses, 6 meses, 1 año o personalizada).
3. El administrador crea/asigna el **perfil digital** del cliente (`Clientes → Ver perfil / licencia → Crear perfil`), definiendo el `slug` que usará su URL pública y QR.
4. El administrador entrega al cliente sus credenciales y el código de activación (tarjeta física con el QR se genera después desde el panel del cliente).
5. El **cliente** inicia sesión, va a `Licencia` e introduce su código de activación.
6. El cliente personaliza su perfil en `Editor de perfil` (bloques, colores, tipografía, plantilla) con vista previa en tiempo real.
7. El cliente genera y descarga su **QR personalizado** en `Código QR` (PNG o SVG) para imprimir en su tarjeta física — este QR siempre apunta a `/p/:slug` y no cambia aunque el contenido se edite.
8. Cualquier persona que escanee el QR llega a la página pública; si la licencia del cliente vence, la página pública muestra automáticamente un aviso de "perfil temporalmente inactivo" hasta que se reactive.

## Desarrollo local (backend y frontend corridos manualmente)

En este modo, Docker solo levanta la infraestructura (MySQL + phpMyAdmin). El backend Go y el frontend Vite se corren cada uno en su propia terminal — así ves logs, hot-reload, y puedes probar desde el celular en la misma red Wi-Fi.

### 0. Levantar solo MySQL + phpMyAdmin

```bash
cp .env.local.example .env
docker compose up -d
```

Esto expone MySQL en `localhost:3306` y phpMyAdmin en `http://localhost:8081`.

> Nota: `.env.local.example` trae valores de ejemplo ya listos para desarrollo (incluyendo tu IP LAN de referencia `192.168.103.139` en `FRONTEND_ORIGIN` / `PUBLIC_BASE_URL`). Ajusta esa IP a la tuya — la terminal del backend la imprime al arrancar, o revisa con `ipconfig` (Windows) / `ip addr` (Linux) la interfaz Wi-Fi/LAN.

### 1. Terminal A — Backend (Go)

```bash
cd backend
go run ./cmd/api
```

Lee las variables desde `../.env` (vía `godotenv`). Aplica las migraciones automáticamente al arrancar y queda escuchando en `0.0.0.0:8080` (todas las interfaces, así que también responde en tu IP LAN).

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
6. Ve a `Editar mi perfil` para personalizar bloques/colores/plantilla con vista previa en vivo, y a `Código QR` para generar y descargar el QR.
7. Visita `http://<tu-ip-lan>:5173/p/mi-negocio-test` desde el celular para ver la página pública tal como la vería un cliente que escanea el QR.

### Producción (Docker Compose completo)

El archivo `docker-compose.prod.yml` incluye backend, frontend y Nginx además de MySQL/phpMyAdmin — es el que se usa para el despliegue real en el droplet de DigitalOcean (ver sección "Despliegue rápido" arriba), corriendo `docker compose -f docker-compose.prod.yml up -d --build` en vez de `docker compose up -d`.

## Estructura del proyecto

```
backend/    API REST en Go (ver ARCHITECTURE.md § Estructura de carpetas)
frontend/   SPA en Vue 3 + TypeScript
nginx/      Configuración del reverse proxy principal
docker-compose.yml
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
