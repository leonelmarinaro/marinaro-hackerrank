package application

import (
	"errors"
	"testing"

	"github.com/lmarinaro/marinaro-hackerrank/internal/domain"
)

func TestGetProduct_HappyPath(t *testing.T) {
	repo := &mockRepository{
		findByIDFn: func(id string) (*domain.Product, error) {
			p := sampleProducts()[0]
			return &p, nil
		},
	}
	uc := NewGetProductUseCase(repo)

	got, err := uc.Execute("1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "1" {
		t.Errorf("expected id=1, got %v", got.ID)
	}
}

func TestGetProduct_EmptyIDRejected(t *testing.T) {
	uc := NewGetProductUseCase(&mockRepository{})

	_, err := uc.Execute("")

	if !errors.Is(err, domain.ErrEmptyIDs) {
		t.Errorf("expected ErrEmptyIDs, got %v", err)
	}
}

func TestGetProduct_NotFoundPropagated(t *testing.T) {
	repo := &mockRepository{
		findByIDFn: func(id string) (*domain.Product, error) {
			return nil, domain.ErrProductNotFound
		},
	}
	uc := NewGetProductUseCase(repo)

	_, err := uc.Execute("99")

	if !errors.Is(err, domain.ErrProductNotFound) {
		t.Errorf("expected ErrProductNotFound, got %v", err)
	}
}
