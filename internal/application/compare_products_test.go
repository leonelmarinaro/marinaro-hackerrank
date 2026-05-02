package application

import (
	"errors"
	"testing"

	"github.com/lmarinaro/marinaro-hackerrank/internal/domain"
)

func TestCompareProducts_HappyPath(t *testing.T) {
	repo := &mockRepository{
		findByIDsFn: func(ids []string) ([]domain.Product, error) {
			return sampleProducts()[:2], nil
		},
	}
	uc := NewCompareProductsUseCase(repo)

	res, err := uc.Execute([]string{"1", "2"}, []string{"name", "price"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(res.Items))
	}
	if _, ok := res.Items[0]["name"]; !ok {
		t.Errorf("expected name in projected fields, got %v", res.Items[0])
	}
	if _, ok := res.Items[0]["category"]; ok {
		t.Errorf("category should be excluded, got %v", res.Items[0])
	}
}

func TestCompareProducts_FullProjectionWhenFieldsEmpty(t *testing.T) {
	repo := &mockRepository{
		findByIDsFn: func(ids []string) ([]domain.Product, error) {
			return sampleProducts()[:1], nil
		},
	}
	uc := NewCompareProductsUseCase(repo)

	res, err := uc.Execute([]string{"1"}, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := res.Items[0]["category"]; !ok {
		t.Errorf("expected full projection, category missing: %v", res.Items[0])
	}
}

func TestCompareProducts_EmptyIDsRejected(t *testing.T) {
	uc := NewCompareProductsUseCase(&mockRepository{})

	_, err := uc.Execute(nil, nil)

	if !errors.Is(err, domain.ErrEmptyIDs) {
		t.Errorf("expected ErrEmptyIDs, got %v", err)
	}
}

func TestCompareProducts_InvalidFieldRejected(t *testing.T) {
	uc := NewCompareProductsUseCase(&mockRepository{})

	_, err := uc.Execute([]string{"1"}, []string{"name", "cost_price"})

	if !errors.Is(err, domain.ErrInvalidField) {
		t.Errorf("expected ErrInvalidField, got %v", err)
	}
}

func TestCompareProducts_RepoErrorPropagated(t *testing.T) {
	missing := &domain.MissingIDsError{Missing: []string{"99"}}
	repo := &mockRepository{
		findByIDsFn: func(ids []string) ([]domain.Product, error) {
			return nil, missing
		},
	}
	uc := NewCompareProductsUseCase(repo)

	_, err := uc.Execute([]string{"1", "99"}, nil)

	// errors.Is debe alcanzar ErrProductNotFound vía Unwrap
	if !errors.Is(err, domain.ErrProductNotFound) {
		t.Errorf("expected error to unwrap to ErrProductNotFound, got %v", err)
	}
	var miss *domain.MissingIDsError
	if !errors.As(err, &miss) {
		t.Errorf("expected MissingIDsError, got %T", err)
	}
}

func TestCompareProducts_DoesNotCallRepoWhenValidationFails(t *testing.T) {
	// Garantiza fail-fast: si la validación de fields falla, NO llamamos al repo.
	called := false
	repo := &mockRepository{
		findByIDsFn: func(ids []string) ([]domain.Product, error) {
			called = true
			return nil, nil
		},
	}
	uc := NewCompareProductsUseCase(repo)

	_, _ = uc.Execute([]string{"1"}, []string{"invalid_field"})

	if called {
		t.Errorf("repo should not be called when fields validation fails")
	}
}
