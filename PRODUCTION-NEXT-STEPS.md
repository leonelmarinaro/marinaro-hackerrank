# Próximos pasos realistas para producción

Este documento prioriza mejoras de impacto real sin inflar el scope.

## 1) Configuración y secretos

**Qué**
- Centralizar config en una capa explícita (env vars validadas al boot).
- Separar secretos de configuración no sensible.

**Por qué**
- Evita defaults peligrosos y fallos silenciosos en deploy.

## 2) Persistencia real (reemplazar JSON in-memory)

**Qué**
- Migrar `ProductRepository` a Postgres (o storage equivalente).
- Mantener el port de dominio para no tocar use cases.

**Por qué**
- JSON local no escala, no tiene concurrencia real ni trazabilidad de cambios.

## 3) Autenticación y autorización

**Qué**
- Definir qué endpoints son públicos y cuáles requieren auth.
- Agregar auth por token/JWT en middleware.

**Por qué**
- En producción, exponer data sin control es un riesgo básico.

## 4) Rate limiting y protección de abuso

**Qué**
- Limitar requests por IP/API key.
- Mantener caps existentes (`size <= 100`, `MaxCompareIDs=50`).

**Por qué**
- Complementa timeouts y reduce riesgo de DoS por abuso de endpoints caros.

## 5) Observabilidad operativa incremental

**Qué**
- Métricas HTTP básicas (RPS, latencia p95/p99, error rate).
- Alertas por 5xx y latencia degradada.

**Por qué**
- Logs solos no alcanzan para alerting proactivo.

## 6) Calidad de entrega

**Qué**
- Mantener CI mínima (fmt/vet/test) y sumar cobertura mínima por paquete.
- Definir política de PR (review obligatorio + checks verdes).

**Por qué**
- Baja regresiones y hace predecible la evolución del código.

## 7) Contrato de API y versionado

**Qué**
- Publicar OpenAPI y versionar endpoints (`/v1`).

**Por qué**
- Evita romper consumidores al evolucionar payloads.

## 8) Seguridad de dependencias y supply chain

**Qué**
- Escaneo de dependencias en CI (ej. `govulncheck`).
- Pin de versiones y política de actualización.

**Por qué**
- Reduce exposición a CVEs y sorpresas por upgrades automáticos.

---

## Orden recomendado (prioridad)

1. Config + secretos
2. Persistencia real
3. Auth
4. Rate limiting
5. Métricas + alertas
6. OpenAPI/versionado

Ese orden maximiza reducción de riesgo con costo razonable para un backend chico.
