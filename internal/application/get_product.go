package application

import "github.com/lmarinaro/marinaro-hackerrank/internal/domain"

// GetProductUseCase: detalle de un producto por ID.
//
// Use case "trivial" intencional: aunque sea un passthrough al repo, mantener
// la simetría arquitectónica vale la pena — todos los handlers HTTP hablan
// con use cases (nunca con el repo directo). Si mañana hay que agregar
// caching, métricas o autorización, el punto de extensión está acá.
type GetProductUseCase struct {
	repo domain.ProductRepository
}

func NewGetProductUseCase(repo domain.ProductRepository) *GetProductUseCase {
	return &GetProductUseCase{repo: repo}
}

func (uc *GetProductUseCase) Execute(id string) (*domain.Product, error) {
	if id == "" {
		return nil, domain.ErrEmptyIDs
	}
	return uc.repo.FindByID(id)
}
