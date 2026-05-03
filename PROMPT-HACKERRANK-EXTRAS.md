# Prompt reutilizable — paquete de 5 extras (HackerRank/Sonnet)

```text
Implementá EXACTAMENTE estos 5 extras en este repo Go, sin sobreingeniería y sin cambiar contratos públicos del API:

1) GitHub Actions mínimo
   - Crear `.github/workflows/ci.yml`
   - Ejecutar solo:
     a) gofmt check (fallar si hay archivos no formateados)
     b) go vet ./...
     c) go test ./...

2) pre-commit liviano
   - Crear hook versionado en `.githooks/pre-commit` con:
     a) gofmt -w .
     b) go vet ./...
     c) go test ./...
   - Dejarlo ejecutable
   - NO agregar frameworks de hooks ni tooling pesado

3) OBSERVABILITY.md
   - Documentar: logs estructurados actuales, request_id, manejo de 500, headers de seguridad, timeouts HTTP y qué mostrar en una demo
   - Enfoque práctico y corto

4) PRODUCTION-NEXT-STEPS.md
   - Lista priorizada de próximos pasos realistas para producción
   - Para cada paso: qué + por qué
   - Sin humo (nada enterprise de adorno)

5) Dockerfile simple
   - Preferencia: multistage
   - Binario Go en etapa builder, runtime liviano
   - Sin docker-compose ni Kubernetes ni scripts raros

Restricciones:
- NO agregar devcontainer
- NO montar CI/CD pesado
- NO agregar Prometheus/Grafana/OTel reales
- NO agregar deploy automation
- NO hacer build salvo estrictamente necesario
- Sí correr checks livianos

Salida esperada:
- lista de archivos creados/modificados
- comandos de validación ejecutados
- riesgos/trade-offs concretos
```
