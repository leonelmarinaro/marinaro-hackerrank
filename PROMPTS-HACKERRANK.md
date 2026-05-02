# Prompts para reconstruir el proyecto en HackerRank con un agente IA

> Cada **Prompt N** es independiente y autosuficiente. Pegalo entero en el chat del agente, esperá a que termine, validá con el comando indicado y seguí con el próximo.
>
> Después de cada prompt, **validás visualmente** que los archivos que generó coincidan con los del repo local (`/Users/lmarinaro/Developer/Investigacion/marinaro-hackerrank`).

---

## Cómo usarlos

1. En HackerRank abrí el chat del agente IA.
2. Empezá por el **Prompt 0** (setup).
3. Después de cada respuesta del agente: corré el comando de **VALIDACIÓN** del prompt. Si pasa, seguís. Si falla, le decís al agente *“el comando X falló con: <pegá el error>, corregilo”*.
4. Si el agente desvía (cambia patrones, agrega lo que no le pediste), cortá y reintentá: *“no sigas. La regla X la rompiste. Corregí solo el archivo Y respetando el patrón.”*

> **Regla de oro**: el agente puede improvisar **dentro** de un prompt, pero **nunca** debe romper las decisiones arquitectónicas que ya están tomadas. Si lo hace, freno duro.

---

## Prompt 0 — Setup del proyecto

```
Vas a construir un backend REST API en Go + Gin siguiendo Clean Architecture liviana. Te voy a guiar en bloques. NO te adelantes — solo hacé lo que te pido en cada bloque.

DECISIONES ARQUITECTÓNICAS (no las cuestiones, ya están tomadas):
- Stack: Go ≥ 1.21 + Gin
- Persistencia: JSON file in-memory (cargado al boot)
- Arquitectura: 3 capas
    domain          — entidades + ports (interfaces). SIN dependencias externas.
    application     — use cases. Depende solo de domain.
    infrastructure  — adapters (HTTP handlers, repo JSON). Depende de application + domain.
- Composition root: cmd/api/main.go (DI manual, sin frameworks de DI)
- Modelo de Product: campos comunes tipados + Specs como map[string]any (flexibilidad > type-safety)
- Tests: stdlib `testing` + `httptest`. Mocks manuales (no gomock).
- Comentarios: SIEMPRE explican WHY (decisión, trade-off). NUNCA explican qué hace el código (el nombre ya lo dice).

TAREA AHORA:
1. Inicializar go.mod con module name `github.com/candidato/meli-comparison-api` (si HackerRank fija otro module name, usá ese).
2. Crear esta estructura de directorios vacíos:
   cmd/api/
   internal/domain/
   internal/application/
   internal/infrastructure/persistence/
   internal/infrastructure/http/
   testdata/
3. Agregar gin: `go get github.com/gin-gonic/gin`

NO escribas código de aplicación todavía. Solo setup.

VALIDACIÓN: `go env GOMOD` debe devolver el path al go.mod.
```

**Validación local**:
```bash
go env GOMOD
ls -la cmd/api internal/domain internal/application internal/infrastructure/{persistence,http} testdata
```

---

## Prompt 1 — Domain layer (entity + errors + port)

```
CONTEXTO: estamos en el paso 1 de Clean Architecture. La capa domain es la más interna — no depende de nada externo.

CREÁ exactamente estos 3 archivos en internal/domain/:

### product.go
- package domain
- Tipo Product con campos: ID, Name, Description, ImageURL, Price, Rating, Category (todos comunes y tipados), Size, Weight, Color (opcionales), Specs map[string]any.
- JSON tags coherentes: snake_case, omitempty para opcionales.
- Variable privada `allowedFields` map[string]struct{} con la WHITELIST de campos seleccionables vía API: id, name, description, image_url, price, rating, category, size, weight, color, specs.
- Función pública `IsAllowedField(name string) bool`.
- Método `(p Product) SelectFields(fields []string) map[string]any`:
  - Si fields está vacío/nil → retorna el producto completo como map.
  - Si fields tiene valores → retorna SOLO los que coincidan con la whitelist y existan en el producto.
  - Campos opcionales con valor cero (Size="", Weight=0, Color="", Specs nil/empty) NO deben aparecer en el map.
- COMENTARIOS: explicar POR QUÉ Specs es map[string]any (extensibilidad de categorías) y POR QUÉ devolvemos map en vez de un Product proyectado (omitir campos > mostrarlos como null).

### errors.go
- package domain
- Sentinel errors públicos:
    ErrProductNotFound       = errors.New("product not found")
    ErrInvalidField          = errors.New("invalid field requested")
    ErrEmptyIDs              = errors.New("at least one id is required")
    ErrInvalidPagination     = errors.New("invalid pagination parameters")
- Tipo MissingIDsError con campo Missing []string. Implementa Error() y Unwrap() devolviendo ErrProductNotFound (para que errors.Is(err, ErrProductNotFound) funcione cuando algún id falta en /compare).
- COMENTARIOS: por qué sentinel errors (mapeo desacoplado por errors.Is) y por qué MissingIDsError carga datos (cliente necesita saber QUÉ falta).

### repository.go
- package domain
- Interface ProductRepository con:
    FindByID(id string) (*Product, error)              // ErrProductNotFound si no existe
    FindByIDs(ids []string) ([]Product, error)          // *MissingIDsError con la lista si algún id falta. Mantiene el ORDEN del input.
    List(offset, limit int) ([]Product, int, error)     // retorna page + total
    Categories() ([]string, error)                       // categorías distintas, ORDENADAS alfabéticamente
- COMENTARIO: por qué la interface vive en domain (inversión de dependencias: el dominio define lo que necesita, infra lo implementa).

### product_test.go (TDD)
- Tests que verifiquen:
  * IsAllowedField con casos válidos, inválidos, vacío y case-sensitivity (NAME en mayúsculas → false).
  * SelectFields con fields=nil retorna todo.
  * SelectFields con subset retorna solo los pedidos.
  * SelectFields con field desconocido lo ignora silenciosamente.
  * SelectFields con campos opcionales en cero los OMITE del map.
  * MissingIDsError satisface errors.Is(err, ErrProductNotFound).

VALIDACIÓN: `go test ./internal/domain/... -v` → todos los tests verde.
```

**Validación local**:
```bash
go test ./internal/domain/... -v
```

---

## Prompt 2 — Application layer (4 use cases con TDD)

```
CONTEXTO: domain ya está. Ahora la capa application — 4 use cases, cada uno con dependencia inyectada del repo (vía constructor). NO conoce HTTP ni JSON ni archivos.

CREÁ en internal/application/:

### compare_products.go
- struct CompareProductsUseCase con campo privado `repo domain.ProductRepository`.
- Constructor: NewCompareProductsUseCase(repo).
- struct CompareProductsResult { Items []map[string]any `json:"items"`; Fields []string `json:"fields"` }.
- Método Execute(ids, fields []string) (*CompareProductsResult, error):
  1. Si ids vacío → ErrEmptyIDs.
  2. Si algún field NO está en la whitelist → wrap ErrInvalidField con %q del field.
  3. Llamar repo.FindByIDs (propaga errores tal cual — el handler los mapea).
  4. Para cada producto, aplicar SelectFields y armar Items.
- COMENTARIO: fail-fast (validar fields antes de tocar el repo) — no gastamos round-trip si el request es inválido.

### list_products.go
- struct ListProductsUseCase + constructor.
- struct ListProductsResult { Items []domain.Product; Total, Page, Size int } con json tags.
- Execute(page, size int):
  - Defaults: page<1 → 1, size<1 → 20, size>100 → 100 (cap defensivo contra DoS).
  - Convertir page→offset = (page-1)*size para hablar con el repo.
- COMENTARIO: por qué cap defensivo de 100.

### get_product.go
- struct GetProductUseCase + constructor.
- Execute(id string):
  - Si id vacío → ErrEmptyIDs.
  - Sino → repo.FindByID(id) (passthrough).
- COMENTARIO: por qué un use case "trivial" tiene valor (simetría arquitectónica + punto de extensión futuro: cache, métricas, auth).

### list_categories.go
- struct ListCategoriesUseCase + constructor.
- Execute() retorna repo.Categories().

### mock_repository_test.go
- struct mockRepository (privado del package de test) con campos func: findByIDFn, findByIDsFn, listFn, categoriesFn.
- Implementa los 4 métodos del port. CADA método nil-checkea su func ANTES de invocarlo y panica con un mensaje descriptivo si está nil ("mockRepository.X called but XFn is nil — check test setup"). Razón: si un test instancia el mock sin setear el func y el use case lo invoca por error, queremos un mensaje claro, no un nil-pointer panic críptico.
- Helper sampleProducts() devolviendo 3 productos: 2 smartphones + 1 book.
- COMENTARIO: por qué mock manual (no gomock — overkill para 4 métodos) y por qué el nil-check.

### *_test.go (uno por use case)
Tests mínimos pero exigentes:

compare_products_test.go:
  - Happy path con projection (verifica que solo aparezcan los fields pedidos).
  - Sin fields → projection completa.
  - ids vacío → ErrEmptyIDs.
  - field inválido → ErrInvalidField.
  - Error del repo (MissingIDsError) propagado y errors.Is(err, ErrProductNotFound) sigue funcionando.
  - Validación de fields ocurre ANTES de llamar al repo (mock con flag `called=false`, verificar que sigue false si validación falla).

list_products_test.go:
  - Defaults aplicados (page=1, size=20) cuando se pasa 0,0.
  - Conversión page→offset (page=3, size=10 → offset=20).
  - Cap a 100 cuando se pide 10000.
  - Total propagado.

get_product_test.go: happy + ErrEmptyIDs + ErrProductNotFound.

list_categories_test.go: passthrough simple.

VALIDACIÓN: `go test ./internal/application/... -v` → 14 tests verde.
```

**Validación local**:
```bash
go test ./internal/application/... -v
```

---

## Prompt 3 — Persistence (JSON in-memory) + fixture

```
CONTEXTO: ahora el adapter del port. Implementa domain.ProductRepository sobre un archivo JSON cargado en memoria al boot.

### internal/infrastructure/persistence/json_repository.go
- package persistence
- struct JSONRepository:
    mu       sync.RWMutex
    products []domain.Product
    byID     map[string]domain.Product   // index O(1)
- Constructor NewJSONRepository(path string) (*JSONRepository, error):
  - os.ReadFile(path) — error wrapeado con %w si falla.
  - json.Unmarshal — error wrapeado.
  - Detectar IDs duplicados durante la indexación → error explícito (no overwrite silencioso).
- Métodos del port:
  - FindByID: byID lookup → ErrProductNotFound si no.
  - FindByIDs: itera ids del input PRESERVANDO ORDEN, junta missing → si len(missing)>0 devuelve *MissingIDsError, sino la slice de productos.
  - List: respeta offset/limit, si offset >= total devuelve slice vacío + total. DEFENSIVE COPY del slice interno para no exponer el array original.
  - Categories: set + sort.Strings para output determinístico.
- TODOS los métodos toman r.mu.RLock() (defer Unlock). Aunque hoy no se escribe, el lock está listo para futuras escrituras.
- COMENTARIOS:
  * Por qué cargar al boot vs leer por request (I/O — trade-off: cambios runtime requieren restart).
  * Por qué RWMutex aunque hoy nadie escriba (preparado, free en lecturas concurrentes).
  * Por qué fail-fast en duplicados (servicio sin catálogo coherente NO debe arrancar).
  * Defensive copy en List (no exponer slice interno).

### testdata/products.json
Array JSON con 8 productos heterogéneos:
- 3 smartphones (ej: iPhone 15 Pro, Galaxy S24 Ultra, Pixel 8 Pro) con specs ricos: brand, model, battery, ram, storage, camera, os, screen.
- 2 libros (Clean Code, Pragmatic Programmer) con specs: author, publisher, pages, language, isbn, year.
- 2 prendas con size, color y specs: material, care, origin.
- 1 electrodoméstico (cafetera) con specs: brand, model, power, voltage, warranty.
Cada uno con: id (string, "1" a "8"), name, description, image_url, price, rating, category. Opcionales según la categoría.

### json_repository_test.go
Tests de integración con archivo temporal (t.TempDir()):
- Carga válida con N productos.
- Falla en archivo inexistente.
- Falla en JSON inválido.
- Falla en IDs duplicados.
- FindByID found / not found.
- FindByIDs preserva ORDEN del input ([3,1] devuelve [3,1]).
- FindByIDs con missing devuelve *MissingIDsError con la lista.
- List paginado.
- List con offset fuera de rango → empty slice + total correcto (no error).
- Categories distintas y ORDENADAS alfabéticamente.

VALIDACIÓN: `go test ./internal/infrastructure/persistence/... -v` → 11 tests verde.
```

**Validación local**:
```bash
go test ./internal/infrastructure/persistence/... -v
```

---

## Prompt 4 — HTTP layer (handlers Gin con TDD)

```
CONTEXTO: capa más externa. Adapta HTTP a use cases. NO contiene lógica de negocio — solo plumbing.

### internal/infrastructure/http/dto.go
- package http (cuidado, hay colisión con net/http — usar alias en imports cuando haga falta)
- struct errorResponse { Error string `json:"error"`; MissingIDs []string `json:"missing_ids,omitempty"` }
- struct healthResponse { Status string `json:"status"` }
- COMENTARIO: formato uniforme de error para que el frontend parsee consistentemente.

### internal/infrastructure/http/error_handler.go
- Función writeError(c *gin.Context, err error):
  1. errors.As para *domain.MissingIDsError → 404 con Error + MissingIDs en body.
  2. errors.Is(err, ErrProductNotFound) → 404.
  3. errors.Is(err, ErrEmptyIDs|ErrInvalidField|ErrInvalidPagination) → 400.
  4. Default → 500 con mensaje GENÉRICO ("internal server error"). NUNCA exponer err.Error() — evita filtrar internals.
- COMENTARIO: por qué centralizar el mapeo (un solo punto de cambio cuando se agreguen errores).

### internal/infrastructure/http/product_handler.go
- struct ProductHandler con los 4 use cases inyectados.
- Constructor NewProductHandler(compare, list, get, categories).
- Métodos:
  - Health → 200 {"status":"ok"}
  - Compare: parsea ids y fields del query (CSV). DEDUPLICAR ids preservando orden ANTES de invocar el use case (comparar X con X no aporta — y devolver duplicados confunde). Invoca use case, mapea error.
  - List: parsea page/size con validación strict — si el cliente PASA el query param con valor inválido (no numérico, < 1) retornar 400 con domain.ErrInvalidPagination wrapeado. Si NO lo pasa, aplicar default (1 / 20).
  - Get: c.Param("id"), invoca use case.
  - Categories: invoca use case y retorna {"categories": [...]}
- Helpers privados:
  - `splitCSV(raw string) []string`: parsea CSV ignorando espacios y vacíos.
  - `dedupPreservingOrder(in []string) []string`: O(n) con set auxiliar, mantiene orden de primera aparición.
  - `parsePositiveInt(c *gin.Context, name string, defaultValue int) (int, error)`: lee query param numérico positivo, error si presente pero inválido.

### internal/infrastructure/http/router.go
- Función NewRouter(h *ProductHandler) *gin.Engine.
- Registra:
    GET /health                  → Health
    GET /products                 → List
    GET /products/compare         → Compare
    GET /products/categories      → Categories
    GET /products/:id             → Get
- ATENCIÓN: las rutas /compare y /categories DEBEN registrarse ANTES que /:id, sino Gin las matchea como id="compare" o id="categories". Comentá esto.

### product_handler_test.go
Tests httptest e2e (sin mocks — repo real con fixture en t.TempDir()):
Helper setupRouter(t) que: arma fixture JSON, crea repo real, instancia los 4 use cases, arma handler, devuelve router.
Helper doRequest(t, router, method, url) → (*ResponseRecorder, map[string]any).

Tests:
- /health → 200 status=ok.
- /products default → 200 con total y items.
- /products/:id found → 200.
- /products/:id NOT found → 404 con error en body.
- /products/categories → 200 con categorías.
- /products/compare?ids=1,2&fields=name,price → 200, items con SOLO name y price (verificar que category NO aparece).
- /products/compare?ids=1,99 → 404 con missing_ids=["99"].
- /products/compare (sin ids) → 400.
- /products/compare?ids=1&fields=cost_price → 400.
- /products/compare?ids=1 (sin fields) → 200, items con todos los campos del producto.
- /products/compare?ids=1,1,2 → 200 con SOLO 2 items (dedup).
- /products?page=999 → 200 con items=[] y total real (offset out-of-range NO es error).
- /products?page=-1 → 400 (validación strict de query params).
- /products?size=abc → 400 (no numérico).

VALIDACIÓN: `go test ./internal/infrastructure/http/... -v` → 14 tests verde.
```

**Validación local**:
```bash
go test ./internal/infrastructure/http/... -v
```

---

## Prompt 5 — Composition root

```
CONTEXTO: último archivo. cmd/api/main.go arma todas las dependencias (DI manual) y arranca el server.

### cmd/api/main.go
- package main
- main():
  1. Leer PRODUCTS_FILE de env (default "testdata/products.json").
  2. Validar el path con validateProductsPath() — rechaza si está vacío, no termina en .json, o tiene segmentos ".." (defensa básica contra typos / configs malas que apunten a archivos no deseados).
  3. Cargar repo (persistence.NewJSONRepository) — si falla, log.Fatalf (fail-fast: servicio sin catálogo NO arranca).
  4. log.Printf de éxito de carga.
  5. Instanciar los 4 use cases con el mismo repo.
  6. Crear handler con los 4 use cases.
  7. Crear router con el handler.
  8. Leer PORT de env (default "8080").
  9. router.Run(":" + port). log.Fatalf si falla.
- Helper portFromEnv() para mantener main() limpio.
- Helper validateProductsPath(path string) error: chequea no-vacío, sufijo ".json", y filepath.Clean no devuelve algo que empiece con "..".
- COMENTARIO en el package: por qué DI manual (10 líneas, claro, sin generación de código).
- COMENTARIO en main(): por qué fail-fast en carga del catálogo.

VALIDACIÓN:
1. `go build ./cmd/api` → compila.
2. `go test ./... -cover` → 45 tests verde, cobertura > 80%.
3. `go vet ./...` → sin warnings.
```

**Validación local**:
```bash
go build ./cmd/api
go test ./... -cover
go vet ./...
```

---

## Prompt 6 — Smoke test end-to-end

```
CONTEXTO: tests pasaron. Ahora validamos que el binario corra y los endpoints respondan correctamente.

TAREA: arrancar el server en background y correr 9 curls verificando status code y body. Pegá esto en una terminal del editor:

go build -o /tmp/api ./cmd/api && /tmp/api &
sleep 2

echo "=== HEALTH ==="
curl -sw "\nHTTP %{http_code}\n" http://localhost:8080/health

echo "=== LIST ==="
curl -sw "\nHTTP %{http_code}\n" "http://localhost:8080/products?page=1&size=2"

echo "=== GET FOUND ==="
curl -sw "\nHTTP %{http_code}\n" http://localhost:8080/products/1

echo "=== GET NOT FOUND ==="
curl -sw "\nHTTP %{http_code}\n" http://localhost:8080/products/9999

echo "=== CATEGORIES ==="
curl -sw "\nHTTP %{http_code}\n" http://localhost:8080/products/categories

echo "=== COMPARE OK ==="
curl -sw "\nHTTP %{http_code}\n" "http://localhost:8080/products/compare?ids=1,2,3&fields=name,price,rating"

echo "=== COMPARE EMPTY IDS (400) ==="
curl -sw "\nHTTP %{http_code}\n" "http://localhost:8080/products/compare"

echo "=== COMPARE MISSING ID (404 + missing_ids) ==="
curl -sw "\nHTTP %{http_code}\n" "http://localhost:8080/products/compare?ids=1,99"

echo "=== COMPARE INVALID FIELD (400) ==="
curl -sw "\nHTTP %{http_code}\n" "http://localhost:8080/products/compare?ids=1&fields=cost_price"

pkill -f /tmp/api

ESPERADOS:
- /health → 200 {"status":"ok"}
- /products → 200 con items + total
- /products/1 → 200 con producto completo
- /products/9999 → 404 {"error":"product not found"}
- /products/categories → 200 {"categories":[...]}
- /products/compare con projection → 200 SOLO con name/price/rating
- /products/compare sin ids → 400 ErrEmptyIDs
- /products/compare?ids=1,99 → 404 con missing_ids:["99"]
- /products/compare?ids=1&fields=cost_price → 400 ErrInvalidField
```

---

## Prompt 7 — Documentación

```
CREÁ:

### problem.md
Pegá EXACTAMENTE el contenido de problem.md del repo local (ya está en español, no lo regeneres).

### README.md
README profesional en español con:
- Título + 1 línea de descripción + tagline ("Clean Architecture liviana + TDD estricto").
- Sección Stack (tabla con Lenguaje/Framework/Persistencia/Testing y por qué).
- Sección Arquitectura con diagrama ASCII de las 3 capas + flujo de dependencias (regla: capas externas conocen internas, NUNCA al revés).
- Estructura de directorios.
- Tabla de Endpoints (método, path, descripción).
- Sección Errores (tabla de status codes).
- Decisión 404 todo-o-nada en /compare con razón.
- Modelo de datos (struct Product) con explicación de Specs map[string]any.
- Setup (requisitos + comandos).
- Tabla de DECISIONES ARQUITECTÓNICAS con trade-offs aceptados.
- Cobertura: 41 tests / 5 packages (domain 6, application 14, persistence 11, http 10).
- "Lo que NO está incluido (out of scope intencional)": auth, rate limit, cache, observabilidad, DB real, CI/CD. Aclarar que el diseño los admite sin reescribir.
- "Cómo extender": tabla con "quiero agregar X / toco Y".

NO inventes features que no estén en el código. Si no estás seguro, releé los archivos.

### HACKERRANK.md
Solo si la plataforma lo pide — guía de copy-paste por bloques (no es necesario para el evaluador).

VALIDACIÓN: leé los 3 archivos y confirmá que README.md tenga al menos 8 secciones bien diferenciadas.
```

---

## Checklist final antes de submit en HackerRank

- [ ] `go test ./... -cover` → 45 tests verde
- [ ] `go vet ./...` sin warnings
- [ ] `go build ./cmd/api` compila
- [ ] `go run ./cmd/api` arranca y responde curl
- [ ] Smoke test del Prompt 6 pasa todos los casos esperados
- [ ] README.md presente con setup + endpoints + decisiones
- [ ] problem.md en español (no spanglish)
- [ ] Comentarios en código explican WHY (no QUÉ)
- [ ] Estructura de directorios coincide con el repo local

## Si el agente desvía — frases para frenarlo

| Síntoma | Frase de freno |
|---|---|
| Cambia el package layout | "no cambies la estructura de packages. Volvé al layout original: domain/application/infrastructure." |
| Quiere agregar features no pedidas | "no agregues funcionalidad que no pedí. Borrá X y dejá solo lo del prompt." |
| Mete frameworks de DI / mocking libs | "no uses Wire/Fx/gomock. DI manual y mocks manuales — esa es la decisión." |
| Comentarios que explican QUÉ hace el código | "los comentarios explican WHY, no WHAT. Reescribí los comentarios siguiendo esa regla." |
| Cambia los nombres de los endpoints | "no cambies las rutas. Son /health, /products, /products/compare, /products/categories, /products/:id." |
| Pone validaciones en domain que pertenecen a application | "domain solo tiene lógica de entity. Las validaciones de fields whitelist se hacen en application — moveselas." |
| Devuelve 200 con productos parciales en /compare | "ESO ESTÁ MAL. Si falta UN solo id en /compare es 404 con missing_ids en el body. NO 200 parcial." |

## Tip final
Cuando el agente termine cada prompt, **pedile que liste los archivos creados/modificados** con un comando único (ej: `git status` o `find . -type f -newer go.mod`). Eso te ayuda a auditar rápido sin leer todo el output.
