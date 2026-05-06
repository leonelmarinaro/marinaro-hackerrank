No actualicé el archivo en GitHub. Te dejo una versión final lista para reemplazar `INTERVIEW-NARRATIVE.md`. La escribí para que puedas estudiarla casi literal y defender tanto las decisiones fuertes como las cosas que, con más tiempo o en producción, harías distinto.


# INTERVIEW-NARRATIVE — Defensa técnica del challenge

Guía final para defender el challenge de backend API de comparación de productos.

Objetivo de esta narrativa: explicar la solución con criterio técnico, sin vender humo, mostrando decisiones, trade-offs, límites del diseño y próximos pasos razonables.

---

# 1. Speech principal — 90 segundos

Diseñé la solución como una API REST en Go para resolver una funcionalidad de comparación de productos. El foco principal fue construir un backend simple de ejecutar, fácil de revisar y con separación clara entre reglas de negocio e infraestructura.

A nivel arquitectónico usé una Clean Architecture liviana. Separé el código en cuatro zonas: `domain`, `application`, `infrastructure` y `cmd/api`. El dominio contiene el modelo de producto, errores y contratos internos; la capa de aplicación concentra los casos de uso; infraestructura adapta HTTP con Gin y persistencia JSON; y `cmd/api` funciona como composition root, donde se conectan las dependencias.

La decisión más importante fue que la lógica de negocio no dependa de Gin, del router ni del archivo JSON. Por ejemplo, el caso de uso de comparación recibe IDs y campos, valida la entrada, deduplica IDs, aplica el límite máximo y ejecuta la comparación. Esa lógica está fuera del handler HTTP, por lo que podría reutilizarse desde otro adapter, como CLI, gRPC o un endpoint implementado con `net/http`.

En `/products/compare` elegí una semántica todo-o-nada: si el cliente pide comparar `1,2,99` y el producto `99` no existe, devuelvo `404` con `missing_ids`. La razón es que una comparación parcial puede ser engañosa. Prefiero fallar explícitamente y decirle al cliente qué falta antes que devolver un `200` con datos incompletos.

Para persistencia usé un JSON cargado en memoria al inicio. Lo elegí porque el enunciado no pedía base de datos real, y para un challenge permite portabilidad, rapidez y cero dependencias externas. Aun así, lo encapsulé detrás de un puerto `ProductRepository`, de modo que migrar a PostgreSQL, Redis o cualquier otro storage implicaría implementar otro adapter sin reescribir los casos de uso.

En no funcionales prioricé tests por capa, manejo explícito de errores, validaciones defensivas, logs estructurados con `slog`, `X-Request-Id`, timeouts del servidor HTTP y graceful shutdown. No lo planteo como una arquitectura final de producción, pero sí como una solución coherente para un challenge: portable, testeable, legible y con trade-offs explícitos.

---

# 2. Versión corta — 30 segundos

Resolví el challenge con una API REST en Go + Gin, usando una Clean Architecture liviana para separar dominio, casos de uso e infraestructura. El caso central es comparar productos con proyección selectiva de campos. Priorizé contrato claro, validaciones estrictas y tests por capa.

Elegí JSON in-memory porque el enunciado no requería base de datos real, pero lo dejé detrás de un puerto para poder reemplazarlo por una DB sin tocar la lógica de negocio. En `/compare` uso una respuesta todo-o-nada: si falta un producto, devuelvo `404` con `missing_ids`, porque una comparación parcial puede inducir decisiones incorrectas.

---

# 3. Frase de apertura recomendada

"Antes de entrar al código, aclaro el criterio general: no intenté hacer una arquitectura de producción completa, sino una solución de challenge que sea clara, testeable, portable y defendible. Algunas decisiones están optimizadas para el contexto de HackerRank, no necesariamente para un sistema real de alto tráfico."

Esta frase sirve porque baja expectativas irreales y muestra criterio contextual.

---

# 4. Diseño general de la solución

La solución tiene cuatro capas principales.

## 4.1. Domain

Contiene el modelo `Product`, los errores de dominio y el puerto `ProductRepository`.

El dominio no sabe que existe Gin, HTTP, JSON, archivos, base de datos ni sistema operativo. Esto permite testear reglas puras sin levantar infraestructura.

Responsabilidades principales:

- Modelar el producto.
- Definir campos permitidos para proyección.
- Implementar `SelectFields`.
- Definir errores como `ErrProductNotFound`, `ErrInvalidField`, `ErrEmptyIDs`, `ErrTooManyIDs`.
- Definir `MissingIDsError` para devolver detalle de productos faltantes.

Decisión importante: `Specs` está modelado como `map[string]any`.

Defensa:

"Usé campos comunes tipados para lo que todos los productos tienen, como id, nombre, descripción, precio, rating y categoría. Para atributos específicos por categoría usé `Specs map[string]any`, porque el catálogo puede tener smartphones, libros, ropa o electrodomésticos, y cada categoría tiene atributos distintos. El trade-off es que pierdo type-safety en esas specs, pero gano flexibilidad para agregar categorías sin cambiar el schema."

---

## 4.2. Application

Contiene los casos de uso:

- `CompareProductsUseCase`
- `ListProductsUseCase`
- `GetProductUseCase`
- `ListCategoriesUseCase`

El caso más importante es `CompareProductsUseCase`.

Responsabilidades:

- Validar que haya IDs.
- Deduplicar IDs preservando orden.
- Aplicar límite máximo de IDs.
- Validar `fields` contra whitelist.
- Llamar al repository.
- Proyectar los productos según los campos pedidos.

Defensa:

"Puse la deduplicación, el cap de IDs y la validación de fields en application, no en HTTP, porque son reglas de comportamiento de la operación. Si mañana uso otro adapter, quiero que reciba la misma semántica."

---

## 4.3. Infrastructure

Tiene dos adapters:

- HTTP adapter con Gin.
- Persistence adapter con JSON in-memory.

El HTTP adapter:

- Parsea query params.
- Invoca use cases.
- Mapea errores a status HTTP.
- Serializa respuestas.
- Agrega middlewares de logging, request ID y security headers.

El adapter de persistencia:

- Carga el JSON al iniciar.
- Valida IDs vacíos y duplicados.
- Construye índice por ID.
- Responde consultas desde memoria.

Defensa:

"El archivo JSON no está mezclado con los use cases. Está encapsulado detrás de un repository. Eso me permite cambiar storage sin cambiar el comportamiento de negocio."

---

## 4.4. cmd/api

Es el composition root.

Responsabilidades:

- Configurar logger.
- Configurar Gin mode.
- Leer `PRODUCTS_FILE` y `PORT`.
- Validar path del archivo de productos.
- Instanciar repository.
- Instanciar use cases.
- Instanciar handlers.
- Crear router.
- Configurar `http.Server`.
- Manejar graceful shutdown.

Defensa:

"`main.go` sí conoce todas las piezas porque su responsabilidad es conectar dependencias. La lógica de negocio no vive ahí."

---

# 5. Decisiones técnicas y trade-offs

## 5.1. Go como lenguaje

Decisión: usar Go.

Por qué:

- Tipado estático.
- Buen soporte nativo de testing.
- Binario portable.
- Excelente performance para APIs.
- Sintaxis relativamente simple para que el evaluador revise rápido.
- Buen fit para backend y sistemas distribuidos.

Trade-off:

- Menos expresivo que Python para prototipado rápido.
- Manejo de errores más verboso.
- Menos magia de frameworks.

Respuesta modelo:

"Elegí Go porque para este challenge quería una solución portable, rápida de ejecutar y fácil de testear. Además, Go tiene buen fit con servicios backend: tipado fuerte, tooling simple, testing estándar y binarios fáciles de correr. El trade-off es más verbosidad, especialmente en manejo de errores, pero para una API donde quiero claridad y control me parece razonable."

---

## 5.2. Gin como framework HTTP

Decisión: usar Gin.

Por qué:

- Routing simple.
- Middlewares conocidos.
- Menos boilerplate que `net/http`.
- Buena performance.
- Permite implementar rápido sin contaminar el core.

Trade-off:

- Agrega una dependencia externa.
- Trae dependencias indirectas.
- Para un challenge chico, `net/http` también alcanzaba.

Respuesta modelo:

"Usé Gin para acelerar la parte de routing y middleware. No lo elegí porque fuera indispensable; de hecho, el diseño evita que Gin entre en el dominio o en los use cases. Si el entorno no permitiera Gin, podría reescribir solo la capa HTTP con `net/http`, manteniendo dominio, aplicación y persistencia."

Cuidado: no decir "Gin es necesario". No lo es.

---

## 5.3. Clean Architecture liviana

Decisión: separar en `domain`, `application`, `infrastructure` y `cmd`.

Por qué:

- Aísla lógica de negocio.
- Facilita testeo por capa.
- Permite cambiar adapters.
- Hace visible el criterio arquitectónico.

Trade-off:

- Más archivos.
- Más wiring inicial.
- Puede parecer overengineering para un challenge muy chico.

Respuesta modelo:

"No apliqué Clean Architecture como dogma, sino en una versión liviana. Para un CRUD mínimo sería discutible, pero en este challenge quería mostrar separación de responsabilidades, testabilidad y capacidad de evolución. El costo son algunos archivos extra y DI manual, pero el beneficio es que la lógica importante queda desacoplada de Gin y del storage."

---

## 5.4. Puerto `ProductRepository`

Decisión: definir un contrato de repository para que application no dependa de una implementación concreta.

Por qué:

- Permite mockear en tests.
- Permite cambiar JSON por DB.
- Mantiene application desacoplada de infraestructura.

Trade-off:

- Una interfaz más para un caso simple.
- En Go conviene no abusar de interfaces.
- Podría ubicarse en `application/ports` en vez de `domain`.

Respuesta honesta:

"Definí el puerto como una interfaz interna para desacoplar los use cases del adapter JSON. En este repo vive en `domain`, lo cual es defendible porque es una capa interna. Dicho eso, en un sistema más grande también sería razonable moverlo a `application/ports`, porque quien realmente consume ese puerto son los casos de uso. No lo defendería como una verdad universal, sino como una decisión aceptable para este tamaño."

Esta respuesta es muy buena porque muestra madurez.

---

## 5.5. JSON in-memory

Decisión: cargar `products.json` al boot y servir desde memoria.

Por qué:

- El enunciado no pide DB real.
- Evita setup externo.
- Hace la solución portable para HackerRank.
- Permite demo simple.
- Evita I/O por request.

Trade-off:

- No hay persistencia runtime.
- Si cambia el archivo, requiere restart.
- No escala si el catálogo crece mucho.
- No hay queries complejas ni índices reales más allá del map por ID.

Respuesta modelo:

"Elegí JSON in-memory porque el challenge permite simular persistencia. Para este contexto, agregar PostgreSQL hubiera sido ruido operativo. Aun así, lo encapsulé detrás de un repository. Si esto fuera producción, cambiaría el adapter por una DB real o cache sin modificar los use cases."

---

## 5.6. Índice por ID

Decisión: construir `map[id]Product` al cargar el JSON.

Por qué:

- `FindByID` queda O(1).
- `FindByIDs` evita escanear todo el catálogo.
- La carga inicial valida duplicados.

Trade-off:

- Duplica algo de memoria.
- Requiere reconstruir índice si hubiera escrituras.
- Para un dataset chico no era estrictamente necesario.

Respuesta modelo:

"Aunque el dataset sea chico, indexar por ID es una mejora simple y barata. El costo de memoria es bajo y el beneficio es que el comportamiento del repository ya se parece más a una consulta eficiente."

---

## 5.7. `Specs` como `map[string]any`

Decisión: specs flexibles.

Por qué:

- Productos heterogéneos.
- Permite smartphones, libros, ropa, electrodomésticos.
- Evita crear structs por categoría.

Trade-off:

- Menor type-safety.
- Errores de nombres o tipos se detectan en runtime.
- Más difícil documentar schema exhaustivo.
- Si el dominio creciera, requeriría validaciones específicas.

Respuesta modelo:

"Lo elegí por flexibilidad. Para los campos comunes mantuve tipado fuerte, pero para atributos específicos por categoría acepté `map[string]any`. En producción, si ciertas categorías fueran críticas, probablemente modelaría specs tipadas por categoría o validaría contra schemas."

---

## 5.8. Whitelist de fields

Decisión: permitir solo campos explícitamente autorizados.

Por qué:

- Evita exponer campos internos futuros.
- Hace estable el contrato público.
- Evita que cualquier campo agregado al struct salga automáticamente por API.

Trade-off:

- Hay que mantener la whitelist.
- Si agrego un campo público, debo acordarme de habilitarlo.
- Es menos dinámico.

Respuesta modelo:

"Preferí whitelist antes que blacklist porque es más segura por defecto. Si mañana aparece `cost_price` o `internal_sku`, no queda expuesto accidentalmente. El trade-off es mantenimiento manual, pero para APIs públicas prefiero explicitud."

---

## 5.9. `/compare` todo-o-nada con 404

Decisión: si falta algún ID, devolver error con `missing_ids`.

Por qué:

- Comparar implica que todos los productos son relevantes.
- Un 200 parcial puede ser engañoso.
- El cliente sabe exactamente qué IDs corregir.

Trade-off:

- El cliente tiene que reintentar.
- En algunos productos podría preferirse respuesta parcial.
- No es la única opción válida.

Respuesta modelo:

"Elegí semántica todo-o-nada porque una comparación parcial puede inducir una decisión incorrecta. Si el usuario quiere comparar tres productos y devuelvo dos con 200, quizás no advierte que falta uno. Por eso prefiero 404 con `missing_ids`. Reconozco que otra API podría devolver 200 con `items` y `missing_ids`; sería válido si el producto priorizara tolerancia parcial. Yo prioricé contrato estricto."

Esta es una respuesta excelente.

---

## 5.10. Deduplicación de IDs

Decisión: `ids=1,1,2` devuelve `1,2`.

Por qué:

- Comparar el mismo producto consigo mismo no aporta.
- Evita respuestas duplicadas.
- Reduce trabajo innecesario.
- Preserva orden de primera aparición.

Trade-off:

- Si el cliente esperaba duplicados, no los recibe.
- Hay una decisión implícita de normalización.

Respuesta modelo:

"Deduplico preservando orden porque una comparación con IDs repetidos no aporta valor y puede confundir la UI. Lo hago en application para que la semántica sea igual aunque el request venga por otro adapter."

---

## 5.11. Cap de IDs y tamaño de página

Decisión: limitar cantidad máxima de IDs en compare y tamaño de página.

Por qué:

- Defensa básica contra abuso.
- Evita requests absurdos.
- Protege memoria y latencia.
- Mejora UX.

Trade-off:

- Un cliente legítimo con muchos productos debe paginar o dividir requests.
- El valor del límite es arbitrario y debería ajustarse con métricas reales.

Respuesta modelo:

"No lo llamaría rate limiting, sino un guardrail. Es un límite defensivo por request. Si el volumen real justificara comparar más productos, lo haría por batching, paginación o un endpoint distinto."

---

## 5.12. Validación estricta de paginación en HTTP

Decisión: si el cliente manda `page=-1` o `size=abc`, devolver 400.

Por qué:

- Evita silent defaults.
- Hace visible el error del cliente.
- Mantiene contrato claro.

Trade-off:

- Menos tolerante.
- Algunos clientes prefieren que se normalice automáticamente.

Respuesta modelo:

"Si el query param no viene, aplico defaults. Pero si viene explícitamente inválido, devuelvo 400. Eso evita que un cliente con un bug crea que está consultando una página cuando en realidad le devuelvo otra."

---

## 5.13. Logs estructurados con `slog` y `X-Request-Id`

Decisión: usar logs JSON con request ID.

Por qué:

- Correlación de requests.
- Mejor debugging.
- Sin dependencia externa porque `slog` está en stdlib moderna.
- Permite ver método, path, status, duración y errores internos.

Trade-off:

- Más verbosidad.
- No reemplaza tracing distribuido.
- Request ID no es lo mismo que OpenTelemetry.

Respuesta modelo:

"Para un servicio único, `request_id` y logs estructurados son suficientes. Si fuera un sistema distribuido, agregaría OpenTelemetry y propagación W3C Trace Context."

---

## 5.14. Timeouts y graceful shutdown

Decisión: configurar `http.Server` con timeouts y shutdown ante señales.

Por qué:

- Evita conexiones lentas indefinidas.
- Permite cerrar sin cortar requests en vuelo.
- Es una práctica básica de runtime robusto.

Trade-off:

- Valores elegidos son aproximados.
- En producción deberían ajustarse con SLOs y métricas.
- No reemplaza configuración de proxy o load balancer.

Respuesta modelo:

"Los timeouts no son una optimización de performance, son higiene operativa. Los valores son razonables para JSON chico, pero en producción los ajustaría según latencia real y comportamiento de clientes."

---

## 5.15. DI manual

Decisión: conectar dependencias a mano en `main.go`.

Por qué:

- Pocas dependencias.
- Código explícito.
- Sin framework de inyección.
- Fácil de revisar.

Trade-off:

- Si el proyecto crece mucho, el wiring puede volverse extenso.
- No hay generación ni contenedor.

Respuesta modelo:

"Para este tamaño, DI manual es más claro que Wire o Fx. Si tuviera decenas de servicios y adapters, evaluaría generación de código o un patrón de bootstrap más estructurado."

---

# 6. Puntos que reconozco que podrían mejorarse

Esta sección es clave. Reconocer límites bien formulados suma puntos.

## 6.1. Agregaría `context.Context` en los puertos de I/O

Respuesta:

"Una mejora que haría si esto evolucionara a producción sería agregar `context.Context` a los métodos del repository. En el JSON in-memory no era crítico, pero para una DB real permitiría cancelación, timeouts y propagación de request scope."

Ejemplo:


FindByID(ctx context.Context, id string) (*Product, error)
FindByIDs(ctx context.Context, ids []string) ([]Product, error)


Por qué suma:

Muestra que entendés backend real.

---

## 6.2. Revisaría ubicación del puerto `ProductRepository`

Respuesta:

"Hoy el puerto vive en `domain`. Es defendible porque está en una capa interna, pero en un diseño más purista probablemente lo pondría en `application/ports`, porque quien realmente necesita persistencia son los casos de uso, no la entidad de dominio."

Por qué suma:

Muestra que no sos dogmático.

---

## 6.3. No vendería esto como producción completa

Respuesta:

"Esto está pensado para un challenge. Para producción faltaría auth si aplica, métricas, tracing, rate limiting en edge o distribuido, CI/CD real, configuración por entorno, DB/cache y posiblemente versionado de API cuando haya clientes reales."

Por qué suma:

Mostrás criterio de alcance.

---

## 6.4. `Specs map[string]any` requiere cuidado

Respuesta:

"`map[string]any` me da flexibilidad, pero si el dominio creciera agregaría validaciones por categoría o schemas. Para productos críticos no dejaría toda la semántica en un mapa libre."

---

## 6.5. El repository no hace deep copy de `Specs`

Respuesta:

"Como la API es read-only y los callers no mutan el producto, acepté shallow copies. Pero si agregara operaciones mutables o más consumidores internos, haría deep copy de `Specs` para evitar compartir mapas mutables."

---

## 6.6. Los números exactos de tests deben verificarse

Respuesta:

"Prefiero hablar de tests por capa y casos cubiertos antes que depender de un número exacto. Antes de una entrega formal correría `go test ./... -v` y actualizaría el README con el conteo real."

---

## 6.7. Versionado de API

Respuesta:

"No agregué `/v1` porque no hay clientes reales ni breaking changes. En producción, cuando exista contrato público estable o evolución de endpoints, sí versionaría."

---

# 7. Preguntas difíciles y respuestas modelo

## 7.1. ¿No está sobrediseñado para un challenge?

Respuesta:

"Puede parecerlo si uno mira solo el tamaño del problema. Por eso lo mantuve liviano: no usé frameworks de DI, no agregué DB, no agregué colas ni observabilidad pesada. La separación en capas me permitió testear bien y mostrar cómo evolucionaría sin acoplar negocio a Gin o JSON. El trade-off son más archivos, pero el costo es bajo y la revisión sigue siendo clara."

---

## 7.2. ¿Por qué no hiciste todo en un solo handler?

Respuesta:

"Porque el handler debería adaptar HTTP, no contener reglas de negocio. Si metía validación de fields, dedup, cap de IDs y semántica de comparación dentro del handler, el código sería más rápido de escribir, pero menos testeable y menos reutilizable. Separarlo me permite probar el caso de uso sin levantar HTTP."

---

## 7.3. ¿Por qué no usaste base de datos?

Respuesta:

"Porque el enunciado no lo requería y agregar una DB hubiera aumentado setup y complejidad operativa sin mejorar la evaluación central. El objetivo era mostrar API design, errores, testing y arquitectura. Aun así, dejé la persistencia detrás de un puerto para poder reemplazar JSON por DB."

---

## 7.4. ¿Qué cambiarías si esto tuviera millones de productos?

Respuesta:

"Cambiaría el adapter JSON por una base de datos o servicio de catálogo. Agregaría índices reales, cache para lecturas frecuentes, paginación cursor si aplica, métricas, profiling y posiblemente un endpoint batch optimizado. También agregaría `context.Context` al puerto de repositorio."

---

## 7.5. ¿Por qué 404 y no 200 con resultados parciales?

Respuesta:

"Porque para mí comparar es una operación sobre un conjunto completo. Si el cliente pidió comparar tres productos y devuelvo dos con 200, puedo inducir una decisión incorrecta. Con 404 y `missing_ids`, el cliente sabe exactamente qué corregir. Reconozco que un 200 parcial también podría ser válido en otro contexto, por ejemplo si el producto priorizara tolerancia a fallos, pero acá prioricé consistencia del contrato."

---

## 7.6. ¿Por qué `map[string]any` si Go es tipado?

Respuesta:

"Porque el dominio tiene campos comunes tipados y campos específicos por categoría. Para los comunes mantuve type-safety. Para specs preferí flexibilidad, porque un smartphone, un libro y una prenda tienen atributos distintos. El trade-off es validación en runtime. En producción, si esas specs fueran críticas, agregaría validación por schema o tipos por categoría."

---

## 7.7. ¿Cómo evitás exponer datos sensibles con `fields`?

Respuesta:

"Uso whitelist. El cliente solo puede pedir campos explícitamente permitidos. Si mañana agrego un campo interno al struct, no se expone automáticamente. Prefiero seguridad por defecto aunque implique mantener la lista."

---

## 7.8. ¿Cómo sabés que la lógica no depende de HTTP?

Respuesta:

"Porque los use cases reciben tipos simples y dependen del puerto `ProductRepository`. No reciben `gin.Context`, no conocen status codes y no serializan JSON. Además, los tests de application prueban las reglas sin levantar servidor."

---

## 7.9. ¿Qué cubren los tests?

Respuesta:

"Hay tests por capa. En domain se prueba proyección y whitelist. En application se prueban reglas como empty IDs, invalid fields, dedup, cap de IDs y propagación de errores. En persistence se prueba carga de JSON, duplicados, IDs vacíos, búsqueda y paginación. En HTTP se prueba contrato externo con `httptest`: happy path, 400, 404, paginación y casos de query params."

No digas un número exacto salvo que lo hayas corrido antes.

---

## 7.10. ¿Qué medidas de seguridad tiene?

Respuesta:

"No lo vendería como seguridad completa. Tiene hardening básico: validación de inputs, límites por request, errores internos no expuestos, security headers simples, timeouts del servidor y trusted proxies deshabilitados por defecto. Para producción faltaría auth si aplica, rate limiting real, CORS si hubiera browser clients y controles en el edge."

---

## 7.11. ¿Por qué no agregaste Swagger/OpenAPI?

Respuesta:

"Para cinco endpoints, un README cuidado era suficiente y más rápido de revisar. Si el API creciera o tuviera consumidores externos, agregaría OpenAPI para contrato formal y generación de documentación."

---

## 7.12. ¿Por qué no agregaste CORS?

Respuesta:

"Porque no hay frontend definido ni cliente browser en el enunciado. Prefiero no habilitar CORS por defecto. Si apareciera un frontend, agregaría middleware con whitelist de orígenes."

---

## 7.13. ¿Por qué no agregaste auth?

Respuesta:

"El enunciado no define usuarios, clientes ni permisos. Agregar JWT hubiera sido especulativo. Si fuera necesario, lo agregaría como middleware en infraestructura HTTP, sin afectar dominio ni casos de uso."

---

## 7.14. ¿Por qué no agregaste métricas Prometheus?

Respuesta:

"Porque para un challenge agregaría dependencia y superficie adicional. Sí agregué logs estructurados y request ID. En producción, Prometheus u OpenTelemetry serían próximos pasos razonables."

---

## 7.15. ¿Qué harías si HackerRank no soporta Gin?

Respuesta:

"Reescribiría solo la capa HTTP con `net/http`. El dominio, los use cases y el adapter de persistencia quedarían iguales. Ese es justamente el beneficio de que Gin no haya entrado en la lógica de negocio."

---

## 7.16. ¿Qué harías si HackerRank no soporta múltiples archivos?

Respuesta:

"Aplanaría todo a `package main`, manteniendo separación lógica por bloques: dominio, errores, repository, use cases, handlers y main. Sería un trade-off impuesto por la plataforma, no una decisión de diseño ideal."

---

## 7.17. ¿Dónde está la lógica más importante?

Respuesta:

"En `CompareProductsUseCase`. Ahí está la semántica principal del challenge: IDs requeridos, deduplicación, límite máximo, whitelist de fields, consulta al repository y proyección selectiva."

---

## 7.18. ¿Qué parte te parece más discutible de tu diseño?

Respuesta:

"Dos cosas. Primero, ubicar el puerto `ProductRepository` en domain; es defendible, pero en un sistema más grande podría moverlo a application/ports. Segundo, `Specs map[string]any`; es flexible, pero sacrifica type-safety. Las elegí por el alcance del challenge, no porque sean universalmente superiores."

Esta respuesta es muy buena para evaluador senior.

---

# 8. Cosas que NO conviene decir

No decir:

"Usé Clean Architecture porque siempre hay que usarla."

Mejor:

"Usé una versión liviana porque en este caso quería testabilidad y desacoplamiento sin meter frameworks pesados."

---

No decir:

"Esto está listo para producción."

Mejor:

"Está bien armado para un challenge. Para producción agregaría context, métricas, DB/cache, rate limiting, CI/CD y decisiones de seguridad según contexto."

---

No decir:

"JSON in-memory escala."

Mejor:

"JSON in-memory es correcto para el alcance del challenge; para volumen real reemplazaría el adapter."

---

No decir:

"Gin es necesario."

Mejor:

"Gin reduce boilerplate, pero el diseño permite reemplazarlo."

---

No decir:

"404 es objetivamente mejor que 200 parcial."

Mejor:

"Elegí 404 porque prioricé consistencia de comparación; 200 parcial también sería defendible con otro contrato."

---

No decir:

"TDD estricto en todo el proyecto."

Mejor:

"Trabajé con enfoque test-first en las reglas críticas y dejé cobertura por capa."

---

No decir:

"El sistema tiene seguridad."

Mejor:

"Tiene hardening básico. Seguridad completa dependería de auth, entorno, gateway, rate limits y threat model."

---

No decir:

"El request ID es tracing distribuido."

Mejor:

"Es correlación básica. Para tracing distribuido agregaría OpenTelemetry."

---

# 9. Demo recomendada en entrevista

## Minuto 0-1: objetivo

"El endpoint central es comparar productos. El usuario puede pedir IDs y elegir campos relevantes."

Mostrar:

```bash
GET /products/compare?ids=1,2&fields=name,price,rating
```

Explicar:

"Esto evita traer campos innecesarios y permite comparar solo dimensiones relevantes."

---

## Minuto 1-3: arquitectura

Mostrar estructura:

```txt
cmd/api
internal/domain
internal/application
internal/infrastructure/http
internal/infrastructure/persistence
```

Decir:

"Lo importante no son las carpetas por sí mismas, sino la dirección de dependencias: infraestructura conoce aplicación y dominio; dominio no conoce infraestructura."

---

## Minuto 3-5: caso de uso principal

Mostrar `CompareProductsUseCase`.

Decir:

"Acá está la lógica relevante: validación de IDs, dedup, cap, whitelist de fields y semántica todo-o-nada."

---

## Minuto 5-6: errores

Mostrar `MissingIDsError` y `writeError`.

Decir:

"Uso errores tipados para que HTTP no dependa de strings. El handler mapea errores de dominio a status codes."

---

## Minuto 6-8: tests

Mostrar tests de application y HTTP.

Decir:

"Los tests de application validan reglas sin servidor. Los tests HTTP validan el contrato real vía `httptest`."

---

## Minuto 8-10: trade-offs

Decir:

"JSON in-memory es una decisión de challenge, no de producción. Pero al estar detrás de un puerto puedo reemplazarlo por DB. También reconozco que agregaría context en los puertos si fuera I/O real."

---

# 10. Cierre recomendado

"En resumen, busqué una solución proporcional al challenge: no una arquitectura enterprise, pero tampoco un handler gigante. La parte más importante es que las decisiones están aisladas: Gin está en infraestructura, JSON está en persistencia, y la lógica de comparación está en application. Eso hace que el código sea testeable, portable y fácil de evolucionar. Los principales trade-offs fueron aceptar más archivos por separación, usar JSON in-memory por portabilidad y usar specs flexibles para productos heterogéneos. Si esto evolucionara a producción, priorizaría context en puertos, DB/cache, métricas, tracing y rate limiting real."

---

# 11. Checklist mental antes de defender

Puedo explicar:

* Por qué Go.
* Por qué Gin, y cómo lo reemplazaría.
* Por qué Clean Architecture liviana.
* Por qué JSON in-memory.
* Por qué `/compare` es todo-o-nada.
* Por qué whitelist de fields.
* Por qué `Specs map[string]any`.
* Cómo están separados handler, use case y repository.
* Qué cubren los tests.
* Qué falta para producción.
* Qué decisión cambiaría con más tiempo.

---

# 12. Respuesta final si preguntan “¿estás conforme con la solución?”

"Sí, para el contexto del challenge estoy conforme. Creo que logra un buen equilibrio entre simplicidad y calidad técnica. No intenté simular una plataforma completa, pero sí dejé claras las decisiones que más importan en backend: contrato API, separación de responsabilidades, manejo de errores, testabilidad, validaciones y portabilidad. También tengo claros los límites: para producción agregaría context en I/O, métricas, tracing, rate limiting real y una persistencia más robusta."

```

Mi recomendación adicional: no estudies esto como recitado palabra por palabra. Estudiá esta estructura:

**decisión → por qué → trade-off → qué haría en producción**.

Ese patrón te va a servir aunque el entrevistador te saque del guion.
```
