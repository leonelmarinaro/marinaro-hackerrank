# Prompts compactos para HackerRank

Pegá el SYSTEM una vez, después los prompts en orden. Tu module path = `MOD` (reemplazalo).

---

## SYSTEM (pegar 1 vez)

```

---

## P14 — 5 extras mínimos (compacto)

```bash
# Implementar sin sobreingeniería:
# 1) .github/workflows/ci.yml con gofmt check + go vet ./... + go test ./...
# 2) .githooks/pre-commit (ejecutable): gofmt -w . ; go vet ./... ; go test ./...
# 3) OBSERVABILITY.md: logs, request_id, 500 genérico, headers, timeouts, demo checklist
# 4) PRODUCTION-NEXT-STEPS.md: pasos realistas + por qué (priorizado)
# 5) Dockerfile multistage simple (sin compose/k8s/deploy automation)

# Restricciones:
# - no devcontainer
# - no CI/CD pesado
# - no Prometheus/Grafana/OTel reales
# - evitar build salvo estrictamente necesario

# Validar:
gofmt -l .
go vet ./...
go test ./...

# Reportar:
# - files changed
# - checks run
# - riesgos/trade-offs
```
Sos un agente que sigue instrucciones al pie. Construimos un backend Go + Gin paso a paso.

REGLAS:
1. NO agregues lo que no pedí. Si dudás, preguntá.
2. NO uses Wire, Fx, gomock, testify/mock. DI y mocks manuales.
3. NO cambies nombres de archivos ni estructura.
4. Comentarios = WHY, nunca WHAT.
5. Tests primero (TDD), impl después.
6. Después de cada prompt listame archivos creados con `git status`.
7. Si algo es ambiguo, preguntá ANTES de improvisar.

ARQUITECTURA:
- Go ≥ 1.21 + Gin
- 3 capas: domain (sin deps) / application (depende de domain) / infrastructure (depende de ambas)
- cmd/api/main.go = composition root
- Persistencia: JSON in-memory cargado al boot
- Specs como map[string]any
- Logs: log/slog JSON
- HTTP server con timeouts + graceful shutdown
```

---

## P0 — Setup

```
1. go mod init github.com/MOD
2. mkdir -p cmd/api internal/domain internal/application internal/infrastructure/persistence internal/infrastructure/http testdata
3. go get github.com/gin-gonic/gin

VALIDAR: go env GOMOD && ls -la
```

---

## P1 — Domain (4 archivos, 6 tests)

```
internal/domain/product.go (package domain)
- struct Product: ID, Name, Description, ImageURL, Price, Rating, Category (string/float64), Size, Weight, Color (opcionales con omitempty), Specs map[string]any.
  JSON tags snake_case.
- var allowedFields map[string]struct{} con: id, name, description, image_url, price, rating, category, size, weight, color, specs.
- func IsAllowedField(name string) bool — case-sensitive.
- método (p Product) SelectFields(fields []string) map[string]any:
    nil/empty → todo el producto como map.
    sino → solo los pedidos que existan.
    campos opcionales en cero NO aparecen nunca.
- método privado (p Product) toMap() map[string]any (helper interno).
- COMENTARIOS WHY: por qué Specs es map[string]any (extensibilidad), por qué whitelist (seguridad), por qué SelectFields retorna map (omitir > null).

internal/domain/errors.go (package domain, imports: errors fmt strings)
- sentinels: ErrProductNotFound, ErrInvalidField, ErrEmptyIDs, ErrInvalidPagination, ErrTooManyIDs (todos errors.New).
- const MaxCompareIDs = 50.
- type MissingIDsError struct { Missing []string }
    Error() = fmt.Sprintf("products not found: %s", strings.Join(e.Missing, ", "))
    Unwrap() = ErrProductNotFound

internal/domain/repository.go (package domain, sin imports)
- interface ProductRepository:
    FindByID(id string) (*Product, error)            // ErrProductNotFound si falta
    FindByIDs(ids []string) ([]Product, error)       // *MissingIDsError si falta alguno; preserva orden
    List(offset, limit int) ([]Product, int, error)  // page + total
    Categories() ([]string, error)                   // distintas, ordenadas

internal/domain/product_test.go (package domain)
Tests:
1. TestIsAllowedField — table: name/price/specs/image_url=true; cost_price/internal_sku/""/NAME=false.
2. TestSelectFields_FullProductWhenFieldsEmpty — nil → id/name/specs presentes.
3. TestSelectFields_SubsetReturnsOnlyRequested — ["name","price"] → len==2, category NO está.
4. TestSelectFields_UnknownFieldSilentlyIgnored — "totally_invented" no debe estar.
5. TestSelectFields_OptionalZeroFieldsOmitted — Product{ID,Name} → size/weight/color/specs NO aparecen.
6. TestMissingIDsError_UnwrapToProductNotFound — errors.Is(err, ErrProductNotFound) true.

Helper newSampleProduct() → iPhone 15 con specs.

VALIDAR: go test ./internal/domain/... -v → 6 tests verde
```

---

## P2 — Application (9 archivos, 16 tests)

```
internal/application/compare_products.go (imports: fmt, MOD/internal/domain)
- struct CompareProductsUseCase{ repo domain.ProductRepository } + NewCompareProductsUseCase(repo).
- struct CompareProductsResult{ Items []map[string]any `json:"items"`; Fields []string `json:"fields"` }.
- Execute(ids, fields []string) (*CompareProductsResult, error) — orden EXACTO:
    a. len(ids)==0 → ErrEmptyIDs.
    b. ids = dedupPreservingOrder(ids).
    c. len(ids) > MaxCompareIDs → fmt.Errorf("%w: got %d, max %d", ErrTooManyIDs, len(ids), MaxCompareIDs).
    d. para cada f: !IsAllowedField(f) → fmt.Errorf("%w: %q", ErrInvalidField, f).
    e. products, err := repo.FindByIDs(ids); err → return.
    f. items := SelectFields por cada producto.
    g. return &CompareProductsResult{Items, Fields}.
- func privada dedupPreservingOrder(in []string) []string — set + slice, O(n), preserva orden.

internal/application/list_products.go
- struct ListProductsUseCase + NewListProductsUseCase.
- struct ListProductsResult{ Items []domain.Product; Total, Page, Size int } con json tags.
- Execute(page, size int):
    page<1 → 1; size<1 → 20; size>100 → 100 (cap).
    offset = (page-1)*size.
    repo.List(offset, size).

internal/application/get_product.go
- struct GetProductUseCase + constructor.
- Execute(id string): id=="" → ErrEmptyIDs; sino repo.FindByID(id).

internal/application/list_categories.go
- struct ListCategoriesUseCase + constructor.
- Execute() → repo.Categories().

internal/application/mock_repository_test.go (package application)
- struct mockRepository con 4 campos func: findByIDFn, findByIDsFn, listFn, categoriesFn.
- 4 métodos del port. CADA UNO nil-checkea su func y panic con mensaje "mockRepository.X called but XFn is nil — check test setup".
- helper sampleProducts() → 3 productos: iPhone 15 (id=1, smartphones), Galaxy S24 (id=2, smartphones), Clean Code (id=3, books).

internal/application/compare_products_test.go (8 tests):
1. HappyPath — Execute([1,2], [name,price]) → 2 items, name presente, category NO.
2. FullProjectionWhenFieldsEmpty — Execute([1], nil) → category SÍ presente.
3. EmptyIDsRejected — Execute(nil, nil) → errors.Is ErrEmptyIDs.
4. InvalidFieldRejected — Execute([1], [name,cost_price]) → errors.Is ErrInvalidField.
5. RepoErrorPropagated — mock devuelve *MissingIDsError → errors.Is ErrProductNotFound + errors.As MissingIDsError true.
6. DeduplicatesIDs — Execute([1,1,2,1], nil) → mock recibe EXACTAMENTE [1,2]. Capturar con closure.
7. RejectsTooManyIDs — generar MaxCompareIDs+1 únicos (ej: string(rune('a'+i%26))+string(rune('0'+i/26))) → errors.Is ErrTooManyIDs.
8. DoesNotCallRepoWhenValidationFails — Execute([1], [invalid]) → flag called sigue false.

internal/application/list_products_test.go (4 tests):
1. DefaultPagination — Execute(0,0) → page=1, size=20.
2. PageToOffsetConversion — Execute(3,10) → mock recibe offset=20.
3. SizeCappedAt100 — Execute(1,10000) → size=100.
4. ReturnsTotalForPagination — total=42.

internal/application/get_product_test.go (3 tests): HappyPath, EmptyIDRejected, NotFoundPropagated.

internal/application/list_categories_test.go (1 test): PassthroughToRepo.

VALIDAR: go test ./internal/application/... -v → 16 tests verde
```

---

## P3 — Persistence (3 archivos, 11 tests)

```
testdata/products.json: array con 8 productos:
- 3 smartphones (ids 1,2,3): iPhone 15 Pro, Galaxy S24 Ultra, Pixel 8 Pro. Specs: brand, model, battery, ram, storage, camera, os, screen.
- 2 libros (ids 4,5): Clean Code, Pragmatic Programmer. Specs: author, publisher, pages, language, isbn, year.
- 2 prendas (ids 6,7): remera blanca, buzo capucha. Con size+color root + specs: material, care, origin.
- 1 electrodoméstico (id 8): cafetera DeLonghi. Specs: brand, model, power, voltage, warranty.
Cada uno: id (string), name, description, image_url, price, rating, category, weight.

internal/infrastructure/persistence/json_repository.go (package persistence, imports: encoding/json fmt os sort sync MOD/internal/domain)
- struct JSONRepository{ mu sync.RWMutex; products []domain.Product; byID map[string]domain.Product }.
- NewJSONRepository(path string) (*JSONRepository, error):
    os.ReadFile + wrap %w; json.Unmarshal + wrap %w.
    iterar: si p.ID == "" → return error "empty id at index N".
    iterar: si byID[p.ID] ya existe → return error "duplicate product id".
- TODOS los métodos: r.mu.RLock() defer Unlock().
- FindByID: byID lookup → ErrProductNotFound si falta.
- FindByIDs: iterar input EN ORDEN. found + missing slices. Si len(missing)>0 → &MissingIDsError{Missing}.
- List(offset, limit): total=len(products); si offset>=total → []domain.Product{}, total, nil. clamp end. defensive copy con copy().
- Categories: set + sort.Strings.
- COMENTARIO crítico en doc del struct: "Specs es read-only para el consumidor — el map se comparte por referencia. No mutar."

internal/infrastructure/persistence/json_repository_test.go (12 tests):
const sampleJSON = `[{"id":"1","name":"A","category":"x","price":10},{"id":"2","name":"B","category":"y","price":20},{"id":"3","name":"C","category":"x","price":30}]`
helper writeTempJSON(t, content) string → t.TempDir() + write file.
helper setupRepo(t) *JSONRepository.

Tests: LoadsValidFile, FailsOnMissingFile, FailsOnInvalidJSON, FailsOnDuplicateIDs, FailsOnEmptyID, FindByID_Found, FindByID_NotFound, FindByIDs_AllFound_PreservesOrder ([3,1] → [3,1]), FindByIDs_PartialMissingReturnsError (2 missing), List_Pagination, List_OffsetBeyondTotalReturnsEmpty (offset=100 → [], total=3), Categories_DistinctAndSorted.

VALIDAR: go test ./internal/infrastructure/persistence/... -v → 12 tests verde
```

---

## P4 — HTTP aux (3 archivos, sin tests)

```
internal/infrastructure/http/dto.go (package http, sin imports)
- type errorResponse struct { Error string `json:"error"`; MissingIDs []string `json:"missing_ids,omitempty"` }
- type healthResponse struct { Status string `json:"status"` }

internal/infrastructure/http/error_handler.go (imports: errors net/http gin MOD/internal/domain)
- func writeError(c *gin.Context, err error):
    var miss *domain.MissingIDsError
    if errors.As(err, &miss) → c.JSON(404, errorResponse{Error: err.Error(), MissingIDs: miss.Missing}); return.
    switch:
      ErrProductNotFound → 404.
      ErrEmptyIDs|ErrInvalidField|ErrInvalidPagination|ErrTooManyIDs → 400.
      default → 500 con "internal server error" GENÉRICO. NUNCA err.Error().

internal/infrastructure/http/middleware.go (imports: crypto/rand encoding/hex log/slog time gin)
- const requestIDHeader = "X-Request-Id"
- func RequestIDMiddleware() gin.HandlerFunc:
    rid := c.GetHeader(requestIDHeader); if "" → newRequestID().
    c.Set("request_id", rid); c.Writer.Header().Set(requestIDHeader, rid); c.Next().
- func LoggingMiddleware(logger *slog.Logger) gin.HandlerFunc:
    start := time.Now(); c.Next().
    logger.Info("http_request", slog.String("request_id", c.GetString("request_id")), String method/path/client_ip, Int status, Duration time.Since(start)).
- func SecurityHeadersMiddleware() gin.HandlerFunc:
    Set "X-Content-Type-Options"="nosniff", "Referrer-Policy"="no-referrer". c.Next().
- func newRequestID() string:
    8 bytes con crypto/rand.Read → hex.EncodeToString. Si falla → time.Now().Format("150405.000000000").

VALIDAR: go build ./internal/infrastructure/http/...
```

---

## P5 — HTTP handler + router

```
internal/infrastructure/http/product_handler.go (imports: fmt net/http strconv strings gin MOD/internal/application MOD/internal/domain)
- struct ProductHandler con 4 use cases privados. NewProductHandler(compare, list, get, categories).
- Health: c.JSON(200, healthResponse{Status:"ok"}).
- Compare: ids=splitCSV(c.Query("ids")); fields=splitCSV(c.Query("fields")); res,err=compareUC.Execute; err→writeError. (NO dedup acá, está en use case.)
- List: page,err=parsePositiveInt(c,"page",1); err→writeError; size,err=parsePositiveInt(c,"size",20); err→writeError; res,err=listUC.Execute(page,size).
- Get: id=c.Param("id"); p,err=getUC.Execute(id).
- Categories: cats,err=categoriesUC.Execute(); c.JSON(200, gin.H{"categories":cats}).
- splitCSV(raw): "" → nil; sino strings.Split + TrimSpace + skip vacíos.
- parsePositiveInt(c, name, defaultValue):
    raw := c.Query(name); raw=="" → defaultValue, nil.
    v,err := strconv.Atoi(raw); err!=nil || v<1 → 0, fmt.Errorf("%w: %s=%q", ErrInvalidPagination, name, raw).
    sino → v, nil.

internal/infrastructure/http/router.go (imports: log/slog gin)
- func NewRouter(h *ProductHandler, logger *slog.Logger) *gin.Engine:
    r := gin.New()                          // NO gin.Default
    _ = r.SetTrustedProxies(nil)
    r.Use(gin.Recovery(), RequestIDMiddleware(), LoggingMiddleware(logger), SecurityHeadersMiddleware())
    r.GET("/health", h.Health)
    products := r.Group("/products")
    products.GET("", h.List)
    products.GET("/compare", h.Compare)        // ANTES que /:id
    products.GET("/categories", h.Categories)  // ANTES que /:id
    products.GET("/:id", h.Get)
    return r

VALIDAR: go build ./internal/infrastructure/http/...
```

---

## P6 — HTTP tests e2e (1 archivo, 14 tests)

```
internal/infrastructure/http/product_handler_test.go (imports: encoding/json io log/slog net/http net/http/httptest os path/filepath testing gin MOD/internal/application MOD/internal/infrastructure/persistence)

const fixtureJSON = `[
  {"id":"1","name":"iPhone","category":"smartphones","price":1000,"rating":4.5,"specs":{"ram":"8GB"}},
  {"id":"2","name":"Galaxy","category":"smartphones","price":900,"rating":4.4,"specs":{"ram":"12GB"}},
  {"id":"3","name":"Book","category":"books","price":30,"rating":4.8}
]`

helper setupRouter(t) *gin.Engine:
  gin.SetMode(gin.TestMode)
  escribir fixture en t.TempDir()/products.json
  repo, err := persistence.NewJSONRepository(path); if err != nil → t.Fatalf("repo: %v", err)
  h := NewProductHandler(
    application.NewCompareProductsUseCase(repo),
    application.NewListProductsUseCase(repo),
    application.NewGetProductUseCase(repo),
    application.NewListCategoriesUseCase(repo),
  )
  silentLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))   // logger silencioso en tests
  return NewRouter(h, silentLogger)

helper doRequest(t, router, method, url) (*ResponseRecorder, map[string]any):
  httptest.NewRecorder + http.NewRequest + ServeHTTP + json.Unmarshal.

Tests (14):
1. TestHealth → 200, status=ok.
2. TestListProducts_Default → 200, total=3.
3. TestGetProduct_Found → 200, name=iPhone.
4. TestGetProduct_NotFound → 404, error no nil.
5. TestCategories → 200, 2 categorías.
6. TestCompare_HappyPath /compare?ids=1,2&fields=name,price → 200, 2 items, name presente, category NO.
7. TestCompare_MissingIDsReturns404 /compare?ids=1,99 → 404, missing_ids=["99"].
8. TestCompare_EmptyIDsReturns400 /compare → 400.
9. TestCompare_InvalidFieldReturns400 /compare?ids=1&fields=cost_price → 400.
10. TestList_PageOutOfRangeReturnsEmptyItemsWithTotal /products?page=999&size=10 → 200, items=[], total=3.
11. TestList_NegativePageReturns400 /products?page=-1 → 400.
12. TestList_NonNumericSizeReturns400 /products?size=abc → 400.
13. TestCompare_DuplicateIDsAreDeduplicated /compare?ids=1,1,2&fields=name → 200, 2 items.
14. TestCompare_FullProjectionWhenFieldsOmitted /compare?ids=1 → 200, todos los campos.

VALIDAR: go test ./... -cover → 47 tests verde
```

---

## P7 — Composition root con graceful shutdown

```
cmd/api/main.go (package main, imports: context errors log/slog net/http os os/signal path/filepath strings syscall time gin MOD/internal/application httpadapter "MOD/internal/infrastructure/http" MOD/internal/infrastructure/persistence)

const (
  readHeaderTimeout = 5 * time.Second
  readTimeout       = 10 * time.Second
  writeTimeout      = 15 * time.Second
  idleTimeout       = 60 * time.Second
  shutdownTimeout   = 10 * time.Second
)

func main():
  1. logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})).
  2. if mode := os.Getenv("GIN_MODE"); mode != "" → gin.SetMode(mode); else gin.SetMode(gin.ReleaseMode).
  3. dataPath := os.Getenv("PRODUCTS_FILE"); "" → "testdata/products.json".
  4. validateProductsPath(dataPath); err → logger.Error + os.Exit(1).
  5. repo, err := persistence.NewJSONRepository(dataPath); err → Error + Exit. logger.Info("products loaded").
  6. instanciar 4 use cases.
  7. handler := httpadapter.NewProductHandler(...); router := httpadapter.NewRouter(handler, logger).
  8. srv := &http.Server{Addr: ":"+portFromEnv(), Handler: router, ReadHeaderTimeout, ReadTimeout, WriteTimeout, IdleTimeout}.
  9. serverErr := make(chan error, 1).
     go func() {
       logger.Info("server listening", slog.String("addr", srv.Addr))
       if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) { serverErr <- err }
     }()
  10. stop := make(chan os.Signal, 1); signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM).
  11. select:
        case err := <-serverErr → logger.Error + os.Exit(1).
        case sig := <-stop → logger.Info("shutdown signal received", String signal).
  12. ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout); defer cancel().
      if err := srv.Shutdown(ctx) → Error("graceful shutdown failed, forcing close") + srv.Close() + Exit(1).
      logger.Info("server stopped cleanly").

func portFromEnv() string: PORT env o "8080".

func validateProductsPath(path string) error:
  path=="" → errors.New("path is empty").
  !strings.HasSuffix(strings.ToLower(path), ".json") → errors.New("path must end in .json").
  strings.HasPrefix(filepath.Clean(path), "..") → errors.New("path contains traversal segments").
  return nil.

VALIDAR:
  go build ./cmd/api && go test ./... -cover && go vet ./...
  smoke shutdown: /tmp/api & PID=$!; sleep 1; kill -TERM $PID; wait $PID; echo $?
  Esperado: logs JSON "shutdown signal received" + "server stopped cleanly", exit 0.
```

---

## P8 — Tooling

```
.gitignore:
/api
/bin/
*.exe
*.test
*.out
coverage.out
coverage.html
.DS_Store
.idea/
.vscode/
*.swp
.env
.env.local

Makefile:
.PHONY: run build test test-v cover cover-html vet fmt tidy vulncheck check
run:        ; go run ./cmd/api
build:      ; go build -o bin/api ./cmd/api
test:       ; go test ./...
test-v:     ; go test -v ./...
cover:      ; go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1
cover-html: ; go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html
vet:        ; go vet ./...
fmt:        ; gofmt -w .
tidy:       ; go mod tidy
vulncheck:  ; @which govulncheck > /dev/null || go install golang.org/x/vuln/cmd/govulncheck@latest
            ; govulncheck ./...
check: fmt vet test

(Cuidado con la sintaxis make: las recetas usan TAB no espacios. Si el ; no funciona en tu HackerRank, usá líneas separadas con TAB.)

VALIDAR: make check && make build
```

---

## P9 — Smoke

```bash
go build -o /tmp/api ./cmd/api && /tmp/api &
PID=$!; sleep 2

curl -sw "\n%{http_code}\n" localhost:8080/health
curl -sw "\n%{http_code}\n" "localhost:8080/products?page=1&size=2"
curl -sw "\n%{http_code}\n" localhost:8080/products/1
curl -sw "\n%{http_code}\n" localhost:8080/products/9999
curl -sw "\n%{http_code}\n" localhost:8080/products/categories
curl -sw "\n%{http_code}\n" "localhost:8080/products/compare?ids=1,2,3&fields=name,price,rating"
curl -sw "\n%{http_code}\n" "localhost:8080/products/compare"
curl -sw "\n%{http_code}\n" "localhost:8080/products/compare?ids=1,99"
curl -sw "\n%{http_code}\n" "localhost:8080/products/compare?ids=1&fields=cost_price"
LONG=$(seq 1 60 | tr '\n' ',' | sed 's/,$//')
curl -sw "\n%{http_code}\n" "localhost:8080/products/compare?ids=$LONG"
curl -s "localhost:8080/products/compare?ids=1,1,2&fields=name"
curl -si localhost:8080/health | grep -iE "x-content|referrer|x-request-id"
curl -si -H "X-Request-Id: trace-123" localhost:8080/health | grep -i x-request-id
curl -sw "\n%{http_code}\n" "localhost:8080/products?page=-1"
curl -sw "\n%{http_code}\n" "localhost:8080/products?size=abc"
curl -sw "\n%{http_code}\n" "localhost:8080/products?page=999"

kill -TERM $PID; wait $PID; echo $?

ESPERADOS: 200/200/200/404/200/200/400/404/200/400/2 items dedup/headers presentes/x-request-id=trace-123/400/400/200 con items=[]/exit 0.
```

---

## P10 — Documentación

```
problem.md: pegar el original (en español).

README.md (en español, secciones en este orden):
1. Título + tagline "Clean Architecture liviana + TDD estricto".
2. Stack (tabla).
3. Arquitectura (diagrama ASCII de 3 capas + composition).
4. Estructura de directorios.
5. Endpoints (tabla método/path/desc).
6. Errores (tabla status/cuándo).
7. Schemas de respuesta (JSON example por endpoint).
8. Detalles: dedup en use case, MaxCompareIDs=50, page out-of-range = 200 vacío, page=-1 = 400 strict, X-Request-Id behavior.
9. Setup.
10. Comandos Makefile.
11. Cobertura + govulncheck.
12. Smoke test (curl + responses esperadas).
13. Decisiones arquitectónicas (tabla con trade-off).
14. Robustez de runtime (tabla 10 items: graceful shutdown, HTTP timeouts, slog JSON, request_id, security headers, trusted proxies, gin release, cap, validación path, fail-fast).
15. Cobertura: 47 tests / 5 packages.
16. Lo que NO está incluido (tabla pieza/por qué afuera/cómo se agregaría): Auth, Rate limit, CORS, Cache, Métricas, OpenAPI, Versionado, OTel, DB real, CI/CD.
17. Cómo extender (tabla "agregar X / toco Y").
18. Cierre: "Saber cuándo NO hacer algo es tan senior como saber hacerlo".
```

---

## Frenos rápidos

| Síntoma | Decir |
|---|---|
| Cambia layout/nombres | "STOP regla 3" |
| Agrega features | "STOP regla 1, eso no estaba" |
| Mete Wire/gomock | "STOP regla 2, manual" |
| Comentarios WHAT | "STOP regla 4, WHY no WHAT" |
| Endpoints renombrados | "STOP, son /health /products /products/compare /products/categories /products/:id" |
| Dedup/cap en handler | "STOP, va en use case" |
| 200 con productos parciales | "STOP, debe ser 404 con missing_ids" |
| router.Run() en main | "STOP, http.Server explícito + Shutdown" |
| gin.Default() | "STOP, gin.New() + middlewares custom" |
| Agrega /ping u otro endpoint | "STOP, solo los 5" |
| Mock sin nil-check | "STOP, panic con mensaje descriptivo si fn==nil" |

---

## Checklist final

- go test ./... → verde (con +8 tests respecto al estado previo)
- go vet ./... → clean
- go build ./cmd/api → ok
- govulncheck → no vulns
- smoke P9 → todos OK
- SIGTERM → "stopped cleanly" exit 0
- Headers en TODA respuesta: nosniff, no-referrer, X-Request-Id
- README + problem.md + Makefile + .gitignore presentes

---

## P11 — Robustez técnica (tests operativos/config)

```bash
# Objetivo: safety net sin cambiar semántica pública

# 1) middleware tests
# - preserva X-Request-Id entrante
# - genera X-Request-Id si falta
# - agrega X-Content-Type-Options=nosniff y Referrer-Policy=no-referrer

# 2) error handler tests
# table mapping:
# ErrProductNotFound->404
# MissingIDsError->404
# ErrEmptyIDs/ErrInvalidField/ErrInvalidPagination/ErrTooManyIDs->400
# unknown error -> 500

# 3) main helpers tests (cmd/api/main_test.go)
# portFromEnv: usa PORT o default 8080
# validateProductsPath: válido json/case-insensitive, inválido empty/ext/traversal

# 4) router precedence
# test puntual: /products/categories debe resolver Categories y NO /products/:id

# Validar (sin build):
go test ./cmd/api ./internal/infrastructure/http -v
go test ./... -v

# Esperado: todo verde, +8 tests netos
```

---

## P12 — Hardening inputs/config (manual y corto)

```
Objetivo: endurecer bordes sin romper contrato público.

Cambios:
1) product_handler_test.go
   - /products/compare?ids=,%20,%20,, -> 400 (CSV vacío)
   - /products/compare?ids=%201%20,%20%202%20,,&fields=%20name%20,%20price%20 -> 200, items=2, fields=[name,price]
   - /products/compare?ids=1&ids=2&fields=name -> 200, items=1 (se conserva semántica actual: primer query value)
   - /products?page=999999999999999999999999 -> 400
   - /products?size=1e3 -> 400

2) main.go / main_test.go
   - Bug real: PRODUCTS_FILE con espacios alrededor puede fallar por falso negativo.
   - Fix mínimo: strings.TrimSpace al leer PRODUCTS_FILE y dentro de validateProductsPath.
   - Tests: "  testdata/products.json  " válido; "testdata/../../secrets.json" inválido.

Validar (sin build):
go test ./internal/infrastructure/http ./cmd/api -v
go test ./... -v
```

---

## P13 — Observabilidad 500 (manual, corto y accionable)

```bash
# Objetivo: trazabilidad interna de 500 sin cambiar contrato público

# 1) Revisar writeError + LoggingMiddleware
# files:
# - internal/infrastructure/http/error_handler.go
# - internal/infrastructure/http/middleware.go

# 2) Cambio mínimo
# - En default 500: registrar causa real en contexto interno (no en payload)
# - Mantener response público: {"error":"internal server error"}
# - Log estructurado debe incluir request_id (si existe) + internal_error

# 3) Tests
# - Ajustar TestWriteError_UnknownErrorReturns500:
#   además de 500, verificar que unknown error queda registrado en gin.Context
# - Agregar test integración liviano:
#   RequestIDMiddleware + LoggingMiddleware + handler que dispara errors.New("db connection timeout")
#   asserts: payload genérico, log contiene request_id + internal_error

# Validar (sin build):
go test ./internal/infrastructure/http -v
go test ./... -v

# Esperado:
# - cliente ve 500 genérico
# - causa real trazable en logs internos
```
