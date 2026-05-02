package application

import "github.com/lmarinaro/marinaro-hackerrank/internal/domain"

// ListProductsUseCase: catálogo paginado.
//
// Por qué paginación obligatoria: aunque el dataset de prueba sea chico,
// exponer un endpoint que retorna "todo el catálogo" es un anti-patrón
// (memoria, latencia, DoS accidental). Mejor sembrar la disciplina desde el
// día 1 — incluso si el límite default cubre todo el seed actual.
type ListProductsUseCase struct {
	repo domain.ProductRepository
}

func NewListProductsUseCase(repo domain.ProductRepository) *ListProductsUseCase {
	return &ListProductsUseCase{repo: repo}
}

// ListProductsResult contiene la página + metadata para que el cliente
// pueda iterar (total, page, size).
type ListProductsResult struct {
	Items []domain.Product `json:"items"`
	Total int              `json:"total"`
	Page  int              `json:"page"`
	Size  int              `json:"size"`
}

// Execute retorna una página de productos.
//
// Convención:
//   - page: 1-based (más natural para humanos y URLs).
//   - size: cantidad por página, default 20, máximo 100 (cap defensivo).
//
// Convertimos page→offset internamente para hablar con el repo en su lenguaje.
func (uc *ListProductsUseCase) Execute(page, size int) (*ListProductsResult, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20 // default razonable
	}
	if size > 100 {
		size = 100 // cap: previene DoS por size=1000000
	}

	offset := (page - 1) * size

	products, total, err := uc.repo.List(offset, size)
	if err != nil {
		return nil, err
	}

	return &ListProductsResult{
		Items: products,
		Total: total,
		Page:  page,
		Size:  size,
	}, nil
}
