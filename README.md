# Backend API — Comparación de Productos (Mercado Libre Challenge)

API REST en **Go + Gin** para servir detalles de productos y comparar items con proyección selectiva de campos.

> Implementado con **Clean Architecture liviana** y **TDD estricto**.

📌 Para preparar defensa técnica de entrevista: ver [`INTERVIEW-NARRATIVE.md`](./INTERVIEW-NARRATIVE.md).

## Stack

| Componente | Elección | Por qué |
|---|---|---|
| Lenguaje | Go 1.21+ | Tipado fuerte, testing nativo, compilación rápida, alineado con stack MELI |
| Framework HTTP | Gin | Routing simple, middleware estándar, performance |
| Persistencia | JSON file in-memory | Sin drivers, portable, fixtures fáciles |
| Testing | `testing` stdlib + `httptest` | Sin dependencias de mocking, idiomático Go |
| Logs | `log/slog` (stdlib) | JSON estructurado con request_id, sin dependencias |
| Lifecycle | `os/signal` + `http.Server.Shutdown` | Graceful shutdown ante SIGTERM/SIGINT |

## Arquitectura

Tres capas con la regla de dependencias de Clean Architecture: las capas externas conocen a las internas, **nunca al revés**.

```
                      ┌─────────────────────────────────┐
                      │      cmd/api (composition)      │
                      └────────────────┬────────────────┘
                                       │ wires
            ┌──────────────────────────┴──────────────────────────┐
            │                                                     │
┌───────────▼─────────────┐                       ┌───────────────▼──────────────┐
│  infrastructure/http    │                       │ infrastructure/persistence   │
│  (handlers Gin, DTOs)   │                       │ (JSONRepository)             │
└───────────┬─────────────┘                       └───────────────┬──────────────┘
            │ invoca                                              │ implementa
            ▼                                                     │
┌─────────────────────────┐                                       │
│      application        │                                       │
│  (use cases)            │◀──── usa puerto ─────────────────────┐│
└────────────┬────────────┘                                      ││
             │                                                   ││
             ▼                                                   │▼
┌──────────────────────────────────────────────────────────────────────┐
│                              domain                                  │
│   Product · errors · ProductRepository (port)                        │
│   SIN dependencias externas — solo stdlib                            │
└──────────────────────────────────────────────────────────────────────┘
```

**Regla de oro**: el dominio no sabe que existe Gin, JSON ni HTTP. Eso permite reemplazar cualquier capa externa sin tocar la lógica de negocio.

### Estructura de directorios

```
.
├── cmd/api/main.go                          # Composition root: wire dependencies
├── internal/
│   ├── domain/
│   │   ├── product.go                       # Entity Product + SelectFields
│   │   ├── errors.go                        # Errores tipados
│   │   ├── repository.go                    # Port ProductRepository
│   │   └── product_test.go
│   ├── application/
│   │   ├── compare_products.go              # Use case principal
│   │   ├── list_products.go
│   │   ├── get_product.go
│   │   ├── list_categories.go
│   │   ├── mock_repository_test.go          # Test double del port
│   │   └── *_test.go
│   └── infrastructure/
│       ├── persistence/
│       │   ├── json_repository.go           # Adapter del port (JSON in-memory)
│       │   └── json_repository_test.go
│       └── http/
│           ├── router.go                    # Setup Gin + rutas
│           ├── product_handler.go
│           ├── error_handler.go             # Mapeo error dominio → HTTP status
│           ├── dto.go                       # Response shapes
│           └── product_handler_test.go      # Tests httptest e2e
├── testdata/products.json                   # Catálogo de prueba (8 productos)
├── problem.md                               # Enunciado del challenge (español)
├── HACKERRANK.md                            # Guía paso a paso para portar a HackerRank
├── README.md                                # Este archivo
├── go.mod
└── go.sum
```

## Endpoints

| Método | Path | Descripción |
|---|---|---|
| GET | `/health` | Healthcheck — `{"status":"ok"}` |
| GET | `/products?page=1&size=20` | Lista paginada (size cap=100) |
| GET | `/products/categories` | Categorías distintas, ordenadas |
| GET | `/products/:id` | Detalle de un producto |
| GET | `/products/compare?ids=1,2,3&fields=name,price,rating` | **Compara N productos con proyección selectiva**. Sin `fields` retorna todo. |

### Errores

| Status | Cuándo |
|---|---|
| 400 | `ids` ausente, `fields` fuera de la whitelist, `page`/`size` inválidos, `ids` excede `MaxCompareIDs=50` |
| 404 | Producto no existe. En `/compare` el body incluye `"missing_ids": [...]` con detalle |
| 500 | Error interno (genérico — nunca filtra detalles) |

### Schemas de respuesta

**`GET /health`** → `200`
```json
{ "status": "ok" }
```

**`GET /products?page=N&size=M`** → `200`
```json
{
  "items":  [Product, ...],
  "total":  8,
  "page":   1,
  "size":   20
}
```

**`GET /products/:id`** → `200`
```json
{ "id": "1", "name": "...", "price": 1299.99, "specs": { ... }, ... }
```

**`GET /products/categories`** → `200`
```json
{ "categories": ["appliances", "books", "clothing", "smartphones"] }
```

**`GET /products/compare?ids=1,2,3&fields=name,price`** → `200`
```json
{
  "items":  [{ "name": "...", "price": 1299.99 }, ...],
  "fields": ["name", "price"]
}
```

**Errores** → `4xx/5xx`
```json
{ "error": "...", "missing_ids": ["99"] }
```
`missing_ids` solo aparece en 404 de `/compare` con IDs faltantes.

### Detalles de comportamiento

- **`/compare` deduplica IDs en el use case**: `?ids=1,1,2` → `[1, 2]`. Cualquier adapter (HTTP, gRPC, CLI) recibe la misma semántica.
- **`/compare` cap de cantidad**: máximo `MaxCompareIDs=50` items por request. Defensa contra DoS por amplificación + guard de UX.
- **`/products?page=999`** (fuera de rango): `200` con `items: []` y `total` real — el cliente sabe que ya iteró.
- **`/products?page=-1`**: `400` (validación strict). Si NO se pasa el query param, se aplica el default silenciosamente.
- **`X-Request-Id`**: si viene en el request lo respetamos (correlación cross-service); si no, generamos uno y lo devolvemos en el response header. Va también en cada log JSON.

#### Decisión: 404 parcial vs 200 con missing
Si pedís `?ids=1,2,99` y `99` no existe, el endpoint **falla con 404** y body:
```json
{"error":"products not found: 99","missing_ids":["99"]}
```
**Por qué**: comparar implica que TODOS los items son relevantes. Devolver 200 con N-1 productos engaña al cliente — nunca sabría qué falta. Trade-off aceptado: el cliente debe re-armar la query.

## Modelo de datos

```go
type Product struct {
    ID, Name, Description, ImageURL, Category string
    Price, Rating, Weight                     float64
    Size, Color                               string         // opcionales
    Specs                                     map[string]any // flexible por categoría
}
```

`Specs` es un mapa flexible: un smartphone tiene `{ram, battery, camera, ...}`, un libro tiene `{author, isbn, pages, ...}`, una prenda tiene `{material, care, ...}`. Trade-off explícito: pierde type-safety en compile time pero permite agregar categorías sin cambiar el schema. Para un catálogo real con miles de SKUs heterogéneos, la flexibilidad gana.

## Setup

### Requisitos
- Go 1.21 o superior

### Correr el servicio

```bash
# Tests
go test ./... -cover

# Build
go build -o api ./cmd/api

# Run (default :8080, testdata/products.json)
./api

# Run con configuración
PORT=3000 PRODUCTS_FILE=/path/to/custom.json ./api
```

### Comandos via Makefile

```bash
make run          # corre el server en :8080
make build        # binario en bin/api
make test         # corre todos los tests
make cover        # coverage agregado por paquete
make cover-html   # genera coverage.html navegable
make vet          # go vet
make fmt          # gofmt -w
make vulncheck    # govulncheck (instala si falta)
make check        # fmt + vet + test (pre-commit)
```

### Cobertura manual

```bash
go test ./... -cover                                       # totales por paquete
go test ./... -coverprofile=cover.out                      # detalle por línea
go tool cover -html=cover.out                              # visualización en browser
govulncheck ./...                                          # CVEs en deps directas/transitives
```

### Smoke test (con respuestas esperadas)

```bash
$ curl -s localhost:8080/health
{"status":"ok"}

$ curl -s "localhost:8080/products/compare?ids=1,2&fields=name,price,rating"
{"items":[
  {"name":"iPhone 15 Pro","price":1299.99,"rating":4.7},
  {"name":"Samsung Galaxy S24 Ultra","price":1399.99,"rating":4.6}
],"fields":["name","price","rating"]}

$ curl -sw "\nHTTP %{http_code}\n" "localhost:8080/products/compare?ids=1,99"
{"error":"products not found: 99","missing_ids":["99"]}
HTTP 404

$ curl -sw "\nHTTP %{http_code}\n" "localhost:8080/products/compare"
{"error":"at least one id is required"}
HTTP 400

$ curl -sw "\nHTTP %{http_code}\n" "localhost:8080/products?page=-1"
{"error":"invalid pagination parameters: page=\"-1\""}
HTTP 400
```

## Decisiones arquitectónicas

| Decisión | Trade-off aceptado |
|---|---|
| Clean Architecture liviana | Más archivos que un layered simple, pero permite testear cada capa aislada |
| Port `ProductRepository` en `domain` | Inversión de dependencias: domain define lo que necesita, infra implementa |
| `Specs` como `map[string]any` | Flexibilidad > type-safety en specs |
| Whitelist de campos (no blacklist) | Seguridad por defecto: campos internos nuevos no se filtran |
| 404 todo-o-nada en `/compare` | Cliente sabe exactamente qué falta vs comparación silenciosamente parcial |
| Cap de `size=100` en paginación | Defensivo contra DoS por `size=999999` |
| JSON in-memory cargado al boot | Sin I/O por request, trade-off: cambios al archivo requieren restart |
| Mock manual del repo en tests | Sin dependencias de gomock, más explícito para 4 use cases |
| DI manual (sin Wire/Fx) | 10 líneas en `main.go`, se lee mejor que generación de código |

## Cobertura de tests

**47 tests** en 5 paquetes:

```
domain          → 6 tests   (entity + errors)
application     → 16 tests  (use cases con mocks, incluye dedup y cap de IDs)
persistence     → 11 tests  (adapter JSON con archivos de prueba)
http            → 14 tests  (e2e vía httptest, paginación strict, dedup, page out-of-range)
```

Cada error path está cubierto: empty IDs, invalid field, too many IDs (>50), missing IDs, not found, paginación inválida, IDs duplicados, page fuera de rango.

## Robustez de runtime (incluida)

| Ítem | Implementación |
|---|---|
| **Graceful shutdown** | `signal.Notify(SIGINT, SIGTERM)` + `srv.Shutdown(ctx)` con timeout 10s — drena requests en vuelo |
| **HTTP server timeouts** | `ReadHeaderTimeout=5s`, `ReadTimeout=10s`, `WriteTimeout=15s`, `IdleTimeout=60s` — defensa anti-slowloris |
| **Logs estructurados** | `log/slog` JSON con `request_id`, `method`, `path`, `status`, `duration`, `client_ip` |
| **Request correlation** | Middleware `X-Request-Id` (honra entrante, genera si falta, devuelve en response) |
| **Security headers** | `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer` |
| **Trusted proxies** | `SetTrustedProxies(nil)` — no confía en headers de proxies por default |
| **Gin release mode** | Default `ReleaseMode` (no debug warnings en stdout); override por `GIN_MODE` env |
| **Cap anti-DoS** | `MaxCompareIDs=50` en `/compare` |
| **Validación de path** | `PRODUCTS_FILE` rechaza no-`.json` y segmentos `..` (defense-in-depth) |
| **Fail-fast** | El binario aborta si no puede cargar el catálogo |

## Lo que NO está incluido (out of scope intencional)

| Pieza | Por qué afuera | Cómo se agregaría |
|---|---|---|
| **Auth / JWT** | El enunciado no lo pide; sin clientes definidos | Middleware Gin antes del router; domain no se entera |
| **Rate limiting** | Out of scope; mejor en el edge (LB/API Gateway de MELI) | Middleware (golang.org/x/time/rate) o Redis-backed para distribuido |
| **CORS** | API server-to-server por default; UI no definida | Middleware con whitelist de orígenes — decisión consciente, no olvido |
| **Cache** | Catálogo chico, latencias triviales | `Cache-Control` en endpoints idempotentes + ETag, o capa Redis |
| **Métricas Prometheus** | Requiere stack adicional (scraper, dashboards) para evaluar | `gin-prometheus` middleware + `/metrics` |
| **OpenAPI/Swagger** | Para 5 endpoints, README curado comunica más rápido | `swaggo/swag` con anotaciones; útil cuando >10 endpoints |
| **Versionado `/v1/...`** | Sin clientes existentes, agregar versión es cargo-cult | Cuando llegue el primer breaking change real |
| **Tracing distribuido (OTel)** | Servicio único, request_id alcanza | OpenTelemetry SDK + exportador (Jaeger/Tempo) |
| **DB real** | Enunciado dice explícitamente "no se requiere" | Nuevo adapter `infrastructure/persistence/postgres_repository.go` que implemente el mismo port |
| **CI/CD pipelines** | Fuera del alcance del challenge | GitHub Actions con `make check` + `govulncheck` |

> **Saber cuándo NO hacer algo es tan senior como saber hacerlo.** El diseño admite cualquiera de estas piezas como adapter adicional sin reescribir el dominio.

## Cómo extender

| Quiero agregar... | Toco... |
|---|---|
| Una categoría de producto nueva | Solo `testdata/products.json` (gracias al `Specs` flexible) |
| Un endpoint de búsqueda | Nuevo use case en `application/` + handler en `infrastructure/http/` |
| Persistencia en PostgreSQL | Nuevo adapter `infrastructure/persistence/postgres_repository.go` que implemente el mismo port — `cmd/api/main.go` cambia 1 línea |
| Autenticación | Middleware Gin en `infrastructure/http/` — domain y application no se enteran |

---

> Para portar este código a la plataforma de HackerRank, ver [`HACKERRANK.md`](./HACKERRANK.md).
