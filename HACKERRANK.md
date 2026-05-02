# Guía paso a paso — Portar a HackerRank

Esta guía describe cómo subir el código a la plataforma de HackerRank simulando el flujo TDD que se trabajó en local. La estrategia es **copiar capa por capa, verificando tests después de cada bloque**.

## Antes de empezar

1. Abrir el challenge → pestaña de archivos / editor.
2. Verificar que la plataforma:
   - Soporta múltiples archivos y subdirectorios (la mayoría sí)
   - Permite Go ≥ 1.21 (necesario para `any`)
   - Ejecuta tests con `go test ./...` o un comando equivalente
3. Confirmar el module path. Si HackerRank fija un module name específico, **reemplazar `github.com/lmarinaro/marinaro-hackerrank` por el que pidan** en todos los `import` y en el `go.mod`.

> Si la plataforma sólo permite **un único archivo**, ver el Apéndice A al final.

## Mapa de archivos

```
cmd/api/main.go                                        ← composition root
internal/domain/product.go
internal/domain/errors.go
internal/domain/repository.go
internal/domain/product_test.go
internal/application/compare_products.go
internal/application/list_products.go
internal/application/get_product.go
internal/application/list_categories.go
internal/application/mock_repository_test.go
internal/application/compare_products_test.go
internal/application/list_products_test.go
internal/application/get_product_test.go
internal/application/list_categories_test.go
internal/infrastructure/persistence/json_repository.go
internal/infrastructure/persistence/json_repository_test.go
internal/infrastructure/http/dto.go
internal/infrastructure/http/error_handler.go
internal/infrastructure/http/product_handler.go
internal/infrastructure/http/router.go
internal/infrastructure/http/product_handler_test.go
testdata/products.json
go.mod
go.sum
README.md
problem.md
```

## Orden de copia (de adentro hacia afuera)

> Después de cada bloque ejecutar `go test ./...` y confirmar verde antes de seguir.

### Bloque 0 — Setup del módulo
1. `go.mod` → si HackerRank fija un module path distinto, ajustarlo.
2. `go.sum` → copiar tal cual.

### Bloque 1 — Domain (sin dependencias)
1. `internal/domain/product.go`
2. `internal/domain/errors.go`
3. `internal/domain/repository.go`
4. `internal/domain/product_test.go`

**Checkpoint**: `go test ./internal/domain/...` → 6 tests OK.

### Bloque 2 — Application (depende sólo de domain)
5. `internal/application/compare_products.go`
6. `internal/application/list_products.go`
7. `internal/application/get_product.go`
8. `internal/application/list_categories.go`
9. `internal/application/mock_repository_test.go`
10. `internal/application/compare_products_test.go`
11. `internal/application/list_products_test.go`
12. `internal/application/get_product_test.go`
13. `internal/application/list_categories_test.go`

**Checkpoint**: `go test ./internal/application/...` → 14 tests OK.

### Bloque 3 — Persistence
14. `testdata/products.json`
15. `internal/infrastructure/persistence/json_repository.go`
16. `internal/infrastructure/persistence/json_repository_test.go`

**Checkpoint**: `go test ./internal/infrastructure/persistence/...` → 11 tests OK.

### Bloque 4 — HTTP layer
17. `internal/infrastructure/http/dto.go`
18. `internal/infrastructure/http/error_handler.go`
19. `internal/infrastructure/http/product_handler.go`
20. `internal/infrastructure/http/router.go`
21. `internal/infrastructure/http/product_handler_test.go`

**Checkpoint**: `go test ./internal/infrastructure/http/...` → 14 tests OK.

> Nota: el HTTP layer ahora incluye `middleware.go` con `RequestIDMiddleware`, `LoggingMiddleware(slog)` y `SecurityHeadersMiddleware`. `NewRouter` recibe un `*slog.Logger`. En tests usar `slog.New(slog.NewJSONHandler(io.Discard, nil))` para silencio.

### Bloque 5 — Composition root
22. `cmd/api/main.go`

**Checkpoint final**:
```bash
go test ./... -cover    # → 47 tests OK
go vet ./...            # sin warnings
go build ./cmd/api      # compila
make vulncheck          # govulncheck sin CVEs activos
```

### Bloque 6 — Documentación
23. `problem.md`
24. `README.md`
25. `HACKERRANK.md` (este archivo, opcional en HackerRank)

## Simulación del ciclo TDD en HackerRank

Si la entrevista valora ver TDD en vivo, mostralo en al menos 1-2 funciones clave. Sugerencia: `CompareProducts.Execute`.

### Demo del ciclo Red → Green → Refactor

**RED** — pegar primero SOLO el test (sin la implementación):
```go
// archivo: internal/application/compare_products_test.go
func TestCompareProducts_HappyPath(t *testing.T) { ... }
```
Correr `go test ./internal/application/...` → fail por método inexistente. Mostrar el error.

**GREEN** — pegar la implementación mínima:
```go
// archivo: internal/application/compare_products.go
func (uc *CompareProductsUseCase) Execute(...) (...) { ... }
```
Correr de nuevo → verde.

**REFACTOR** — extraer validaciones, agregar comentarios, sin romper tests. Re-correr → sigue verde.

Hacer este ciclo en vivo demuestra disciplina TDD sin alargar la entrevista.

## Checklist de verificación final

Antes de marcar el challenge como entregado:

- [ ] `go test ./...` pasa todo verde (47 tests)
- [ ] `go vet ./...` sin warnings
- [ ] `go build ./cmd/api` compila sin error
- [ ] `README.md` presente con setup, endpoints, decisiones
- [ ] `problem.md` en español
- [ ] `testdata/products.json` con productos heterogéneos (smartphones + libros + ropa + electrodomésticos)
- [ ] Si la plataforma permite ejecutar el binario:
  - [ ] `curl localhost:8080/health` → 200
  - [ ] `curl localhost:8080/products/compare?ids=1,2&fields=name,price` → 200 con projection
  - [ ] `curl localhost:8080/products/compare?ids=99` → 404 con `missing_ids`

## Ajustes por configuraciones de HackerRank

### Si limitan el path de archivos al directorio raíz
La plataforma de HackerRank a veces no permite subdirectorios profundos. Plan B:

- Mover todo a un único package `main` y aplanar la estructura. **Trade-off**: pierde la separación de capas a nivel package, pero la separación lógica (entity / use case / handler / repo) sigue clara dentro del archivo.
- Alternativa: dejar todo en `internal/` y mover solamente el `main.go` arriba.

### Si Go < 1.21
Reemplazar todos los `any` por `interface{}` (es exactamente el mismo tipo, alias agregado en Go 1.18).

### Si no permiten `gin`
Plan B: re-escribir `infrastructure/http/router.go` y `product_handler.go` usando `net/http` + `http.ServeMux`. El resto del código (domain, application, persistence) NO cambia — esa es la ventaja de Clean Architecture. La migración tomaría ~30 min.

## Apéndice A — Modo "single file"

Si HackerRank exige un único archivo, concatenar en este orden dentro de `package main`:
1. Tipos del `domain` (Product, errors, ProductRepository interface)
2. Adapter JSON (la struct y métodos)
3. Use cases (4)
4. Handlers Gin / net-http
5. `main()` con el wiring

Mover todos los tests a `main_test.go`. Aclarar al revisor que la estructura es plana **por restricción de la plataforma**, no por decisión de diseño — y mostrar el `README.md` con la estructura "nativa" para que se entienda el criterio.

## Apéndice B — Comandos útiles dentro del editor

```bash
# Cobertura por paquete
go test ./... -cover

# Cobertura detallada con HTML
go test ./... -coverprofile=cover.out && go tool cover -html=cover.out

# Tests verbose
go test ./... -v

# Solo tests del use case principal
go test ./internal/application/ -run TestCompareProducts -v

# Format check
gofmt -d .

# Vet (linter built-in)
go vet ./...
```

---

**Recordá**: HackerRank evalúa criterio + ejecución. Mostrá el orden de TDD, comentarios con WHY, y disciplina arquitectónica. La entrega debe ser legible, no impresionante.
