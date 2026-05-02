// Package application contiene los casos de uso de la aplicación.
//
// Capa intermedia de Clean Architecture: depende solo de `domain`. Orquesta
// las reglas de negocio sin conocer detalles de HTTP, JSON ni almacenamiento.
// Los handlers HTTP invocan estos use cases — los use cases hablan con el
// repository port (definido en domain).
package application

import (
	"fmt"

	"github.com/lmarinaro/marinaro-hackerrank/internal/domain"
)

// CompareProductsUseCase orquesta la comparación de N productos con
// proyección selectiva de campos.
//
// Decisión: estructura con dependencia inyectada (no función global).
// Razón: facilita el wiring en cmd/api (DI manual sin frameworks) y permite
// testear contra mocks del repo. Coste: un struct por use case.
type CompareProductsUseCase struct {
	repo domain.ProductRepository
}

// NewCompareProductsUseCase es el constructor explícito.
// Constructor en lugar de field público: encapsula la dependencia y permite
// agregar validaciones de wiring en el futuro sin romper callers.
func NewCompareProductsUseCase(repo domain.ProductRepository) *CompareProductsUseCase {
	return &CompareProductsUseCase{repo: repo}
}

// CompareProductsResult es la respuesta del use case.
//
// Cada item es un map[string]any porque la lista de campos es dinámica
// (el cliente elige qué campos comparar) — un struct fijo no sirve.
type CompareProductsResult struct {
	Items  []map[string]any `json:"items"`
	Fields []string         `json:"fields"`
}

// Execute compara los productos solicitados.
//
// Validaciones (en orden, fail-fast):
//  1. ids no puede estar vacío → ErrEmptyIDs.
//  2. dedup de ids preservando orden — comparar X con X no aporta y devolver
//     duplicados confunde al cliente. Movido del handler al use case para que
//     CUALQUIER adapter (HTTP, gRPC, CLI) reciba la misma semántica.
//  3. cap de cantidad — defensa anti-DoS y guard de UX (ver MaxCompareIDs).
//  4. fields (si se pasan) deben ser todos válidos → ErrInvalidField.
//  5. todos los ids deben existir → *MissingIDsError con detalle.
//
// Decisión "todo o nada" en (5): si falta UN solo id, devolvemos error. Una
// comparación parcial es semánticamente inválida — el cliente pidió comparar
// estos N productos específicos, devolverle N-1 silenciosamente lo engaña.
func (uc *CompareProductsUseCase) Execute(ids, fields []string) (*CompareProductsResult, error) {
	if len(ids) == 0 {
		return nil, domain.ErrEmptyIDs
	}

	ids = dedupPreservingOrder(ids)

	if len(ids) > domain.MaxCompareIDs {
		return nil, fmt.Errorf("%w: got %d, max %d", domain.ErrTooManyIDs, len(ids), domain.MaxCompareIDs)
	}

	// Validamos campos contra la whitelist ANTES de tocar el repo.
	// Fail-fast: si el request es inválido, no gastamos un round-trip.
	for _, f := range fields {
		if !domain.IsAllowedField(f) {
			return nil, fmt.Errorf("%w: %q", domain.ErrInvalidField, f)
		}
	}

	products, err := uc.repo.FindByIDs(ids)
	if err != nil {
		return nil, err
	}

	items := make([]map[string]any, 0, len(products))
	for _, p := range products {
		items = append(items, p.SelectFields(fields))
	}

	return &CompareProductsResult{
		Items:  items,
		Fields: fields,
	}, nil
}

// dedupPreservingOrder elimina duplicados manteniendo el orden de aparición.
// O(n) con set auxiliar. Preservar orden importa porque el cliente esperaría
// la comparación rendereada en el orden recibido.
func dedupPreservingOrder(in []string) []string {
	if len(in) <= 1 {
		return in
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
