# Guía de narrativa de entrevista (HackerRank)

Material para estudiar y defender el challenge frente a un evaluador técnico exigente.

---

## 1) Speech de 90 segundos

> Diseñé esta solución como una API REST en Go con una Clean Architecture liviana para separar bien dominio, casos de uso e infraestructura. La decisión clave fue mantener la lógica de negocio aislada de Gin y de la persistencia JSON, así puedo cambiar framework HTTP o storage sin tocar reglas de negocio.
>
> El caso principal es comparar productos con proyección selectiva de campos. Ahí prioricé tres cosas: validaciones estrictas, contrato claro y comportamiento predecible. Por ejemplo, en `/compare` elegí 404 cuando faltan IDs, en lugar de devolver 200 parcial, porque comparar con datos incompletos puede inducir decisiones erróneas del cliente.
>
> En no funcionales, reforcé testing, observabilidad y hardening. Hay cobertura por capas con tests de dominio, casos de uso, repositorio y handlers HTTP con `httptest`; logs estructurados con `slog` y `request_id` para trazabilidad; y límites defensivos como cap de paginación y cap de IDs para evitar abuso.
>
> También cuidé portabilidad para HackerRank: persistencia en JSON en memoria, dependencias mínimas, DI manual y guía explícita de migración por capas. Si la plataforma no permite Gin o múltiples archivos, el diseño permite degradar a `net/http` o a archivo único con impacto acotado y trade-offs transparentes.

---

## 2) Versión corta (30 segundos)

> Resolví el challenge con Go + Gin y Clean Architecture para desacoplar negocio de infraestructura. Priorizé contrato API confiable, tests por capas y hardening básico. En `/compare` uso 404 todo-o-nada cuando faltan productos para evitar respuestas engañosas. Además dejé la solución portable para HackerRank: JSON in-memory, pocas dependencias y plan B claro para `net/http` o single-file.

---

## 3) Decisiones defendibles (con trade-offs)

## 1. Clean Architecture liviana
- **Decisión**: separar `domain`, `application`, `infrastructure`, `cmd`.
- **Por qué**: aislar reglas de negocio y facilitar testeo.
- **Trade-off**: más archivos y wiring manual inicial.

## 2. Puerto `ProductRepository` en dominio
- **Decisión**: el dominio define el contrato de persistencia.
- **Por qué**: inversión de dependencias real.
- **Trade-off**: más interfaces/abstracciones para un challenge chico.

## 3. Persistencia JSON in-memory al boot
- **Decisión**: cargar catálogo una vez y servir desde memoria.
- **Por qué**: portabilidad, velocidad de demo, cero dependencia externa.
- **Trade-off**: cambios en archivo requieren restart; no hay durabilidad runtime.

## 4. `Specs` como `map[string]any`
- **Decisión**: modelar atributos específicos por categoría de forma flexible.
- **Por qué**: catálogo heterogéneo (smartphones, libros, ropa, etc.).
- **Trade-off**: menor type-safety de compile-time en specs.

## 5. `/compare` con 404 todo-o-nada
- **Decisión**: si falta cualquier ID, devolver error con `missing_ids`.
- **Por qué**: evita comparación parcial silenciosa.
- **Trade-off**: el cliente debe corregir/reintentar la query.

## 6. Whitelist de `fields`
- **Decisión**: sólo permitir campos explícitamente autorizados.
- **Por qué**: seguridad por defecto y contrato estable.
- **Trade-off**: mantenimiento de lista al agregar campos nuevos.

## 7. Hardening básico (caps y validaciones)
- **Decisión**: límite de IDs en compare y límite de tamaño de página.
- **Por qué**: reducir riesgo de abuso/DoS y proteger latencia.
- **Trade-off**: requests extremos legítimos requieren paginación o batch.

## 8. Observabilidad con `slog` + `X-Request-Id`
- **Decisión**: logs JSON estructurados y correlación por request.
- **Por qué**: trazabilidad rápida en debugging/soporte.
- **Trade-off**: mayor verbosidad de logs y disciplina para mantener contexto.

---

## 4) Preguntas probables del evaluador + respuestas modelo

## 1. ¿Por qué Go para este challenge?
**Respuesta**: Por tipado fuerte, testing estándar sólido, latencia baja y binario portable. Para un challenge evaluado en HackerRank, reduce incertidumbre de entorno y acelera feedback.

## 2. ¿Por qué Gin y no `net/http` puro?
**Respuesta**: Gin me da routing y middleware con menos boilerplate. Igual, el diseño desacopla handlers de negocio; migrar a `net/http` es de impacto local en infraestructura.

## 3. ¿No es overengineering usar Clean Architecture acá?
**Respuesta**: Es una versión liviana: pocas capas y DI manual. El costo extra es bajo y me permite mostrar criterio de separación, testeo y mantenibilidad sin frameworks pesados.

## 4. ¿Por qué 404 en compare cuando falta un producto?
**Respuesta**: Porque la comparación depende del conjunto completo. Un 200 parcial puede ocultar faltantes y romper decisiones del usuario. Devuelvo `missing_ids` para que el cliente actúe explícitamente.

## 5. ¿Qué riesgos tiene `map[string]any` en `Specs`?
**Respuesta**: Menor type-safety y más validación en runtime. Lo acepto por flexibilidad del catálogo y porque el contrato principal del endpoint sigue controlado por whitelist de campos.

## 6. ¿Cómo garantizás que la lógica de negocio no depende de HTTP?
**Respuesta**: Los use cases trabajan con tipos de dominio y puertos, no con `gin.Context`. Los tests de application prueban reglas sin levantar servidor.

## 7. ¿Cómo estructuraste el testing?
**Respuesta**: Tests por capa: dominio, casos de uso, repositorio y HTTP con `httptest`. Esto permite detectar si el problema está en reglas, adapter o wiring.

## 8. ¿Qué hiciste en observabilidad concretamente?
**Respuesta**: Logging estructurado con `slog` y `request_id` propagado por header/respuesta. Así correlaciono request, errores y timings en una misma traza lógica.

## 9. ¿Qué medidas de hardening incluíste?
**Respuesta**: Validación estricta de query params, caps defensivos (`size`, cantidad de `ids`), manejo de errores sin filtrar detalles internos y headers de seguridad en middleware.

## 10. ¿Qué harías si el volumen crece 100x?
**Respuesta**: Cambiaría adapter de persistencia por DB/cache, manteniendo use cases. Agregaría profiling, índices, paginación cursor y métricas. La arquitectura ya separa el punto de cambio.

## 11. ¿Cómo portarías si HackerRank no permite subdirectorios?
**Respuesta**: Aplano a package `main` manteniendo separación lógica por bloques y explicito el trade-off. Si permite subdirectorios, mantengo estructura actual por claridad.

## 12. ¿Cómo justificás DI manual?
**Respuesta**: Para este tamaño, DI manual es más legible y auditable que incorporar un contenedor. Menos magia, menor costo cognitivo en entrevista.

## 13. ¿Cómo evitás exponer campos sensibles al proyectar `fields`?
**Respuesta**: Con whitelist positiva. Si mañana agrego un campo interno al modelo, no sale por API hasta habilitarlo explícitamente.

## 14. ¿Cómo manejarías versionado de API?
**Respuesta**: Introduciría versionado por path (`/v1`) y mantendría contratos estables por versión. La separación por capas ayuda a coexistir comportamientos.

## 15. ¿Cuál fue el criterio principal de diseño?
**Respuesta**: Priorizar confiabilidad del contrato + portabilidad de ejecución. En entrevista vale más una solución coherente, testeada y defendible que una complejidad innecesaria.

---

## 5) Red flags: qué NO decir

- “Elegí X porque estaba de moda.”
- “No hice tests porque no llegaba.”
- “Si falta un producto, da igual devolver parcial.”
- “Metí todo en handlers porque era más rápido.”
- “No pensé límites de entrada.”
- “Los logs están por si acaso, pero no los uso.”
- “No consideré el entorno de HackerRank.”
- “Clean Architecture siempre, en cualquier caso, sin trade-offs.”

**Regla práctica**: nunca vendas dogma. Mostrá criterio contextual.

---

## 6) Estrategia de demo en vivo (8-12 min)

## Min 0-1: encuadre
- Explicar objetivo funcional (`compare`) y no funcional (tests, observabilidad, portabilidad).

## Min 1-3: arquitectura
- Mostrar estructura de capas y la regla de dependencias.
- Marcar dónde vive cada decisión (dominio, use case, adapter, main).

## Min 3-6: contrato API
- Ejecutar 3 casos:
  1) Happy path `/compare?ids=1,2&fields=name,price`
  2) Validación 400 (por query inválida)
  3) 404 con `missing_ids` (caso defendible)

## Min 6-8: calidad técnica
- Mostrar tests por capa y mencionar cobertura total.
- Mostrar cómo aparece `request_id` en logs.

## Min 8-10: trade-offs y portabilidad
- Explicar por qué JSON in-memory y cómo migrar a DB o `net/http` sin romper negocio.

## Min 10-12: cierre
- Resumir 3 decisiones fuertes + próximos pasos si fuera producción.

---

## 7) Mini plan de estudio / rehearsal (20-30 min)

## Bloque A (7 min) — discurso base
- Practicar speech de 90s (2 repeticiones).
- Ajustar muletillas y eliminar adjetivos vacíos.

## Bloque B (8 min) — preguntas difíciles
- Elegir 6 preguntas de la sección 4.
- Responder en voz alta en formato: **decisión → por qué → trade-off**.

## Bloque C (8 min) — demo
- Simular demo completa con cronómetro.
- Objetivo: no superar 12 minutos y cubrir 3 requests clave.

## Bloque D (5 min) — cierre ejecutivo
- Practicar cierre de 30s con: arquitectura, confiabilidad del contrato, portabilidad.

**Checklist de salida**
- [ ] Puedo explicar 5 decisiones sin leer
- [ ] Puedo defender al menos 2 trade-offs no obvios
- [ ] Puedo responder “¿qué harías en producción?” en 60s
- [ ] Puedo justificar por qué esta solución es portable a HackerRank
