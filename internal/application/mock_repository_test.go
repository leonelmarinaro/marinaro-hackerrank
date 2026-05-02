package application

import (
	"github.com/lmarinaro/marinaro-hackerrank/internal/domain"
)

// mockRepository es un test double del puerto ProductRepository.
//
// Decisión: mock manual en lugar de gomock/testify-mock.
// Razón: para 4 use cases con interfaces simples, una librería de mocking
// es overkill. Un struct con campos func es más explícito, debuggeable y
// no agrega dependencias. Trade-off: verbose si la interfaz crece — punto
// en el que vale la pena migrar a gomock.
type mockRepository struct {
	findByIDFn   func(id string) (*domain.Product, error)
	findByIDsFn  func(ids []string) ([]domain.Product, error)
	listFn       func(offset, limit int) ([]domain.Product, int, error)
	categoriesFn func() ([]string, error)
}

// Cada método nil-checkea el func antes de invocarlo. Razón: si un test
// instancia el mock sin setear el func correspondiente y el use case por error
// llega a llamarlo, queremos un mensaje claro — no un nil-pointer panic
// críptico que esconde el bug real.
func (m *mockRepository) FindByID(id string) (*domain.Product, error) {
	if m.findByIDFn == nil {
		panic("mockRepository.FindByID called but findByIDFn is nil — check test setup")
	}
	return m.findByIDFn(id)
}

func (m *mockRepository) FindByIDs(ids []string) ([]domain.Product, error) {
	if m.findByIDsFn == nil {
		panic("mockRepository.FindByIDs called but findByIDsFn is nil — check test setup")
	}
	return m.findByIDsFn(ids)
}

func (m *mockRepository) List(offset, limit int) ([]domain.Product, int, error) {
	if m.listFn == nil {
		panic("mockRepository.List called but listFn is nil — check test setup")
	}
	return m.listFn(offset, limit)
}

func (m *mockRepository) Categories() ([]string, error) {
	if m.categoriesFn == nil {
		panic("mockRepository.Categories called but categoriesFn is nil — check test setup")
	}
	return m.categoriesFn()
}

// sampleProducts retorna fixtures reusables entre tests.
func sampleProducts() []domain.Product {
	return []domain.Product{
		{
			ID: "1", Name: "iPhone 15", Category: "smartphones",
			Price: 1299.99, Rating: 4.7,
			Specs: map[string]any{"ram": "8GB"},
		},
		{
			ID: "2", Name: "Galaxy S24", Category: "smartphones",
			Price: 1199.99, Rating: 4.6,
			Specs: map[string]any{"ram": "12GB"},
		},
		{
			ID: "3", Name: "Clean Code", Category: "books",
			Price: 35.00, Rating: 4.8,
		},
	}
}
