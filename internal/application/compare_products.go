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
// Validaciones:
//  1. ids no puede estar vacío → ErrEmptyIDs
//  2. fields (si se pasan) deben ser todos válidos → ErrInvalidField
//  3. todos los ids deben existir → *MissingIDsError con detalle
//
// Decisión "todo o nada": si falta UN solo id, devolvemos error. Una
// comparación parcial es semánticamente inválida — el cliente pidió comparar
// estos N productos específicos, devolverle N-1 silenciosamente lo engaña.
func (uc *CompareProductsUseCase) Execute(ids, fields []string) (*CompareProductsResult, error) {
	if len(ids) == 0 {
		return nil, domain.ErrEmptyIDs
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
