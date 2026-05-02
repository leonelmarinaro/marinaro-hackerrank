# Prompts para reconstruir el proyecto en HackerRank con un agente IA

> **Cómo usar**: pegá el **System Prompt** UNA VEZ al inicio de la sesión, después pegá los prompts en orden (0 → 10). Después de cada uno, corré la validación. Si pasa, seguís. Si falla, le pegás el output del error al agente.

---

## System Prompt (pegar UNA SOLA VEZ al inicio)

```
Sos un agente que sigue instrucciones al pie. Vas a construir un backend REST API en Go conmigo, paso a paso.

REGLAS GLOBALES (no negociables, aplican a TODOS los prompts):

1. NO agregues funcionalidad que no te pedí explícitamente. Si dudás, preguntá.
2. NO uses frameworks de DI (Wire, Fx) ni libs de mocking (gomock, testify/mock). Hacelo a mano.
3. NO cambies nombres de archivos, paths, ni estructura de directorios.
4. Comentarios en código: SIEMPRE explican POR QUÉ (decisión, trade-off). NUNCA explican QUÉ hace el código (el nombre de la función ya lo dice).
5. Tests primero (TDD), implementación después. Cuando te pida tests, escribilos antes que la impl.
6. Mantené las decisiones arquitectónicas tomadas. Si te parece que algo se podría hacer "mejor", PREGUNTÁ antes de cambiarlo.
7. Después de cada prompt, listame los archivos que creaste/modificaste con un comando único (ej: `git status` o `find . -name "*.go" -newer go.mod`).
8. Si algo del prompt es ambiguo, preguntame ANTES de improvisar.

DECISIONES ARQUITECTÓNICAS (no las cuestiones):
- Stack: Go ≥ 1.21 + Gin
- Persistencia: archivo JSON cargado in-memory al boot (sin DB real)
- Arquitectura: Clean Architecture liviana en 3 capas
    domain          → entidades + interfaces. SIN dependencias externas.
    application     → use cases. Solo depende de domain.
    infrastructure  → adapters (HTTP handlers, repo JSON). Depende de application + domain.
- Composition root en cmd/api/main.go (DI manual, sin frameworks)
- Modelo Product: campos comunes tipados + Specs como map[string]any (flexible por categoría)
- Tests: stdlib `testing` + `httptest` + mocks manuales
- Logs: log/slog (stdlib desde Go 1.21) en JSON
- Lifecycle: HTTP server con timeouts explícitos + graceful shutdown ante SIGTERM/SIGINT

Confirmá que entendiste estas reglas antes de empezar.
```

---

## Prompt 0 — Setup del proyecto

```
TAREA: setup inicial. Hacé EXACTAMENTE esto, en orden:

1. Inicializar el módulo Go:
   go mod init github.com/candidato/meli-comparison-api

   (Si HackerRank fija otro module path, usá ese y avisame.)

2. Crear esta estructura de directorios EXACTA (vacíos por ahora):
   cmd/api/
   internal/domain/
   internal/application/
   internal/infrastructure/persistence/
   internal/infrastructure/http/
   testdata/

3. Instalar la única dependencia externa:
   go get github.com/gin-gonic/gin

4. NO escribas código de aplicación todavía.

VALIDACIÓN — corré esto y mostrame el output:
   go env GOMOD
   ls cmd/api internal/domain internal/application internal/infrastructure/persistence internal/infrastructure/http testdata
   cat go.mod

ESPERADO: el path al go.mod, los 6 directorios listados, go.mod con `require github.com/gin-gonic/gin`.

NO AVANCES al siguiente prompt hasta que esto pase.
```

---

## Prompt 1 — Domain layer (entity + errors + repository) con TDD

```
CAPA ACTUAL: domain (la más interna). NO depende de nada — solo stdlib.

CREÁ EXACTAMENTE estos 4 archivos en internal/domain/.

═══════════════════════════════════════════════════════════════════════════
ARCHIVO 1: internal/domain/product.go
═══════════════════════════════════════════════════════════════════════════

PACKAGE: domain

CONTENIDO:

1. Tipo `Product` (struct público) con estos campos EXACTOS y JSON tags:
   - ID          string         `json:"id"`
   - Name        string         `json:"name"`
   - Description string         `json:"description"`
   - ImageURL    string         `json:"image_url"`
   - Price       float64        `json:"price"`
   - Rating      float64        `json:"rating"`
   - Category    string         `json:"category"`
   - Size        string         `json:"size,omitempty"`
   - Weight      float64        `json:"weight,omitempty"`
   - Color       string         `json:"color,omitempty"`
   - Specs       map[string]any `json:"specs,omitempty"`

2. Variable PRIVADA `allowedFields` de tipo `map[string]struct{}` con ESTAS claves
   (whitelist de campos seleccionables vía API):
   id, name, description, image_url, price, rating, category, size, weight, color, specs

3. Función PÚBLICA `IsAllowedField(name string) bool` que retorna si el campo
   está en la whitelist. Case-sensitive.

4. Método PÚBLICO `(p Product) SelectFields(fields []string) map[string]any`:
   - Si fields está vacío/nil → retorna TODO el producto como map.
   - Si fields tiene valores → retorna SOLO los pedidos que existan en la whitelist.
   - Campos opcionales en cero (Size="", Weight=0, Color="", Specs nil/empty) NO
     deben aparecer en el output, ni siquiera con SelectFields(nil).

5. Método PRIVADO `(p Product) toMap() map[string]any` para uso interno
   (genera el map base que SelectFields consume).

COMENTARIOS REQUERIDOS (en este orden):
- Doc-comment del package: rol del paquete + regla de no-dependencias.
- Doc-comment sobre Product: explicar POR QUÉ Specs es map[string]any
  (extensibilidad por categoría > type-safety en compile time).
- Doc-comment sobre allowedFields: POR QUÉ whitelist y no blacklist
  (seguridad por defecto: campo interno nuevo no se filtra).
- Doc-comment sobre SelectFields: POR QUÉ retorna map y no Product proyectado
  (el requerimiento es OMITIR campos no pedidos, no mostrarlos como null).

═══════════════════════════════════════════════════════════════════════════
ARCHIVO 2: internal/domain/errors.go
═══════════════════════════════════════════════════════════════════════════

PACKAGE: domain
IMPORTS: errors, fmt, strings

CONTENIDO:

1. Sentinel errors PÚBLICOS:
   ErrProductNotFound    = errors.New("product not found")
   ErrInvalidField       = errors.New("invalid field requested")
   ErrEmptyIDs           = errors.New("at least one id is required")
   ErrInvalidPagination  = errors.New("invalid pagination parameters")
   ErrTooManyIDs         = errors.New("too many ids requested")

2. Constante PÚBLICA: `MaxCompareIDs = 50`
   Comentario: cap para /products/compare. Defensa anti-DoS por amplificación
   y guard de UX (nadie compara 50+ items razonablemente).

3. Tipo PÚBLICO `MissingIDsError` con UN campo:
   Missing []string

4. Métodos de MissingIDsError:
   - Error() string: retorna `"products not found: " + strings.Join(e.Missing, ", ")` (usá fmt.Sprintf).
   - Unwrap() error: retorna ErrProductNotFound.

COMENTARIOS REQUERIDOS:
- Sobre el bloque de sentinel errors: POR QUÉ patrón sentinel (errors.Is desacopla
  el mapeo error→status del texto del mensaje).
- Sobre MissingIDsError: POR QUÉ carga datos en vez de ser sentinel
  (cliente necesita saber QUÉ falta para corregir el request).
- Sobre Unwrap: POR QUÉ existe (permite errors.Is(missingErr, ErrProductNotFound)
  para que handlers traten ambos casos como 404).

═══════════════════════════════════════════════════════════════════════════
ARCHIVO 3: internal/domain/repository.go
═══════════════════════════════════════════════════════════════════════════

PACKAGE: domain
SIN imports.

CONTENIDO: interface PÚBLICA `ProductRepository` con EXACTAMENTE estos 4 métodos:

   FindByID(id string) (*Product, error)
   FindByIDs(ids []string) ([]Product, error)
   List(offset, limit int) ([]Product, int, error)
   Categories() ([]string, error)

CONTRATOS (documentar en doc-comment de cada método):
- FindByID: retorna ErrProductNotFound si no existe.
- FindByIDs: si algún ID falta retorna *MissingIDsError con la lista. Mantiene
  el ORDEN del input.
- List: offset 0-based, limit>0. Retorna (página, total del catálogo, error).
- Categories: distintas, ORDENADAS alfabéticamente (output determinístico).

COMENTARIO REQUERIDO sobre la interface: POR QUÉ vive en domain y no en
application (Inversión de Dependencias: domain DEFINE lo que necesita, infra
lo IMPLEMENTA — permite testear con mocks y cambiar adapter sin tocar
lógica de negocio).

═══════════════════════════════════════════════════════════════════════════
ARCHIVO 4: internal/domain/product_test.go
═══════════════════════════════════════════════════════════════════════════

PACKAGE: domain (mismo package, no _test)
IMPORTS: errors, testing

TESTS REQUERIDOS (escribilos ANTES de validar, TDD):

1. TestIsAllowedField — table-driven con AL MENOS estos casos:
   - "name", "price", "specs", "image_url" → true
   - "cost_price", "internal_sku" → false (campos hipotéticos internos)
   - "" → false
   - "NAME" → false (case-sensitive intencional)

2. TestSelectFields_FullProductWhenFieldsEmpty:
   - SelectFields(nil) sobre un producto sample completo.
   - Verificar que id, name están presentes.
   - Verificar que specs está presente cuando tiene valor.

3. TestSelectFields_SubsetReturnsOnlyRequested:
   - SelectFields([]string{"name", "price"}).
   - Verificar len==2, name está, description NO está, specs NO está.

4. TestSelectFields_UnknownFieldSilentlyIgnored:
   - SelectFields([]string{"name", "totally_invented"}).
   - "totally_invented" NO debe estar en el output.
   - "name" SÍ debe estar.

5. TestSelectFields_OptionalZeroFieldsOmitted:
   - Product minimal con solo ID y Name.
   - SelectFields(nil).
   - Ningún campo opcional (size, weight, color, specs) debe aparecer.

6. TestMissingIDsError_UnwrapToProductNotFound:
   - errors.Is(missErr, ErrProductNotFound) debe ser true.
   - Error() no debe estar vacío.

HELPER privado `newSampleProduct()` con un iPhone 15 (ID="p1", Name="iPhone 15",
Price=1299.99, Rating=4.7, Category="smartphones", Color="black", Weight=0.171,
Specs={battery:"3349mAh", ram:"8GB", storage:"256GB", os:"iOS 17"}).

═══════════════════════════════════════════════════════════════════════════
VALIDACIÓN
═══════════════════════════════════════════════════════════════════════════

   go test ./internal/domain/... -v

ESPERADO: 6 tests verde. Si algún test falla, NO sigas — corregí y mostrame.

LISTAME los archivos creados con: ls -la internal/domain/
```

---

## Prompt 2 — Application layer (4 use cases + mock + tests)

```
PRECONDICIÓN: domain debe estar verde (6 tests pasando).

CAPA ACTUAL: application. Solo depende de domain. Cada use case recibe el repo
por constructor (DI manual).

═══════════════════════════════════════════════════════════════════════════
ARCHIVO 1: internal/application/compare_products.go
═══════════════════════════════════════════════════════════════════════════

PACKAGE: application
IMPORTS: fmt, github.com/<TU-MODULE>/internal/domain

CONTENIDO:

1. Doc-comment del package: rol (use cases, depende solo de domain, sin HTTP).

2. struct PÚBLICA `CompareProductsUseCase` con UN campo privado:
   repo domain.ProductRepository

3. Constructor `NewCompareProductsUseCase(repo domain.ProductRepository) *CompareProductsUseCase`.

4. struct PÚBLICA `CompareProductsResult`:
   Items  []map[string]any `json:"items"`
   Fields []string         `json:"fields"`

5. Método PÚBLICO `Execute(ids, fields []string) (*CompareProductsResult, error)`
   con esta secuencia EXACTA de validaciones (fail-fast):

   a. Si len(ids)==0 → return nil, domain.ErrEmptyIDs.

   b. ids = dedupPreservingOrder(ids)  ← DEDUP en use case, NO en handler.

   c. Si len(ids) > domain.MaxCompareIDs → return wrap:
      fmt.Errorf("%w: got %d, max %d", domain.ErrTooManyIDs, len(ids), domain.MaxCompareIDs)

   d. Para cada f in fields, si !domain.IsAllowedField(f) → return wrap:
      fmt.Errorf("%w: %q", domain.ErrInvalidField, f)

   e. products, err := uc.repo.FindByIDs(ids). Si err != nil → return nil, err.

   f. Construir Items con p.SelectFields(fields) para cada p en products.

   g. Return &CompareProductsResult{Items, Fields}, nil.

6. Función PRIVADA `dedupPreservingOrder(in []string) []string`:
   - Si len(in) <= 1 → return in.
   - Set auxiliar map[string]struct{}, slice de salida con orden de primera aparición.

COMENTARIOS REQUERIDOS:
- Sobre Execute: la secuencia EXACTA de validaciones y POR QUÉ fail-fast
  (no gastamos round-trip al repo si el request es inválido).
- Sobre dedup en use case (NO en handler): cualquier adapter (HTTP, gRPC, CLI)
  recibe la misma semántica.
- Sobre el cap MaxCompareIDs: defensa anti-DoS + guard UX.
- Sobre "todo o nada" en (e): comparar parcialmente engaña al cliente.

═══════════════════════════════════════════════════════════════════════════
ARCHIVO 2: internal/application/list_products.go
═══════════════════════════════════════════════════════════════════════════

CONTENIDO:

1. struct `ListProductsUseCase` con campo `repo domain.ProductRepository`.

2. Constructor `NewListProductsUseCase`.

3. struct `ListProductsResult`:
   Items []domain.Product `json:"items"`
   Total int              `json:"total"`
   Page  int              `json:"page"`
   Size  int              `json:"size"`

4. Método `Execute(page, size int) (*ListProductsResult, error)` con esta lógica:
   - Si page < 1 → page = 1 (default lenient como salvaguarda; el handler hace strict).
   - Si size < 1 → size = 20 (default).
   - Si size > 100 → size = 100 (cap defensivo anti-DoS).
   - offset := (page - 1) * size.
   - products, total, err := uc.repo.List(offset, size).
   - Return &ListProductsResult{products, total, page, size}, nil.

COMENTARIO REQUERIDO sobre Execute: POR QUÉ cap defensivo de 100 (DoS por
size=999999) y POR QUÉ paginación obligatoria desde el día 1.

═══════════════════════════════════════════════════════════════════════════
ARCHIVO 3: internal/application/get_product.go
═══════════════════════════════════════════════════════════════════════════

CONTENIDO:

1. struct `GetProductUseCase` + constructor.
2. Método `Execute(id string) (*domain.Product, error)`:
   - Si id == "" → return nil, domain.ErrEmptyIDs.
   - Sino → return uc.repo.FindByID(id).

COMENTARIO REQUERIDO: POR QUÉ un use case "trivial" tiene valor (simetría
arquitectónica + punto de extensión: cache, métricas, auth futuras).

═══════════════════════════════════════════════════════════════════════════
ARCHIVO 4: internal/application/list_categories.go
═══════════════════════════════════════════════════════════════════════════

CONTENIDO:

1. struct `ListCategoriesUseCase` + constructor.
2. Método `Execute() ([]string, error)`: pass-through a uc.repo.Categories().

═══════════════════════════════════════════════════════════════════════════
ARCHIVO 5: internal/application/mock_repository_test.go
═══════════════════════════════════════════════════════════════════════════

PACKAGE: application
IMPORTS: github.com/<TU-MODULE>/internal/domain

CONTENIDO:

1. struct PRIVADA `mockRepository` con 4 campos func:
   findByIDFn   func(id string) (*domain.Product, error)
   findByIDsFn  func(ids []string) ([]domain.Product, error)
   listFn       func(offset, limit int) ([]domain.Product, int, error)
   categoriesFn func() ([]string, error)

2. Implementa los 4 métodos del port. CADA UNO debe NIL-CHECKEAR su func ANTES
   de invocar y panic con mensaje DESCRIPTIVO si está nil:

   func (m *mockRepository) FindByID(id string) (*domain.Product, error) {
       if m.findByIDFn == nil {
           panic("mockRepository.FindByID called but findByIDFn is nil — check test setup")
       }
       return m.findByIDFn(id)
   }

   (Hacé lo mismo para los otros 3 métodos.)

3. Helper `sampleProducts() []domain.Product` con 3 productos:
   - {ID:"1", Name:"iPhone 15", Category:"smartphones", Price:1299.99, Rating:4.7, Specs:{ram:"8GB"}}
   - {ID:"2", Name:"Galaxy S24", Category:"smartphones", Price:1199.99, Rating:4.6, Specs:{ram:"12GB"}}
   - {ID:"3", Name:"Clean Code", Category:"books", Price:35.00, Rating:4.8}

COMENTARIO REQUERIDO: POR QUÉ mock manual (no gomock — overkill para 4 métodos)
y POR QUÉ el nil-check (mensaje claro vs nil-pointer panic críptico).

═══════════════════════════════════════════════════════════════════════════
ARCHIVO 6: internal/application/compare_products_test.go
═══════════════════════════════════════════════════════════════════════════

TESTS REQUERIDOS (TDD — escribilos ANTES de probar):

1. TestCompareProducts_HappyPath:
   - Mock con findByIDsFn que devuelve sampleProducts()[:2].
   - Execute([]string{"1","2"}, []string{"name","price"}).
   - len(res.Items)==2, "name" presente, "category" NO presente.

2. TestCompareProducts_FullProjectionWhenFieldsEmpty:
   - Execute([]string{"1"}, nil) → "category" SÍ debe estar.

3. TestCompareProducts_EmptyIDsRejected:
   - Execute(nil, nil) → errors.Is(err, ErrEmptyIDs).

4. TestCompareProducts_InvalidFieldRejected:
   - Execute([]string{"1"}, []string{"name","cost_price"}) → errors.Is(err, ErrInvalidField).

5. TestCompareProducts_RepoErrorPropagated:
   - Mock que devuelve *MissingIDsError{Missing:["99"]}.
   - errors.Is(err, ErrProductNotFound) y errors.As a *MissingIDsError ambos true.

6. TestCompareProducts_DeduplicatesIDs:
   - Execute([]string{"1","1","2","1"}, nil).
   - Capturar los ids que recibió el mock con un closure.
   - Verificar que el mock recibió EXACTAMENTE [1, 2] preservando orden.

7. TestCompareProducts_RejectsTooManyIDs:
   - Generar MaxCompareIDs+1 ids ÚNICOS (sino el dedup los reduciría).
     Tip: usar `string(rune('a'+i%26)) + string(rune('0'+i/26))` en un loop.
   - errors.Is(err, ErrTooManyIDs).

8. TestCompareProducts_DoesNotCallRepoWhenValidationFails:
   - Mock con flag `called=false` que setea true al invocar findByIDsFn.
   - Execute([]string{"1"}, []string{"invalid_field"}).
   - Verificar que called sigue false (validación es fail-fast).

═══════════════════════════════════════════════════════════════════════════
ARCHIVO 7: internal/application/list_products_test.go
═══════════════════════════════════════════════════════════════════════════

TESTS:

1. TestListProducts_DefaultPagination — Execute(0, 0): defaults page=1, size=20.
2. TestListProducts_PageToOffsetConversion — Execute(3, 10) → mock recibe offset=20.
3. TestListProducts_SizeCappedAt100 — Execute(1, 10000) → size se clampea a 100.
4. TestListProducts_ReturnsTotalForPagination — mock devuelve total=42, res.Total==42.

═══════════════════════════════════════════════════════════════════════════
ARCHIVO 8: internal/application/get_product_test.go
═══════════════════════════════════════════════════════════════════════════

TESTS:
1. TestGetProduct_HappyPath
2. TestGetProduct_EmptyIDRejected → ErrEmptyIDs
3. TestGetProduct_NotFoundPropagated → ErrProductNotFound

═══════════════════════════════════════════════════════════════════════════
ARCHIVO 9: internal/application/list_categories_test.go
═══════════════════════════════════════════════════════════════════════════

UN test:
TestListCategories_PassthroughToRepo — mock devuelve ["books","smartphones"], verificar.

═══════════════════════════════════════════════════════════════════════════
VALIDACIÓN
═══════════════════════════════════════════════════════════════════════════

   go test ./internal/application/... -v

ESPERADO: 16 tests verde (8 compare + 4 list + 3 get + 1 categories).

ANTI-PATTERN: si el agente quiere agregar testify/assert o gomock, FRENALO.
Stdlib es suficiente.

LISTAME los archivos creados.
```

---

## Prompt 3 — Persistence (JSON in-memory) + fixture

```
PRECONDICIÓN: domain (6 tests) + application (16 tests) verde.

CAPA ACTUAL: infrastructure/persistence. Implementa el port domain.ProductRepository.

═══════════════════════════════════════════════════════════════════════════
ARCHIVO 1: testdata/products.json
═══════════════════════════════════════════════════════════════════════════

Array JSON con 8 productos. RESPETÁ esta heterogeneidad (importante para
demostrar que el modelo soporta categorías distintas):

- 3 SMARTPHONES (ids "1","2","3"): iPhone 15 Pro, Samsung Galaxy S24 Ultra,
  Google Pixel 8 Pro. Specs ricos: brand, model, battery, ram, storage,
  camera, os, screen.
- 2 LIBROS (ids "4","5"): Clean Code, The Pragmatic Programmer.
  Specs: author, publisher, pages, language, isbn, year.
- 2 PRENDAS (ids "6","7"): remera blanca, buzo con capucha.
  Tienen size y color a nivel root + specs: material, care, origin.
- 1 ELECTRODOMÉSTICO (id "8"): cafetera espresso DeLonghi Magnifica S.
  Specs: brand, model, power, voltage, warranty.

Cada uno con: id (string), name, description, image_url, price, rating,
category, weight, color (cuando aplique).

═══════════════════════════════════════════════════════════════════════════
ARCHIVO 2: internal/infrastructure/persistence/json_repository.go
═══════════════════════════════════════════════════════════════════════════

PACKAGE: persistence
IMPORTS: encoding/json, fmt, os, sort, sync, github.com/<TU-MODULE>/internal/domain

CONTENIDO:

1. struct PÚBLICA `JSONRepository`:
   mu       sync.RWMutex
   products []domain.Product
   byID     map[string]domain.Product   // index O(1)

2. Constructor `NewJSONRepository(path string) (*JSONRepository, error)`:
   - os.ReadFile(path) — wrapear error con fmt.Errorf("reading products file %q: %w", path, err).
   - json.Unmarshal a []domain.Product — wrapear error.
   - Iterar productos construyendo byID. Si detectás ID duplicado → return error
     explícito. NUNCA overwrite silencioso.

3. Métodos del port (TODOS toman r.mu.RLock() defer r.mu.RUnlock()):

   a. FindByID(id string) (*domain.Product, error):
      - byID lookup. Si no existe → return nil, domain.ErrProductNotFound.
      - return &p, nil.

   b. FindByIDs(ids []string) ([]domain.Product, error):
      - Iterar ids EN ORDEN del input. Para cada uno: lookup en byID, si está
        agregar a `found`, si no agregar a `missing`.
      - Si len(missing) > 0 → return nil, &domain.MissingIDsError{Missing: missing}.
      - Sino → return found, nil.

   c. List(offset, limit int) ([]domain.Product, int, error):
      - total := len(r.products).
      - Si offset >= total → return []domain.Product{}, total, nil (NO es error).
      - end := offset + limit; clampear si end > total.
      - DEFENSIVE COPY: page := make([]domain.Product, end-offset); copy(page, r.products[offset:end]).
      - Return page, total, nil.

   d. Categories() ([]string, error):
      - Set map[string]struct{} con categorías distintas.
      - Convertir a slice y sort.Strings() para output determinístico.

COMENTARIOS REQUERIDOS:
- Sobre el struct: POR QUÉ cargar al boot vs leer por request (I/O).
- Sobre RWMutex: POR QUÉ aunque hoy no se escribe (free en lecturas, listo para futuro).
- Sobre fail-fast en duplicados: servicio sin catálogo coherente NO debe arrancar.
- Sobre defensive copy en List: no exponer slice interno.
- IMPORTANTE — agregá este bloque al doc-comment del struct:

  // IMPORTANTE — Mutabilidad de Product.Specs: los métodos retornan copias
  // shallow del struct Product, pero el campo Specs (map[string]any) se
  // COMPARTE por referencia con el catálogo interno. Si un consumidor mutara
  // el map (`p.Specs["x"] = y`), corromperíamos el dataset compartido sin
  // holdear el lock. Hoy ningún caller muta — está documentado como contrato:
  // Specs es read-only para el consumidor.

═══════════════════════════════════════════════════════════════════════════
ARCHIVO 3: internal/infrastructure/persistence/json_repository_test.go
═══════════════════════════════════════════════════════════════════════════

Tests de integración con archivo TEMPORAL (t.TempDir()):

Helper `writeTempJSON(t *testing.T, content string) string`: escribe contenido
en t.TempDir()+"/products.json" y retorna el path.

Helper `setupRepo(t *testing.T) *JSONRepository`: usa el JSON de muestra
(constante `sampleJSON` con 3 productos: ids "1","2","3", categorías "x" y "y").

TESTS REQUERIDOS:

1. TestNewJSONRepository_LoadsValidFile — total==3.
2. TestNewJSONRepository_FailsOnMissingFile.
3. TestNewJSONRepository_FailsOnInvalidJSON.
4. TestNewJSONRepository_FailsOnDuplicateIDs.
5. TestFindByID_Found.
6. TestFindByID_NotFound — errors.Is(err, ErrProductNotFound).
7. TestFindByIDs_AllFound_PreservesOrder — pedir [3,1] → recibir [3,1].
8. TestFindByIDs_PartialMissingReturnsError — pedir [1,999,888] → *MissingIDsError con 2 missing.
9. TestList_Pagination — offset=1 limit=1 → 1 producto, total=3.
10. TestList_OffsetBeyondTotalReturnsEmpty — offset=100 → []  + total=3, sin error.
11. TestCategories_DistinctAndSorted — categorías ["x","y","x"] → ["x","y"] (sort).

═══════════════════════════════════════════════════════════════════════════
VALIDACIÓN
═══════════════════════════════════════════════════════════════════════════

   go test ./internal/infrastructure/persistence/... -v

ESPERADO: 11 tests verde.

LISTAME los archivos creados.
```

---

## Prompt 4 — HTTP layer parte 1: dto + error_handler + middleware

```
PRECONDICIÓN: domain + application + persistence verdes (33 tests acumulados).

CAPA ACTUAL: infrastructure/http. Adapta HTTP a use cases. NO contiene lógica
de negocio.

═══════════════════════════════════════════════════════════════════════════
ARCHIVO 1: internal/infrastructure/http/dto.go
═══════════════════════════════════════════════════════════════════════════

PACKAGE: http
SIN imports.

CONTENIDO:

1. Doc-comment del package: rol del paquete + responsabilidades (parsear req,
   invocar use case, mapear errores, serializar response).

2. struct privada `errorResponse`:
   Error      string   `json:"error"`
   MissingIDs []string `json:"missing_ids,omitempty"`

3. struct privada `healthResponse`:
   Status string `json:"status"`

COMENTARIO sobre errorResponse: formato uniforme para que el frontend parsee
consistentemente; missing_ids es opcional (solo aparece en /compare con IDs faltantes).

═══════════════════════════════════════════════════════════════════════════
ARCHIVO 2: internal/infrastructure/http/error_handler.go
═══════════════════════════════════════════════════════════════════════════

PACKAGE: http
IMPORTS: errors, net/http, github.com/gin-gonic/gin, github.com/<TU-MODULE>/internal/domain

CONTENIDO: función PRIVADA `writeError(c *gin.Context, err error)`:

   1. var miss *domain.MissingIDsError
      Si errors.As(err, &miss) → c.JSON(404, errorResponse{Error: err.Error(), MissingIDs: miss.Missing}). return.

   2. switch:
      - errors.Is(err, ErrProductNotFound) → 404 + errorResponse{Error: err.Error()}.
      - errors.Is(err, ErrEmptyIDs) || ErrInvalidField || ErrInvalidPagination || ErrTooManyIDs → 400.
      - default → 500 con mensaje GENÉRICO "internal server error".

REGLA INVIOLABLE: en el default 500, JAMÁS exponer err.Error() al cliente.

COMENTARIO REQUERIDO: POR QUÉ centralizar el mapeo (un solo punto de cambio
cuando se agreguen errores nuevos) y POR QUÉ default genérico (no filtrar internals).

═══════════════════════════════════════════════════════════════════════════
ARCHIVO 3: internal/infrastructure/http/middleware.go
═══════════════════════════════════════════════════════════════════════════

PACKAGE: http
IMPORTS: crypto/rand, encoding/hex, log/slog, time, github.com/gin-gonic/gin

CONTENIDO:

1. Constante: `requestIDHeader = "X-Request-Id"`. Comentar: convención de facto
   (Heroku, AWS, Datadog).

2. Función PÚBLICA `RequestIDMiddleware() gin.HandlerFunc`:
   - Lee header X-Request-Id entrante.
   - Si vacío → genera con newRequestID().
   - c.Set("request_id", rid); c.Writer.Header().Set(requestIDHeader, rid); c.Next().
   - Comentar: honrar header entrante = correlación cross-service.

3. Función PÚBLICA `LoggingMiddleware(logger *slog.Logger) gin.HandlerFunc`:
   - Capturar start := time.Now() ANTES de c.Next().
   - DESPUÉS de c.Next() loguear con slog:
     logger.Info("http_request",
       slog.String("request_id", c.GetString("request_id")),
       slog.String("method", c.Request.Method),
       slog.String("path", c.Request.URL.Path),
       slog.Int("status", c.Writer.Status()),
       slog.Duration("duration", time.Since(start)),
       slog.String("client_ip", c.ClientIP()),
     )
   - Comentar: POR QUÉ no usamos gin.Logger() default (texto plano sin estructura,
     riesgo de leak de query string).

4. Función PÚBLICA `SecurityHeadersMiddleware() gin.HandlerFunc`:
   - c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
   - c.Writer.Header().Set("Referrer-Policy", "no-referrer")
   - c.Next()
   - Comentar: para una API JSON pura no aplica casi nada "web", pero estos dos
     son cero coste y señal explícita de hardening.

5. Función PRIVADA `newRequestID() string`:
   - 8 bytes con crypto/rand.Read; encoding/hex.EncodeToString.
   - Si rand.Read falla (extremadamente raro), fallback a time.Now().Format("150405.000000000").

═══════════════════════════════════════════════════════════════════════════
VALIDACIÓN
═══════════════════════════════════════════════════════════════════════════

Estos 3 archivos NO tienen tests propios — se ejercitan en los tests del
handler (próximo prompt). Acá solo validamos que compilen:

   go build ./internal/infrastructure/http/...

ESPERADO: sin errores.

LISTAME los archivos creados.
```

---

## Prompt 5 — HTTP layer parte 2: product_handler + router

```
PRECONDICIÓN: prompt 4 OK (build limpio).

═══════════════════════════════════════════════════════════════════════════
ARCHIVO 1: internal/infrastructure/http/product_handler.go
═══════════════════════════════════════════════════════════════════════════

PACKAGE: http
IMPORTS: fmt, net/http, strconv, strings, github.com/gin-gonic/gin,
         github.com/<TU-MODULE>/internal/application,
         github.com/<TU-MODULE>/internal/domain

CONTENIDO:

1. struct PÚBLICA `ProductHandler` con los 4 use cases inyectados (campos
   privados):
   compareUC    *application.CompareProductsUseCase
   listUC       *application.ListProductsUseCase
   getUC        *application.GetProductUseCase
   categoriesUC *application.ListCategoriesUseCase

2. Constructor `NewProductHandler(compareUC, listUC, getUC, categoriesUC)`.

3. Método Health(c *gin.Context):
   c.JSON(200, healthResponse{Status: "ok"}).

4. Método Compare(c *gin.Context):
   ids := splitCSV(c.Query("ids"))
   fields := splitCSV(c.Query("fields"))
   res, err := h.compareUC.Execute(ids, fields)
   if err != nil { writeError(c, err); return }
   c.JSON(200, res)

   COMENTARIO: la dedup, el cap y la validación de fields ocurren en el USE CASE.
   El handler es puro transporte.

5. Método List(c *gin.Context):
   page, err := parsePositiveInt(c, "page", 1)
   if err != nil { writeError(c, err); return }
   size, err := parsePositiveInt(c, "size", 20)
   if err != nil { writeError(c, err); return }
   res, err := h.listUC.Execute(page, size)
   if err != nil { writeError(c, err); return }
   c.JSON(200, res)

   COMENTARIO: validación strict en handler (cliente que pasa explícito inválido
   → 400). Default silencioso solo cuando el query param NO viene. El use case
   mantiene clamp como defense-in-depth.

6. Método Get(c *gin.Context):
   id := c.Param("id")
   p, err := h.getUC.Execute(id)
   if err != nil { writeError(c, err); return }
   c.JSON(200, p)

7. Método Categories(c *gin.Context):
   cats, err := h.categoriesUC.Execute()
   if err != nil { writeError(c, err); return }
   c.JSON(200, gin.H{"categories": cats})

8. Helper PRIVADO `splitCSV(raw string) []string`:
   Si raw == "" → return nil.
   strings.Split + TrimSpace + skip vacíos. Devuelve slice de tokens limpios.

9. Helper PRIVADO `parsePositiveInt(c *gin.Context, name string, defaultValue int) (int, error)`:
   raw := c.Query(name)
   Si raw == "" → return defaultValue, nil.
   v, err := strconv.Atoi(raw)
   Si err != nil || v < 1 → return 0, fmt.Errorf("%w: %s=%q", domain.ErrInvalidPagination, name, raw).
   Sino → return v, nil.

ANTI-PATTERN: NO meter dedup, NO meter validación de fields acá. Eso vive en el USE CASE.

═══════════════════════════════════════════════════════════════════════════
ARCHIVO 2: internal/infrastructure/http/router.go
═══════════════════════════════════════════════════════════════════════════

PACKAGE: http
IMPORTS: log/slog, github.com/gin-gonic/gin

CONTENIDO: función PÚBLICA `NewRouter(h *ProductHandler, logger *slog.Logger) *gin.Engine`:

   r := gin.New()                       // NO gin.Default() — elegimos middlewares explícitos
   _ = r.SetTrustedProxies(nil)         // no confiar en X-Forwarded-* de clientes directos

   r.Use(
       gin.Recovery(),
       RequestIDMiddleware(),
       LoggingMiddleware(logger),
       SecurityHeadersMiddleware(),
   )

   r.GET("/health", h.Health)

   products := r.Group("/products")
   {
       products.GET("", h.List)
       products.GET("/compare", h.Compare)        // ANTES que /:id
       products.GET("/categories", h.Categories)  // ANTES que /:id
       products.GET("/:id", h.Get)
   }

   return r

REGLA CRÍTICA: las rutas /compare y /categories DEBEN registrarse ANTES que /:id.
Gin matchea por orden — si /:id va primero, "compare" cae como id.

COMENTARIOS REQUERIDOS:
- POR QUÉ gin.New() y no gin.Default() (elegimos stack de middlewares explícito,
  reemplazamos Logger por uno estructurado con slog).
- POR QUÉ SetTrustedProxies(nil) (no confiar en headers de proxies por default;
  en despliegue real se configuraría la lista del LB).
- Sobre el orden de rutas (el comentario que ya está arriba).

═══════════════════════════════════════════════════════════════════════════
VALIDACIÓN
═══════════════════════════════════════════════════════════════════════════

   go build ./internal/infrastructure/http/...

ESPERADO: sin errores.

NO hay tests todavía — vienen en el próximo prompt.

LISTAME los archivos creados/modificados.
```

---

## Prompt 6 — HTTP tests e2e (httptest)

```
PRECONDICIÓN: handler + router compilan.

ARCHIVO ÚNICO: internal/infrastructure/http/product_handler_test.go

PACKAGE: http
IMPORTS:
   encoding/json, io, log/slog, net/http, net/http/httptest, os,
   path/filepath, testing,
   github.com/gin-gonic/gin,
   github.com/<TU-MODULE>/internal/application,
   github.com/<TU-MODULE>/internal/infrastructure/persistence

ESTRATEGIA: tests e2e con stack completo (repo real con fixture en t.TempDir()
+ use cases + handler + router). NO mocks — verificamos el contrato HTTP real.

═══════════════════════════════════════════════════════════════════════════

1. Constante `fixtureJSON` con 3 productos:
   [
     {"id":"1","name":"iPhone","category":"smartphones","price":1000,"rating":4.5,"specs":{"ram":"8GB"}},
     {"id":"2","name":"Galaxy","category":"smartphones","price":900,"rating":4.4,"specs":{"ram":"12GB"}},
     {"id":"3","name":"Book","category":"books","price":30,"rating":4.8}
   ]

2. Helper `setupRouter(t *testing.T) *gin.Engine`:
   - gin.SetMode(gin.TestMode).
   - Escribir fixtureJSON en t.TempDir() + "/products.json".
   - persistence.NewJSONRepository(path).
   - Instanciar los 4 use cases.
   - h := NewProductHandler(...).
   - LOGGER SILENCIOSO: silentLogger := slog.New(slog.NewJSONHandler(io.Discard, nil)).
   - return NewRouter(h, silentLogger).

3. Helper `doRequest(t, router, method, url) (*httptest.ResponseRecorder, map[string]any)`:
   - httptest.NewRecorder().
   - http.NewRequest(method, url, nil).
   - router.ServeHTTP(w, req).
   - json.Unmarshal(w.Body.Bytes(), &body) si body no vacío.
   - return w, body.

═══════════════════════════════════════════════════════════════════════════
TESTS REQUERIDOS (14 en total)
═══════════════════════════════════════════════════════════════════════════

1. TestHealth — 200 + status="ok".
2. TestListProducts_Default — 200 + total==3.
3. TestGetProduct_Found — 200 + name=="iPhone".
4. TestGetProduct_NotFound — 404 + body["error"] no nil.
5. TestCategories — 200 + 2 categorías.
6. TestCompare_HappyPath — /compare?ids=1,2&fields=name,price → 200.
   - len(items)==2.
   - first["name"] presente, first["category"] NO presente.
7. TestCompare_MissingIDsReturns404 — /compare?ids=1,99 → 404 + missing_ids==["99"].
8. TestCompare_EmptyIDsReturns400 — /compare → 400.
9. TestCompare_InvalidFieldReturns400 — /compare?ids=1&fields=cost_price → 400.
10. TestList_PageOutOfRangeReturnsEmptyItemsWithTotal — /products?page=999&size=10 → 200, items=[], total=3.
11. TestList_NegativePageReturns400 — /products?page=-1 → 400.
12. TestList_NonNumericSizeReturns400 — /products?size=abc → 400.
13. TestCompare_DuplicateIDsAreDeduplicated — /compare?ids=1,1,2&fields=name → 200, len(items)==2.
14. TestCompare_FullProjectionWhenFieldsOmitted — /compare?ids=1 → 200, todos los campos del producto.

═══════════════════════════════════════════════════════════════════════════
VALIDACIÓN
═══════════════════════════════════════════════════════════════════════════

   go test ./internal/infrastructure/http/... -v

ESPERADO: 14 tests verde.

   go test ./... -cover

ESPERADO ACUMULADO: 47 tests verde, cobertura > 80%.

LISTAME los archivos creados.
```

---

## Prompt 7 — Composition root con graceful shutdown

```
PRECONDICIÓN: 47 tests verde.

CAPA ACTUAL: cmd/api. Composition root + lifecycle del proceso.

ARCHIVO ÚNICO: cmd/api/main.go

PACKAGE: main
IMPORTS: context, errors, log/slog, net/http, os, os/signal, path/filepath,
         strings, syscall, time,
         github.com/gin-gonic/gin,
         github.com/<TU-MODULE>/internal/application,
         httpadapter "github.com/<TU-MODULE>/internal/infrastructure/http",
         github.com/<TU-MODULE>/internal/infrastructure/persistence

NOTA SOBRE EL ALIAS `httpadapter`: usar alias para evitar colisión con net/http.

═══════════════════════════════════════════════════════════════════════════
CONTENIDO
═══════════════════════════════════════════════════════════════════════════

1. Constantes con doc-comment EXPLICATIVO sobre slowloris:
   const (
       readHeaderTimeout = 5 * time.Second
       readTimeout       = 10 * time.Second
       writeTimeout      = 15 * time.Second
       idleTimeout       = 60 * time.Second
       shutdownTimeout   = 10 * time.Second
   )

2. Función `main()`:

   a. Crear logger JSON estructurado:
      logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

   b. Configurar Gin mode (env var override + default release):
      if mode := os.Getenv("GIN_MODE"); mode != "" {
          gin.SetMode(mode)
      } else {
          gin.SetMode(gin.ReleaseMode)
      }

   c. Leer PRODUCTS_FILE de env, default "testdata/products.json".

   d. Validar el path con validateProductsPath(). Si falla:
      logger.Error("invalid PRODUCTS_FILE", slog.String("path", dataPath), slog.Any("error", err))
      os.Exit(1)

   e. Cargar repo. Si falla → logger.Error + os.Exit(1).
      Sino → logger.Info("products loaded", ...).

   f. Instanciar los 4 use cases.

   g. handler := httpadapter.NewProductHandler(...).
      router := httpadapter.NewRouter(handler, logger).

   h. Construir el http.Server EXPLÍCITAMENTE (NO usar router.Run()):
      srv := &http.Server{
          Addr:              ":" + portFromEnv(),
          Handler:           router,
          ReadHeaderTimeout: readHeaderTimeout,
          ReadTimeout:       readTimeout,
          WriteTimeout:      writeTimeout,
          IdleTimeout:       idleTimeout,
      }

   i. Lanzar server en goroutine, mandando errores no-ErrServerClosed a un channel:
      serverErr := make(chan error, 1)
      go func() {
          logger.Info("server listening", slog.String("addr", srv.Addr))
          if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
              serverErr <- err
          }
      }()

   j. Setup signal handler:
      stop := make(chan os.Signal, 1)
      signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

   k. Select entre serverErr y stop:
      select {
      case err := <-serverErr:
          logger.Error("server failed", slog.Any("error", err))
          os.Exit(1)
      case sig := <-stop:
          logger.Info("shutdown signal received", slog.String("signal", sig.String()))
      }

   l. Graceful shutdown con timeout:
      ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
      defer cancel()
      if err := srv.Shutdown(ctx); err != nil {
          logger.Error("graceful shutdown failed, forcing close", slog.Any("error", err))
          _ = srv.Close()
          os.Exit(1)
      }
      logger.Info("server stopped cleanly")

3. Helper `portFromEnv() string`: PORT env o "8080".

4. Helper `validateProductsPath(path string) error`:
   - path == "" → errors.New("path is empty")
   - !strings.HasSuffix(strings.ToLower(path), ".json") → errors.New("path must end in .json")
   - strings.HasPrefix(filepath.Clean(path), "..") → errors.New("path contains traversal segments")
   - return nil

═══════════════════════════════════════════════════════════════════════════
VALIDACIÓN
═══════════════════════════════════════════════════════════════════════════

1. Build:
   go build ./cmd/api

2. Tests completos:
   go test ./... -cover
   ESPERADO: 47 tests verde, cobertura > 80%.

3. Vet:
   go vet ./...
   ESPERADO: sin warnings.

4. Smoke de graceful shutdown:
   go build -o /tmp/api ./cmd/api && /tmp/api &
   PID=$!
   sleep 1
   kill -TERM $PID
   wait $PID
   echo "exit code: $?"

   ESPERADO en logs JSON: "shutdown signal received" + "server stopped cleanly".
   exit code: 0.

LISTAME los archivos modificados.
```

---

## Prompt 8 — Tooling: Makefile + .gitignore

```
PRECONDICIÓN: el proyecto compila y tests verdes.

═══════════════════════════════════════════════════════════════════════════
ARCHIVO 1: .gitignore
═══════════════════════════════════════════════════════════════════════════

# Binaries y artefactos de build
/api
/bin/
*.exe
*.test
*.out

# Coverage
coverage.out
coverage.html

# Editor / OS
.DS_Store
.idea/
.vscode/
*.swp

# Env / secrets locales
.env
.env.local

═══════════════════════════════════════════════════════════════════════════
ARCHIVO 2: Makefile
═══════════════════════════════════════════════════════════════════════════

.PHONY: run build test test-v cover cover-html vet fmt lint lint-install tidy vulncheck check

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

test:
	go test ./...

test-v:
	go test -v ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

cover-html:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "open coverage.html"

vet:
	go vet ./...

fmt:
	gofmt -w .

lint:
	golangci-lint run ./...

lint-install:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

tidy:
	go mod tidy

vulncheck:
	@which govulncheck > /dev/null || go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

# check: lo mínimo que debería pasar antes de un commit/PR.
check: fmt vet test

═══════════════════════════════════════════════════════════════════════════
VALIDACIÓN
═══════════════════════════════════════════════════════════════════════════

   make check
   make build
   ls bin/api

ESPERADO: tests verdes, fmt+vet limpios, binario en bin/api.
```

---

## Prompt 9 — Smoke test end-to-end

```
PRECONDICIÓN: todo compilado y verde.

TAREA: arrancar el server en background y correr todos los curls que validan
los comportamientos no triviales. Pegá esto en una terminal:

```bash
go build -o /tmp/api ./cmd/api && /tmp/api &
SERVER_PID=$!
sleep 2

echo "=== HEALTH ==="
curl -sw "\nHTTP %{http_code}\n" http://localhost:8080/health

echo "=== LIST con paginación ==="
curl -sw "\nHTTP %{http_code}\n" "http://localhost:8080/products?page=1&size=2"

echo "=== GET found ==="
curl -sw "\nHTTP %{http_code}\n" http://localhost:8080/products/1

echo "=== GET not found (404) ==="
curl -sw "\nHTTP %{http_code}\n" http://localhost:8080/products/9999

echo "=== CATEGORIES ==="
curl -sw "\nHTTP %{http_code}\n" http://localhost:8080/products/categories

echo "=== COMPARE happy path ==="
curl -sw "\nHTTP %{http_code}\n" "http://localhost:8080/products/compare?ids=1,2,3&fields=name,price,rating"

echo "=== COMPARE empty ids (400) ==="
curl -sw "\nHTTP %{http_code}\n" "http://localhost:8080/products/compare"

echo "=== COMPARE missing id (404 + missing_ids) ==="
curl -sw "\nHTTP %{http_code}\n" "http://localhost:8080/products/compare?ids=1,99"

echo "=== COMPARE invalid field (400) ==="
curl -sw "\nHTTP %{http_code}\n" "http://localhost:8080/products/compare?ids=1&fields=cost_price"

echo "=== COMPARE too many ids (400) ==="
LONG_IDS=$(seq 1 60 | tr '\n' ',' | sed 's/,$//')
curl -sw "\nHTTP %{http_code}\n" "http://localhost:8080/products/compare?ids=$LONG_IDS"

echo "=== COMPARE dedup (?ids=1,1,2 → 2 items) ==="
curl -s "http://localhost:8080/products/compare?ids=1,1,2&fields=name"

echo "=== Headers de seguridad + Request ID ==="
curl -si http://localhost:8080/health | grep -iE "x-content-type-options|referrer-policy|x-request-id"

echo "=== Request ID propagado del cliente ==="
curl -si -H "X-Request-Id: trace-abc-123" http://localhost:8080/health | grep -i "x-request-id"

echo "=== List page=-1 → 400 ==="
curl -sw "\nHTTP %{http_code}\n" "http://localhost:8080/products?page=-1"

echo "=== List size=abc → 400 ==="
curl -sw "\nHTTP %{http_code}\n" "http://localhost:8080/products?size=abc"

echo "=== List page=999 fuera de rango → 200 con items=[] ==="
curl -sw "\nHTTP %{http_code}\n" "http://localhost:8080/products?page=999&size=10"

echo "=== Graceful shutdown (SIGTERM) ==="
kill -TERM $SERVER_PID
wait $SERVER_PID
echo "exit code: $?"
```

CHECKLIST DE RESULTADOS ESPERADOS:
- /health → 200 {"status":"ok"}
- /products → 200 con items + total + page + size
- /products/1 → 200 con producto completo
- /products/9999 → 404 {"error":"product not found"}
- /products/categories → 200 {"categories":[...]} ordenado alfabéticamente
- /products/compare con projection → 200 con SOLO los fields pedidos
- /products/compare sin ids → 400
- /products/compare?ids=1,99 → 404 con missing_ids:["99"]
- /products/compare?ids=1&fields=cost_price → 400
- /products/compare con 60 ids → 400 ErrTooManyIDs
- /products/compare?ids=1,1,2 → 200 con SOLO 2 items (dedup)
- Headers `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer`, `X-Request-Id: <hex>` en TODA respuesta
- X-Request-Id propagado: si lo mandás, el response tiene el mismo
- /products?page=-1 → 400
- /products?size=abc → 400
- /products?page=999 → 200 con items=[]
- SIGTERM → log JSON "shutdown signal received" + "server stopped cleanly", exit 0
```

---

## Prompt 10 — Documentación (problem.md + README.md)

```
PRECONDICIÓN: smoke test pasó.

═══════════════════════════════════════════════════════════════════════════
ARCHIVO 1: problem.md
═══════════════════════════════════════════════════════════════════════════

Pegá EL contenido del problem.md del repo local. Está en español, no lo regeneres.

Si no lo tenés a mano, decímelo y te lo doy.

═══════════════════════════════════════════════════════════════════════════
ARCHIVO 2: README.md
═══════════════════════════════════════════════════════════════════════════

README profesional en español con ESTAS SECCIONES en este orden:

1. **Título + tagline** (1 línea): "Backend API — Comparación de Productos (Mercado Libre Challenge)" + "Implementado con Clean Architecture liviana y TDD estricto."

2. **Stack** (tabla): Lenguaje | Framework | Persistencia | Testing | Logs | Lifecycle. Cada uno con "por qué".

3. **Arquitectura** (diagrama ASCII de las 3 capas + composition root + flujo de dependencias). Regla de oro destacada: "el dominio no sabe que existe Gin, JSON ni HTTP".

4. **Estructura de directorios** (árbol).

5. **Endpoints** (tabla método/path/descripción).

6. **Errores** (tabla status/cuándo).

7. **Schemas de respuesta** (JSON example por endpoint: /health, /products, /products/:id, /products/categories, /products/compare, error response).

8. **Detalles de comportamiento**:
   - dedup en use case (no handler)
   - cap MaxCompareIDs=50
   - page fuera de rango → 200 con items=[]
   - page=-1 → 400 (strict en handler)
   - X-Request-Id (honra entrante o genera, devuelve en response, va en logs)

9. **Setup**: requisitos + comandos básicos (go test, go build, go run).

10. **Comandos via Makefile** (run/build/test/cover/cover-html/vet/fmt/vulncheck/check).

11. **Cobertura**: comandos manuales + govulncheck.

12. **Smoke test** (con curls + responses esperadas reales).

13. **Decisiones arquitectónicas** (tabla decisión/trade-off).

14. **Robustez de runtime (incluida)** — tabla con: graceful shutdown, HTTP timeouts, logs estructurados, request correlation, security headers, trusted proxies, gin release mode, cap anti-DoS, validación de path, fail-fast.

15. **Cobertura de tests**: 47 tests en 5 paquetes (domain 6, application 16, persistence 11, http 14).

16. **Lo que NO está incluido (out of scope intencional)** — tabla con pieza/por qué afuera/cómo se agregaría:
    Auth/JWT, Rate limiting, CORS, Cache, Métricas Prometheus, OpenAPI/Swagger,
    Versionado /v1, Tracing distribuido (OTel), DB real, CI/CD pipelines.

17. **Cómo extender** (tabla "quiero agregar X / toco Y").

18. Frase de cierre: "Saber cuándo NO hacer algo es tan senior como saber hacerlo."

═══════════════════════════════════════════════════════════════════════════
VALIDACIÓN
═══════════════════════════════════════════════════════════════════════════

Releé el README. Confirmá que TODOS los comportamientos descriptos en
"Detalles de comportamiento" coinciden con lo que devuelven los curls del Prompt 9.
Si hay algún desajuste, decímelo.
```

---

## Frases de freno (anti-improvisación)

Si el agente desvía, frenálo con la frase exacta:

| Síntoma | Frase de freno |
|---|---|
| Cambia el package layout o nombres de archivos | "STOP. No cambies la estructura de packages ni los nombres de archivos. Volvé al layout original que te di en el Prompt 0." |
| Quiere agregar features no pedidas (auth, cache, métricas) | "STOP. Eso no estaba en el prompt. Borralo y dejá solo lo que te pedí." |
| Mete frameworks de DI / mocking libs | "STOP. La regla 2 del System Prompt prohíbe Wire/Fx/gomock. DI manual y mocks manuales. Reescribilo." |
| Comentarios que explican QUÉ hace el código | "STOP. La regla 4 del System Prompt: comentarios explican WHY, no WHAT. Reescribí los comentarios siguiendo esa regla." |
| Cambia los nombres de los endpoints o de los métodos | "STOP. Los endpoints son /health, /products, /products/compare, /products/categories, /products/:id. No cambies nada." |
| Pone validaciones en domain que pertenecen a application | "STOP. domain solo tiene lógica de entity. Las validaciones de fields whitelist se hacen en application." |
| Pone dedup o cap en handler en vez de use case | "STOP. Dedup y cap viven en el USE CASE (compare_products.go), no en el handler. Movélos." |
| Devuelve 200 con productos parciales en /compare | "STOP — eso es bug semántico. Si falta UN solo id en /compare es 404 con missing_ids en el body. NO 200 parcial." |
| Usa router.Run() en vez de http.Server explícito | "STOP. main.go DEBE construir &http.Server{...} con timeouts y usar Shutdown(). router.Run() no permite eso." |
| Usa gin.Default() en vez de gin.New() + middlewares custom | "STOP. NewRouter usa gin.New() y monta nuestros middlewares custom (Recovery, RequestID, Logging slog, SecurityHeaders). No gin.Default()." |
| Agrega un /ping o cualquier endpoint extra | "STOP. Solo los 5 endpoints del Prompt 5: /health, /products, /products/compare, /products/categories, /products/:id." |
| Olvida los nil-checks en el mock | "STOP. mockRepository TIENE que panic con mensaje descriptivo si el func correspondiente es nil. Sin eso, los tests fallan crípticamente." |

---

## Checklist final antes de submit en HackerRank

- [ ] `make check` (o `go test ./...`) → 47 tests verde
- [ ] `go vet ./...` sin warnings
- [ ] `go build ./cmd/api` compila
- [ ] `make vulncheck` (govulncheck) → sin CVEs activos
- [ ] Smoke test del Prompt 9 pasa todos los casos esperados
- [ ] Headers `X-Content-Type-Options`, `Referrer-Policy`, `X-Request-Id` aparecen en todas las respuestas
- [ ] El binario hace graceful shutdown ante SIGTERM (logs JSON "stopped cleanly", exit 0)
- [ ] README.md presente con setup + endpoints + decisiones + robustez + "lo que no está"
- [ ] problem.md en español
- [ ] Comentarios en código explican WHY, no WHAT
- [ ] Estructura de directorios coincide con el repo local
- [ ] No quedó ningún `/api` binario o archivos de coverage en el árbol (.gitignore los cubre)

---

## Tip final

Después de cada prompt, ejecutá esta línea para ver qué cambió y validar visualmente:

```
git status --short && git diff --stat
```

Si algún archivo aparece que NO debería (ej: `cmd/api/cli.go`, `internal/utils/`), el agente improvisó — frenálo con la frase de freno correspondiente.

---

## Prompt 11 — Bloque 1 de robustez técnica (safety net operativo/config)

```
TAREA: agregar tests de robustez técnica SIN cambiar la semántica pública del API.

ALCANCE EXACTO:

1) internal/infrastructure/http/middleware_test.go
   - TestRequestIDMiddleware_PreservesIncomingHeader
     Verifica que si entra X-Request-Id, la respuesta devuelve exactamente ese valor.
   - TestRequestIDMiddleware_GeneratesHeaderWhenMissing
     Verifica que si no entra X-Request-Id, la respuesta lo incluye no vacío.
   - TestSecurityHeadersMiddleware_AddsExpectedHeaders
     Verifica headers: X-Content-Type-Options=nosniff y Referrer-Policy=no-referrer.

2) internal/infrastructure/http/error_handler_test.go
   - Tabla de mapeo de errores conocidos:
     ErrProductNotFound -> 404
     MissingIDsError -> 404
     ErrEmptyIDs -> 400
     ErrInvalidField -> 400
     ErrInvalidPagination -> 400
     ErrTooManyIDs -> 400
   - Caso unknown error -> 500.

3) cmd/api/main_test.go
   - TestPortFromEnv:
     - usa PORT cuando está seteado
     - default 8080 cuando falta
   - TestValidateProductsPath (table-driven):
     - válidos: testdata/products.json, testdata/PRODUCTS.JSON
     - inválidos: path vacío, extensión no .json, path traversal ../secrets.json

4) Router precedence (ruta estática vs dinámica)
   - Agregar test puntual en internal/infrastructure/http/product_handler_test.go
     para asegurar que GET /products/categories matchea Categories y NO /products/:id.

REGLAS:
- No tocar código productivo salvo que sea estrictamente necesario para testear.
- No cambiar contratos HTTP ni payloads públicos.
- Mantener estilo table-driven y nombres de tests descriptivos.

VALIDACIÓN (solo tests, no build):
  go test ./cmd/api ./internal/infrastructure/http -v
  go test ./... -v

ESPERADO:
- Suite verde
- Se agregan 8 tests netos (el total depende del estado actual del repo)

REPORTÁ:
- Archivos creados/modificados
- Si hubo que tocar comportamiento (y por qué)
```

---

## Prompt 12 — Bloque 2 de observabilidad (trazabilidad de errores 500)

```

---

## Prompt 13 — Bloque 3 de hardening de inputs/config

```
TAREA: endurecer bordes de entrada HTTP + configuración con cambios mínimos y sin romper contrato público.

OBJETIVO:
- Blindar parsing de query params frente a entradas sucias/extremas
- Revisar PRODUCTS_FILE y cambiar comportamiento SOLO si hay bug real

ALCANCE EXACTO:

1) Consolidar contrato actual en tests HTTP (internal/infrastructure/http/product_handler_test.go):
   - CSV vacío con solo tokens vacíos: /products/compare?ids=,%20,%20,, -> 400
   - Espacios raros en CSV: /products/compare?ids=%201%20,%20%202%20,,&fields=%20name%20,%20price%20 -> 200, len(items)=2, fields=["name","price"]
   - Params repetidos: /products/compare?ids=1&ids=2&fields=name -> mantener semántica actual (Query("ids") toma primer valor), len(items)=1
   - Valores extremos/no parseables en paginación:
     * /products?page=999999999999999999999999 -> 400
     * /products?size=1e3 -> 400

2) PRODUCTS_FILE (cmd/api/main.go + cmd/api/main_test.go):
   - Detectar si hay bug real. Bug justificado: env con espacios alrededor (común en CI/.env) hoy puede fallar por falso negativo.
   - Fix mínimo: aplicar strings.TrimSpace al leer PRODUCTS_FILE y al inicio de validateProductsPath.
   - Mantener reglas existentes: vacío inválido, extensión .json requerida, traversal inválido.
   - Agregar tests de contrato:
     * válido con espacios alrededor: "  testdata/products.json  " -> válido
     * nested traversal: testdata/../../secrets.json -> inválido

REGLAS:
- No cambiar payloads/status públicos fuera de casos bugfix justificados.
- No agregar dependencias nuevas.
- Mantener cambios chicos, auditables y orientados a bordes.

VALIDACIÓN (sin build):
  go test ./internal/infrastructure/http ./cmd/api -v
  go test ./... -v

REPORTÁ:
- archivos modificados
- bug real detectado (si aplica) y por qué el fix no rompe semántica pública
- riesgos/trade-offs
```
TAREA: mejorar observabilidad interna de errores 500 SIN cambiar semántica pública del API.

ALCANCE EXACTO:

1) Revisar el manejo actual de unknown/internal errors en la capa HTTP
   - Punto de entrada: internal/infrastructure/http/error_handler.go (writeError)
   - Logging actual: internal/infrastructure/http/middleware.go (LoggingMiddleware)

2) Cambio mínimo requerido
   Cuando writeError responda el default 500 genérico:
   - registrar la causa real del error para uso interno (logs/troubleshooting)
   - si existe request_id en contexto, que quede en el log esperado junto con esa causa
   - mantener payload público 500 genérico: {"error":"internal server error"}
   - NO exponer err.Error() al cliente

3) Tests (agregar/ajustar)
   En internal/infrastructure/http/error_handler_test.go:
   - fortalecer TestWriteError_UnknownErrorReturns500 para validar que el error
     unknown queda registrado internamente en el contexto de Gin (no solo status)
   - agregar test de integración liviano con RequestIDMiddleware + LoggingMiddleware:
     * request con X-Request-Id explícito
     * handler fuerza unknown error (ej: errors.New("db connection timeout"))
     * assert: HTTP 500 + payload genérico
     * assert: el log estructurado incluye request_id e internal_error con la causa real

REGLAS:
- No cambiar contratos públicos HTTP ni formato de error público.
- No introducir librerías nuevas.
- Mantener enfoque mínimo (sin over-engineering).

VALIDACIÓN (solo tests):
  go test ./internal/infrastructure/http -v
  go test ./... -v

ESPERADO:
- Suite verde
- 500 sigue siendo genérico para cliente
- Causa real trazable en logs internos con request_id cuando aplica

REPORTÁ:
- Archivos modificados
- Qué se agregó para observabilidad interna
- Riesgos o trade-offs
```
