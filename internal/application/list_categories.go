package application

import "github.com/lmarinaro/marinaro-hackerrank/internal/domain"

// ListCategoriesUseCase: categorías distintas del catálogo.
//
// Útil para que la UI arme filtros dinámicos sin hardcodear la lista —
// si el catálogo agrega "electrodomésticos", el endpoint lo refleja
// automáticamente.
type ListCategoriesUseCase struct {
	repo domain.ProductRepository
}

func NewListCategoriesUseCase(repo domain.ProductRepository) *ListCategoriesUseCase {
	return &ListCategoriesUseCase{repo: repo}
}

func (uc *ListCategoriesUseCase) Execute() ([]string, error) {
	return uc.repo.Categories()
}
