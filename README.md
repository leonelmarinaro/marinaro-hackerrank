# Backend API — Comparación de Productos (Mercado Libre Challenge)

API REST en **Go + Gin** para servir detalles de productos y comparar items con proyección selectiva de campos.

> Implementado con **Clean Architecture liviana** y **TDD estricto**.

## Stack

| Componente | Elección | Por qué |
|---|---|---|
| Lenguaje | Go 1.21+ | Tipado fuerte, testing nativo, compilación rápida, alineado con stack MELI |
| Framework HTTP | Gin | Routing simple, middleware estándar, performance |
| Persistencia | JSON file in-memory | Sin drivers, portable, fixtures fáciles |
| Testing | `testing` stdlib + `httptest` | Sin dependencias de mocking, idiomático Go |

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
| 400 | `ids` ausente, `fields` con valor fuera de la whitelist, `page` o `size` no numéricos / negativos |
| 404 | Producto no existe. En `/compare` el body incluye `"missing_ids": [...]` con detalle |
| 500 | Error interno (genérico — nunca filtra detalles) |

### Detalles de comportamiento

- **`/compare` deduplica IDs**: `?ids=1,1,2` se procesa como `[1, 2]`. Comparar X con X no aporta información y devolver duplicados confunde al cliente.
- **`/products?page=999`** (fuera de rango): `200` con `items: []` y `total` real — el cliente sabe que ya iteró.
- **`/products?page=-1`**: `400` (validación strict). Si NO se pasa el query param, se aplica el default silenciosamente.

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

### Cobertura

```bash
go test ./... -cover                                       # totales por paquete
go test ./... -coverprofile=cover.out                      # detalle por línea
go tool cover -html=cover.out                              # visualización en browser
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

**45 tests** en 5 paquetes:

```
domain          → 6 tests   (entity + errors)
application     → 14 tests  (use cases con mocks)
persistence     → 11 tests  (adapter JSON con archivos de prueba)
http            → 14 tests  (e2e vía httptest, incluye dedup, paginación strict, page out-of-range)
```

Cada error path está cubierto: empty IDs, invalid field, missing IDs, not found, paginación inválida, IDs duplicados, page fuera de rango.

## Lo que NO está incluido (out of scope intencional)

- Auth / JWT
- Rate limiting
- Cache
- Observabilidad (Prometheus, OpenTelemetry)
- DB real (PostgreSQL, Redis)
- CI/CD pipelines

Estas piezas son críticas en producción pero el challenge no las pide. Agregarlas aquí leería como over-engineering. El diseño las admite sin reescribir nada — son adapters adicionales.

## Cómo extender

| Quiero agregar... | Toco... |
|---|---|
| Una categoría de producto nueva | Solo `testdata/products.json` (gracias al `Specs` flexible) |
| Un endpoint de búsqueda | Nuevo use case en `application/` + handler en `infrastructure/http/` |
| Persistencia en PostgreSQL | Nuevo adapter `infrastructure/persistence/postgres_repository.go` que implemente el mismo port — `cmd/api/main.go` cambia 1 línea |
| Autenticación | Middleware Gin en `infrastructure/http/` — domain y application no se enteran |

---

> Para portar este código a la plataforma de HackerRank, ver [`HACKERRANK.md`](./HACKERRANK.md).
