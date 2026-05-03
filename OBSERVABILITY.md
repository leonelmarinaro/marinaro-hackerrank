# Observability mínima para demo técnica

Este proyecto ya tiene una base simple y efectiva para observar comportamiento en runtime sin sobreingeniería.

## 1) Logs estructurados

- Formato: JSON (`log/slog`).
- Evento principal: `http_request` por request.
- Campos clave:
  - `request_id`
  - `method`
  - `path`
  - `client_ip`
  - `status`
  - `duration`

Por qué importa: te permite correlacionar requests, detectar latencia y segmentar errores por endpoint rápido.

## 2) request_id end-to-end

- Header soportado: `X-Request-Id`.
- Si el cliente lo manda, se respeta.
- Si no viene, el middleware genera uno y lo devuelve en la respuesta.

Checklist de demo:
- Hacer un request con `X-Request-Id: demo-123` y verificar que:
  1. vuelve en response header
  2. aparece igual en logs

## 3) Manejo de errores 500

- Los errores de negocio se mapean a 4xx.
- Para errores inesperados, el handler devuelve:
  - status: `500`
  - body: `{"error":"internal server error"}`
- Nunca se filtra `err.Error()` en respuestas 500.

Por qué importa: evita exponer detalles internos y reduce superficie de ataque.

## 4) Headers de seguridad básicos

El middleware agrega:
- `X-Content-Type-Options: nosniff`
- `Referrer-Policy: no-referrer`

No es una suite completa de hardening, pero cubre lo mínimo defendible para challenge.

## 5) Timeouts del servidor HTTP

Definidos en `cmd/api/main.go`:
- `ReadHeaderTimeout`: 5s
- `ReadTimeout`: 10s
- `WriteTimeout`: 15s
- `IdleTimeout`: 60s
- `ShutdownTimeout`: 10s

Por qué importa: previene conexiones colgadas y limita consumo de recursos.

## 6) Qué mirar en una demo (5 minutos)

1. **Health**
   - `GET /health` responde 200.
2. **Correlación**
   - request con `X-Request-Id` y confirmación en logs/headers.
3. **Error controlado**
   - `/products/compare?ids=1,99` → 404 con `missing_ids`.
4. **Error inesperado (explicado)**
   - Mostrar que el contrato para 500 es genérico y no filtra internals.
5. **Latencia básica**
   - Revisar campo `duration` en 2-3 endpoints.

Con esto ya tenés observabilidad suficiente para una instancia de entrevista sin meter Prometheus/OTel/Grafana.
