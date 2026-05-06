# Guía de narrativa de entrevista — MELI HackerRank Challenge

Material para defender el challenge frente a un evaluador técnico exigente.
Este archivo se cruza con el código real: cada decisión enlaza al archivo y línea concreta.

> **Origen del proyecto**: prueba técnica de Mercado Libre (HackerRank) — backend API REST para comparación de productos con proyección selectiva de campos.

---

## 0) Cheat-sheet (memorizar)

| Dato | Valor |
|---|---|
| Stack | Go 1.21+, Gin, `log/slog` (stdlib), `httptest` (stdlib) |
| Arquitectura | Clean Architecture liviana (3 capas) + DI manual |
| Tests | **78 pasando** en 5 paquetes |
| Coverage | domain 95.5% · application 97.7% · persistence 100% · http 94.5% · cmd 21.6% |
| Endpoints | 5 (`/health`, `/products`, `/products/:id`, `/products/compare`, `/products/categories`) |
| Endpoint estrella | `GET /products/compare?ids=1,2,3&fields=name,price,rating` |
| Caps defensivos | `MaxCompareIDs=50`, `size<=100` en paginación |
| Persistencia | JSON in-memory cargado al boot (fail-fast si falla) |
| Observabilidad | `slog` JSON + `X-Request-Id` end-to-end |
| Lifecycle | Graceful shutdown SIGINT/SIGTERM con drain 10s |

---

## 1) Speech de 90 segundos

> Diseñé esta solución como una **API REST en Go con Gin** y **Clean Architecture liviana** — tres capas con regla de dependencias estricta: el dominio no sabe que existen Gin, JSON ni HTTP. Eso me permite cambiar el framework HTTP o la persistencia sin tocar reglas de negocio.
>
> El caso principal es comparar productos con **proyección selectiva** vía `GET /products/compare?ids=1,2&fields=name,price`. Ahí prioricé tres cosas: validaciones estrictas, contrato claro y comportamiento predecible. La decisión más defendible: si falta cualquier ID devuelvo **404 con `missing_ids` explícito**, en lugar de 200 parcial — comparar con datos incompletos induce decisiones erróneas del cliente.
>
> En no funcionales reforcé tres ejes. **Testing**: 78 tests pasando, 95-100% de coverage en las capas de negocio, mocks manuales del repo para aislar use cases. **Observabilidad**: `log/slog` JSON con `X-Request-Id` propagado por header — si necesito debuggear un request, lo correlaciono en una sola query. **Hardening**: timeouts HTTP explícitos contra slowloris, graceful shutdown con drain de 10s, caps anti-DoS (`MaxCompareIDs=50`, `size<=100`), whitelist de campos, validación de path del catálogo, y headers de seguridad básicos.
>
> También cuidé portabilidad para HackerRank: persistencia en JSON sin drivers, **una sola dependencia externa (Gin)**, DI manual sin frameworks, y guía explícita de migración por capas. Si la plataforma no permite Gin o subdirectorios, el diseño permite degradar a `net/http` o single-file con impacto local en infraestructura — el dominio no se entera.

---

## 2) Versión corta (30 segundos)

> Backend REST en Go + Gin con Clean Architecture liviana para desacoplar negocio de infraestructura. El endpoint clave, `/products/compare`, hace proyección selectiva de campos y falla con 404 + `missing_ids` si falta cualquier producto — evito comparaciones silenciosamente parciales. 78 tests, coverage 95-100% en negocio, observabilidad con `slog` + `X-Request-Id`, graceful shutdown y caps anti-DoS. Solución portable a HackerRank: JSON in-memory, una sola dependencia, plan B documentado para `net/http` o single-file.

---

## 3) Decisiones defendibles (con trade-offs)

### 1. Clean Architecture liviana en 3 capas
- **Decisión**: `internal/domain`, `internal/application`, `internal/infrastructure/{http,persistence}`, `cmd/api` como composition root.
- **Por qué**: regla de dependencias estricta — el dominio no importa nada externo (solo stdlib). Permite testear cada capa aislada y cambiar adapters sin tocar negocio.
- **Trade-off**: más archivos y wiring manual inicial vs. layered simple. En un challenge chico se nota el costo extra; lo acepto porque demuestra criterio arquitectónico.
- **Código**: ver `internal/domain/repository.go` (port) y `cmd/api/main.go:80-87` (wiring).

### 2. Port `ProductRepository` definido en `domain` (no en application)
- **Decisión**: la interfaz vive en `domain/repository.go`. Application la consume, infrastructure la implementa.
- **Por qué**: **Inversión de Dependencias real** (el dominio define lo que necesita, no lo que le dan). Cambiar JSON → Postgres es agregar un nuevo adapter, no tocar use cases ni dominio.
- **Trade-off**: una interfaz "extra" para un challenge chico — vale la pena por la coherencia con el principio.

### 3. Persistencia JSON in-memory cargada al boot
- **Decisión**: `JSONRepository` lee el archivo una vez en `NewJSONRepository`, mantiene el catálogo en memoria + index `byID` para lookup O(1).
- **Por qué**: portabilidad (sin drivers), velocidad (cero I/O por request), demo simple.
- **Trade-off**: cambios al archivo requieren restart; no hay durabilidad runtime. Para producción cambiaríamos el adapter (puerto ya está).
- **Detalle senior**: uso `sync.RWMutex` aunque hoy nunca escribo. ¿Por qué? Si mañana agrego `POST /products`, la clase ya está cubierta. RLock en lecturas concurrentes es prácticamente free.

### 4. Fail-fast en boot del repo
- **Decisión**: si el JSON tiene IDs vacíos o duplicados, `NewJSONRepository` retorna error y el binario no arranca.
- **Por qué**: un servicio sin catálogo válido no debería arrancar pretendiendo estar sano. ID duplicado rompe lookups silenciosamente — mejor explotar al boot que devolver datos inconsistentes en runtime.
- **Código**: `internal/infrastructure/persistence/json_repository.go:62-73`.

### 5. `Specs` como `map[string]any`
- **Decisión**: campos comunes tipados (id, name, price...) + `Specs map[string]any` para atributos por categoría.
- **Por qué**: catálogo heterogéneo (smartphones, libros, ropa, electrodomésticos). Agregar una categoría nueva NO requiere cambiar el schema — solo el JSON.
- **Trade-off**: menor type-safety en specs (no detecto typos en compile time). Lo acepto porque la flexibilidad gana en e-commerce real con miles de SKUs heterogéneos. La whitelist de campos top-level sigue protegiendo el contrato del API.
- **Alternativa descartada**: structs por categoría con embedding. Rígido — cada categoría nueva requiere código nuevo. No escala.

### 6. `/products/compare` con 404 todo-o-nada + `missing_ids`
- **Decisión**: si falta cualquier ID solicitado, devuelvo `404` con body `{"error":"products not found: 99","missing_ids":["99"]}`.
- **Por qué**: comparar implica que TODOS los items son relevantes. Devolver 200 con N-1 productos engaña al cliente — nunca sabría qué falta. La UX correcta es fallar explícito y devolver QUÉ falta para que el cliente corrija.
- **Trade-off**: el cliente debe re-armar la query. Aceptado: prefiero fricción explícita a decisión silenciosa.

### 7. Whitelist positiva de `fields` (no blacklist)
- **Decisión**: `allowedFields` enumera explícitamente los campos seleccionables. Cualquier campo no listado se rechaza con `400`.
- **Por qué**: **seguridad por defecto**. Si mañana agrego `cost_price` o `internal_sku` al modelo, no se filtra al cliente hasta habilitarlo explícitamente. Blacklist olvida campos nuevos; whitelist no.
- **Trade-off**: mantener la lista al agregar campos públicos. Costo bajísimo y obvio en code review.

### 8. Dedup de IDs en el use case (no en el handler)
- **Decisión**: `?ids=1,1,2` → use case ve `[1, 2]`. Lo hace `dedupPreservingOrder` antes de tocar el repo.
- **Por qué**: la dedup es **regla de negocio**, no detalle de transporte. Si mañana expongo el use case por gRPC o CLI, recibe la misma semántica sin reimplementarla.
- **Código**: `internal/application/compare_products.go:60`.

### 9. Cap de IDs (`MaxCompareIDs=50`) y de page size (`<=100`)
- **Decisión**: `/products/compare` rechaza más de 50 IDs con 400. `/products` capea silenciosamente size a 100.
- **Por qué**: defensa anti-DoS por amplificación. Un `?ids=1,1,...(100k)` gasta CPU+memoria en parseo y dedup aunque el catálogo sea chico. Y comparar 1000 productos no tiene sentido de UX.
- **Trade-off**: cliente con caso legítimo extremo necesita paginar/batchar. No conozco ese caso real para "comparar".

### 10. Timeouts HTTP explícitos (defensa anti-slowloris)
- **Decisión**: `ReadHeaderTimeout=5s`, `ReadTimeout=10s`, `WriteTimeout=15s`, `IdleTimeout=60s`.
- **Por qué**: `net/http.Server` con valores cero permite que un cliente lento mantenga conexiones abiertas indefinidamente — slowloris attack. Estos valores son conservadores pero seguros para una API que devuelve JSON pequeño.
- **Código**: `cmd/api/main.go:36-42`.

### 11. Graceful shutdown con SIGINT/SIGTERM
- **Decisión**: `signal.Notify(SIGINT, SIGTERM)` + `srv.Shutdown(ctx)` con timeout 10s. Si no drena en 10s, `srv.Close()` fuerza.
- **Por qué**: en kubernetes/systemd, SIGTERM es la señal canónica. Cortar abruptamente requests en vuelo rompe clientes. 10s es suficiente para drenar JSON pequeño.
- **Código**: `cmd/api/main.go:107-128`.

### 12. Observabilidad: `slog` JSON + `X-Request-Id` end-to-end
- **Decisión**: middleware genera/respeta `X-Request-Id`, lo loggea en cada request junto con `method`, `path`, `status`, `duration`, `client_ip`.
- **Por qué**: trazabilidad. Si un cliente reporta un bug y me da el request_id, lo correlaciono en una sola query de logs. Si tengo edge proxy, honro su ID — correlación cross-service gratis.
- **Trade-off**: mayor verbosidad de logs vs. silencio. En API server siempre prefiero logs estructurados — son baratos en JSON y caros de NO tener cuando algo se rompe.

### 13. Mapeo error de dominio → HTTP centralizado
- **Decisión**: un solo `writeError` en `infrastructure/http/error_handler.go` mapea errores de dominio a status codes con `errors.Is` y `errors.As`.
- **Por qué**: agregar un nuevo error tipado requiere cambiar UN archivo, no N handlers. `MissingIDsError` implementa `Unwrap` → `errors.Is(err, ErrProductNotFound)` funciona Y puedo extraer la lista de faltantes con `errors.As`. Lo mejor de los dos mundos.
- **Trade-off**: si un día necesito mapeos por endpoint, hay que customizar. Hoy todos los endpoints comparten el mismo mapeo.

### 14. DI manual (sin Wire/Fx)
- **Decisión**: 10 líneas en `cmd/api/main.go` instancian repo + use cases + handler + router.
- **Por qué**: para 4 use cases + 1 repo, DI manual es más legible y auditable que un container con generación de código. Menos magia, menor costo cognitivo en entrevista.
- **Trade-off**: si crece a 20+ dependencias, refactorizar a Wire vale la pena. No es el caso.

### 15. Validación defensiva de `PRODUCTS_FILE`
- **Decisión**: `validateProductsPath` rechaza paths sin `.json` o con `..` (path traversal).
- **Por qué**: defense-in-depth. Si alguien tiene control de env vars ya tiene RCE; esto es contra typos y configs malas en deploys.
- **Código**: `cmd/api/main.go:139-156`.

---

## 4) Preguntas probables del evaluador + respuestas modelo

### 1. ¿Por qué Go para este challenge?
**Respuesta**: Tipado fuerte, testing y `httptest` en stdlib, latencia baja, binario portable sin runtime. Para un challenge en HackerRank, reduce incertidumbre del entorno y acelera feedback. Bonus: `log/slog` y `slices` en stdlib (1.21+) bajan dependencias a casi cero. Y MELI tiene Go en su stack — alineado con la realidad.

### 2. ¿Por qué Gin y no `net/http` puro?
**Respuesta**: Gin me da routing con path params, grupos de rutas y middleware con menos boilerplate. Para 5 endpoints podría haber usado `http.ServeMux` (Go 1.22+ ya soporta path params), pero Gin tiene ergonomía probada y la abstracción de middleware es estándar. **El diseño desacopla handlers de negocio**: migrar a `net/http` es trabajo local en `infrastructure/http/`, ~30 min, sin tocar dominio ni use cases.

### 3. ¿No es overengineering Clean Architecture para 5 endpoints?
**Respuesta**: Es una versión LIVIANA: 3 capas, DI manual, pocas interfaces. El costo extra son ~5 archivos más que un layered simple. A cambio gano: tests aislados por capa, posibilidad de cambiar adapters sin tocar negocio, y demuestro criterio de separación. En entrevista vale más una solución coherente y defendible que velocidad sin diseño. Si fuera un script de 100 líneas no aplicaría — es proporción.

### 4. ¿Por qué 404 en compare cuando falta un producto, en lugar de 200 parcial?
**Respuesta**: Porque la **comparación** depende del conjunto completo. Un 200 con N-1 productos puede romper la decisión del usuario sin que se entere — comparar X con Y vs. comparar X solo es semánticamente distinto. Devuelvo 404 con `missing_ids` para que el cliente actúe explícitamente: o corrige el ID, o sabe que ya no existe. Trade-off aceptado: fricción explícita sobre silencio engañoso.

### 5. ¿Por qué dedup en el use case y no en el handler?
**Respuesta**: La dedup es **regla de negocio**, no detalle de transporte. Si expongo el mismo use case por gRPC o CLI, debe recibir la misma semántica. Poniendo la dedup en el handler de Gin, cualquier otro adapter lo reimplementa o se desincroniza. La regla "comparar X con X dos veces es lo mismo que una vez" pertenece al dominio del problema.

### 6. ¿Qué riesgos tiene `map[string]any` en `Specs`?
**Respuesta**: Pierdo type-safety en compile time — un typo en una key no se detecta hasta runtime. Lo acepto por flexibilidad: el catálogo es heterogéneo (smartphones, libros, ropa, electrodomésticos) y agregar una categoría sin tocar schema vale la pena. La protección está en otro lado: la **whitelist de fields top-level** controla qué se expone; `Specs` se proyecta como bloque entero o se omite, no se filtra por sub-key. Si fuera un dominio rígido (ej. transacciones financieras), iría con structs por tipo + interfaz polimórfica.

### 7. ¿Cómo garantizás que el dominio no depende de HTTP?
**Respuesta**: Los use cases trabajan con tipos de dominio (`domain.Product`, `[]string`) y devuelven errores tipados — nunca `gin.Context`. Los tests de `application/` usan un mock manual del repo y no levantan servidor. Si quito el handler HTTP, los use cases siguen testeables. Si quito Gin, el dominio no se entera.

### 8. ¿Cómo estructuraste el testing?
**Respuesta**: Tests por capa, **78 tests pasando**:
- `domain/`: 6 tests — entity (`SelectFields`), errors (`MissingIDsError.Unwrap`).
- `application/`: 16 tests — use cases con mocks, incluye dedup, cap de IDs, repo errors propagados.
- `persistence/`: 12 tests — adapter JSON con archivos de prueba (válido, inválido, duplicados, IDs vacíos).
- `http/`: 25 tests — e2e vía `httptest`, paginación strict, dedup, page out-of-range, middlewares, error_handler.
- `cmd/`: 2 tests — `portFromEnv` y `validateProductsPath`.

Coverage real: domain 95.5%, application 97.7%, persistence 100%, http 94.5%, cmd 21.6% (intencional — `main()` no se testea, solo helpers).

### 9. ¿Qué hiciste en observabilidad concretamente?
**Respuesta**:
- `log/slog` JSON estructurado (stdlib desde 1.21, sin dependencias).
- Middleware `RequestIDMiddleware`: genera `X-Request-Id` si no viene, lo respeta si sí; lo devuelve en response header y lo incluye en cada log.
- Middleware `LoggingMiddleware`: por cada request loguea `method`, `path`, `status`, `duration`, `client_ip`, `request_id`.
- En errores 500, el error interno va a logs (con `request_id`) pero el cliente recibe payload genérico — no filtramos detalles de implementación.

Si tuviera que crecer: métricas Prometheus (`gin-prometheus`), traces OpenTelemetry. Para servicio único el request_id es suficiente.

### 10. ¿Qué medidas de hardening incluíste?
**Respuesta**: Por capas:
- **Validación de input**: query params estrictos (page negativo → 400, no clamp silencioso), whitelist de fields, dedup, caps.
- **Caps anti-DoS**: `MaxCompareIDs=50`, `size<=100`, `ReadHeaderTimeout=5s`, `ReadTimeout=10s`, `WriteTimeout=15s`, `IdleTimeout=60s` (anti-slowloris).
- **Errores 500 genéricos**: nunca filtran `err.Error()` al cliente, solo a logs.
- **Headers**: `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer`.
- **Trusted proxies**: `SetTrustedProxies(nil)` — no confío en headers de proxies por default.
- **Gin Release mode**: por default (no debug warnings en stdout).
- **Validación de PRODUCTS_FILE**: rechaza no-`.json` y `..` (defense-in-depth).
- **Fail-fast en boot**: IDs duplicados o vacíos abortan el arranque.
- **Graceful shutdown**: drain de 10s ante SIGTERM.

### 11. ¿Cómo manejás concurrencia en el repo?
**Respuesta**: `JSONRepository` usa `sync.RWMutex`. Hoy todas las operaciones son lecturas, así que solo `RLock`. Lo agregué aunque no escriba porque:
1. Si mañana agrego `POST /products`, ya está cubierto sin refactor.
2. RLock en lecturas concurrentes es prácticamente free.
3. Cero coste cognitivo, ganamos seguridad por defecto.

**Caveat documentado**: `Product.Specs` es un map compartido por referencia con el catálogo interno. Si un consumidor mutara el map fuera del lock, corromperíamos el dataset. Hoy nadie muta — está documentado como contrato (Specs es read-only para el consumidor). Si en el futuro se necesita mutar, deep-copy explícito en ese punto. No pago el costo de copy en cada `FindByID` cuando el 99% de callers no muta.

### 12. ¿Qué harías si el volumen crece 100x?
**Respuesta**: La arquitectura ya separa el punto de cambio:
1. Reemplazar `JSONRepository` por `PostgresRepository` que implemente el mismo port → use cases no se enteran.
2. Agregar índice por `id`, `category` (ya lo tengo en memoria, en DB es B-tree).
3. Paginación cursor-based en lugar de offset (offset N escanea N filas; cursor usa el último ID).
4. Cache HTTP con `Cache-Control` y ETag para endpoints idempotentes.
5. Métricas (RPS, p95, p99, error rate) y profiling con `pprof`.

El cambio toca un adapter nuevo y `cmd/api/main.go` (1 línea de wiring). Dominio y use cases intactos.

### 13. ¿Cómo portarías si HackerRank no permite subdirectorios?
**Respuesta**: Tengo plan B documentado en `HACKERRANK.md`. Aplano todo a `package main` manteniendo separación lógica por bloques en el archivo (entity → port → use cases → handler → main). Pierde separación a nivel package pero la lógica se preserva. Si la plataforma permite subdirectorios, mantengo la estructura nativa.

### 14. ¿Cómo justificás DI manual en lugar de Wire/Fx?
**Respuesta**: Para 4 use cases + 1 repo + 1 router, DI manual son **10 líneas en `main.go`** legibles top-to-bottom. Wire genera código que parece magia en code review; Fx tiene runtime overhead. Si crece a 20+ dependencias, refactorizar a Wire vale la pena — pero ahora es over-engineering. Menos magia, menor costo cognitivo en entrevista y onboarding.

### 15. ¿Cómo evitás exponer campos sensibles al proyectar `fields`?
**Respuesta**: Whitelist positiva en `domain.allowedFields`. Si mañana agrego `cost_price` al struct `Product`, no sale por API hasta que lo enliste explícitamente. Blacklist olvida; whitelist no. Es seguridad-por-defecto.

### 16. ¿Cómo manejarías versionado de API?
**Respuesta**: Hoy no lo necesito (sin clientes existentes). Cuando llegue el primer breaking change real: versionado por path (`/v1/products/...`). La separación por capas ayuda a coexistir comportamientos: handler v1 y v2 invocan use cases compartidos o variantes. Versionar antes de tener clientes es cargo-cult.

### 17. ¿Por qué `errors.Is`/`errors.As` y no comparar strings de error?
**Respuesta**: Porque cambiar el texto de un error nunca debe romper el mapeo HTTP. Sentinel errors (`ErrProductNotFound`, `ErrInvalidField`) + `errors.Is` desacoplan mensaje de identidad. Para errores con datos (`MissingIDsError`), `errors.As` extrae el struct concreto. Bonus: `MissingIDsError.Unwrap()` retorna `ErrProductNotFound`, así un handler genérico puede tratar ambos casos como 404 sin perder el detalle.

### 18. ¿Cómo aislás use cases del repo en tests?
**Respuesta**: Mock manual de `ProductRepository` en `application/mock_repository_test.go`. Implementa la interfaz con campos públicos para configurar respuestas y un contador de llamadas para verificar wiring. Sin gomock, sin generación de código — para 4 métodos es más explícito y legible. Cada test de use case configura su mock con la condición exacta que prueba.

### 19. ¿Por qué el orden de rutas importa en Gin?
**Respuesta**: Gin matchea rutas estáticas vs. dinámicas por orden de registro. Si registro `/products/:id` ANTES de `/products/categories`, la palabra "categories" cae como `id="categories"` en el handler `Get`. Por eso registro primero las estáticas (`/compare`, `/categories`) y después la dinámica (`/:id`). Hay un test específico que defiende este orden: `TestRouter_StaticCategoriesRouteTakesPrecedenceOverDynamicID`.

### 20. ¿Qué pasa si `crypto/rand` falla al generar el request_id?
**Respuesta**: Fallback a un timestamp con nanosegundos (`150405.000000000`). No bloqueo la request por algo accesorio como un ID. Es extremadamente raro que `crypto/rand` falle en Linux/Mac, pero defense-in-depth: prefiero un ID degradado a un 500.

### 21. ¿Cuál fue el criterio principal de diseño?
**Respuesta**: Priorizar **confiabilidad del contrato + portabilidad de ejecución**. En entrevista vale más una solución coherente, testeada y defendible que complejidad innecesaria o features no pedidas. Cada decisión la puedo justificar con un trade-off explícito; no hay nada por dogma.

### 22. ¿Qué dejaste afuera intencionalmente y por qué?
**Respuesta**: Auth/JWT (no hay clientes definidos), rate limiting (debería estar en el edge — LB/API Gateway de MELI), CORS (server-to-server por default), cache (catálogo chico), métricas Prometheus (requiere stack adicional), OpenAPI (5 endpoints, README curado comunica más rápido), versionado (sin breaking changes reales), tracing distribuido (servicio único, request_id alcanza), DB real (el enunciado dice "no se requiere"). Todo está documentado en `README.md` con el "cómo se agregaría" — el diseño admite cualquiera como adapter sin reescribir dominio. **Saber cuándo NO hacer algo es tan senior como saber hacerlo**.

---

## 5) Red flags: qué NO decir

- "Elegí X porque estaba de moda."
- "No hice tests porque no llegaba."
- "Si falta un producto, da igual devolver parcial."
- "Metí todo en handlers porque era más rápido."
- "No pensé límites de entrada."
- "Los logs están por si acaso, pero no los uso."
- "No consideré el entorno de HackerRank."
- "Clean Architecture siempre, en cualquier caso, sin trade-offs."
- "Le metí Wire/Fx porque queda mejor."
- "Validé everything en el handler porque es más simple."
- "El repo no necesita mutex porque solo lee."

**Regla práctica**: nunca vendas dogma. Mostrá criterio contextual con trade-off explícito.

---

## 6) Estrategia de demo en vivo (10-12 min)

### Min 0-1: encuadre
- "API REST en Go + Gin para comparación de productos. Foco: contrato confiable, testing, hardening, portabilidad."
- Mostrar `README.md` (estructura de capas y endpoints en una pantalla).

### Min 1-3: arquitectura
- Abrir el árbol de `internal/`: `domain` → `application` → `infrastructure/{http,persistence}`, `cmd/api/main.go`.
- Marcar la regla de dependencias: domain solo importa stdlib (mostrar imports).
- Mostrar `domain/repository.go` (port) y `cmd/api/main.go:80-87` (wiring de 10 líneas).

### Min 3-7: contrato API en vivo
Tener el server corriendo (`make run`). Ejecutar:

```bash
# 1. Happy path con proyección selectiva
curl -s "localhost:8080/products/compare?ids=1,2&fields=name,price,rating" | jq

# 2. Validación 400 (campo fuera de whitelist)
curl -sw "\nHTTP %{http_code}\n" "localhost:8080/products/compare?ids=1,2&fields=cost_price"

# 3. 404 con missing_ids (defendible)
curl -sw "\nHTTP %{http_code}\n" "localhost:8080/products/compare?ids=1,99"

# 4. Correlación con request_id
curl -i -H "X-Request-Id: demo-123" "localhost:8080/products/compare?ids=1,2&fields=name"
# Mostrar el header en respuesta + el log JSON con request_id=demo-123

# 5. Cap de page size
curl -sw "\nHTTP %{http_code}\n" "localhost:8080/products?page=-1"
```

### Min 7-9: calidad técnica
- `make test` → 78 tests verde.
- `make cover` → coverage por paquete (resaltar 95-100% en negocio).
- Mostrar un test representativo: `TestCompareProducts_DeduplicatesIDs` (use case con mock).

### Min 9-11: trade-offs y portabilidad
- Explicar 404 todo-o-nada (la decisión más defendible).
- Explicar por qué JSON in-memory y cómo migrar a Postgres (1 adapter nuevo, 1 línea en main).
- Mostrar `HACKERRANK.md` con el plan B para single-file o sin Gin.

### Min 11-12: cierre
- Resumir 3 decisiones fuertes:
  1. **Clean Architecture liviana** → tests aislados, adapters intercambiables.
  2. **404 + missing_ids en compare** → contrato honesto con el cliente.
  3. **Hardening explícito** → timeouts, caps, graceful shutdown, fail-fast.
- "Lo que NO está incluido está documentado con el cómo se agregaría — diseño admite extensiones sin reescribir negocio."

---

## 7) Mini plan de estudio / rehearsal (30 min)

### Bloque A (8 min) — discurso base
- Practicar speech de 90s (3 repeticiones).
- Practicar speech de 30s (2 repeticiones).
- Ajustar muletillas, eliminar adjetivos vacíos.

### Bloque B (10 min) — preguntas difíciles
- Elegir 8 preguntas de la sección 4 (mezclar técnicas profundas con trade-offs).
- Responder en voz alta en formato: **decisión → por qué → trade-off**.
- Sugeridas: 4 (404), 5 (dedup en use case), 6 (Specs), 11 (concurrencia), 14 (DI), 17 (errors.Is), 19 (orden de rutas), 22 (lo que dejé afuera).

### Bloque C (8 min) — demo
- Simular demo completa con cronómetro y server real.
- Objetivo: no superar 12 minutos y cubrir 5 requests clave + tests + coverage.

### Bloque D (4 min) — cierre ejecutivo
- Practicar cierre de 30s con: arquitectura, confiabilidad del contrato, portabilidad.

**Checklist de salida**
- [ ] Puedo explicar 5 decisiones sin leer
- [ ] Puedo defender al menos 3 trade-offs no obvios (404 todo-o-nada, dedup en use case, mutex en repo read-only)
- [ ] Puedo responder "¿qué harías en producción?" en 60s
- [ ] Puedo justificar por qué la solución es portable a HackerRank
- [ ] Tengo el server corriendo y los curls memorizados
- [ ] `make test` y `make cover` listos para mostrar

---

## 8) Anti-cheatsheet — frases poderosas para usar

Cuando defiendas decisiones, estas frases proyectan criterio senior:

- **"Saber cuándo NO hacer algo es tan senior como saber hacerlo."**
- **"Prefiero fricción explícita a decisión silenciosa."** (defensa del 404 todo-o-nada)
- **"La regla de negocio vive en el use case, no en el adapter."** (defensa de dedup en use case)
- **"Whitelist es seguridad por defecto. Blacklist olvida; whitelist no."**
- **"Defense-in-depth, no boundary security."** (defensa de validateProductsPath)
- **"Fail-fast en boot beats fail-soft en runtime."** (defensa de duplicate ID check)
- **"Versionar antes de tener clientes es cargo-cult."**
- **"Cero magia, máxima auditabilidad."** (defensa de DI manual)
- **"Loguear caro a runtime es barato; NO loguear y debuggear ciego es carísimo."**
- **"El dominio no sabe que existe Gin. Esa es la prueba."**

---

## 9) Errores comunes a evitar en la entrevista

1. **Mencionar tecnologías que no usaste** → te van a preguntar y se nota.
2. **Defender Clean Architecture como dogma** → siempre con trade-off contextual.
3. **No saber el coverage real** → 95-100% en negocio, sabés. Decilo con números.
4. **Olvidar que es `/products/compare`, no `/compare`** → leélo dos veces antes de demo.
5. **Decir "para producción haría X" sin priorizar** → ver `PRODUCTION-NEXT-STEPS.md` para el orden razonado.
6. **Vender features no pedidas como ventaja** → "agregué auth" suena a no entender el scope.
7. **Confundir `errors.Is` y `errors.As`** → `Is` para identidad sentinel, `As` para extraer struct con datos.
8. **No reconocer limitaciones** → "JSON in-memory no escala a producción real" es la respuesta correcta, no defenderlo a muerte.

---

## 10) Plan B si te preguntan algo que no sabés

1. **No inventes**. "Buena pregunta, no lo profundicé en este alcance — lo razonaría así: [hipótesis estructurada]."
2. **Reconocé el límite del scope**. "Para este challenge prioricé X. Si tuviera Y horas más, atacaría Z."
3. **Pivotá a algo que sí sabés**. "No usé OpenTelemetry acá, pero el patrón de propagar contexto es similar al request_id que sí implementé."
4. **Hacé la pregunta de vuelta**. "¿En qué escenario lo estás pensando? Para volumen X mi respuesta cambia respecto de volumen Y."

---

> **Recordá**: la entrevista no es un examen de memoria. Es una conversación donde demostrás criterio, trade-offs y capacidad de decisión bajo restricciones. La solución no tiene que ser perfecta — tiene que ser **defendible y coherente**.
