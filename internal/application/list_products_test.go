package application

import (
	"testing"

	"github.com/lmarinaro/marinaro-hackerrank/internal/domain"
)

func TestListProducts_DefaultPagination(t *testing.T) {
	var capturedOffset, capturedLimit int
	repo := &mockRepository{
		listFn: func(offset, limit int) ([]domain.Product, int, error) {
			capturedOffset, capturedLimit = offset, limit
			return sampleProducts(), 3, nil
		},
	}
	uc := NewListProductsUseCase(repo)

	res, err := uc.Execute(0, 0) // valores inválidos → debe usar defaults

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Page != 1 || res.Size != 20 {
		t.Errorf("expected defaults page=1 size=20, got page=%d size=%d", res.Page, res.Size)
	}
	if capturedOffset != 0 || capturedLimit != 20 {
		t.Errorf("expected offset=0 limit=20, got offset=%d limit=%d", capturedOffset, capturedLimit)
	}
}

func TestListProducts_PageToOffsetConversion(t *testing.T) {
	var capturedOffset int
	repo := &mockRepository{
		listFn: func(offset, limit int) ([]domain.Product, int, error) {
			capturedOffset = offset
			return nil, 100, nil
		},
	}
	uc := NewListProductsUseCase(repo)

	_, _ = uc.Execute(3, 10) // page 3, size 10 → offset 20

	if capturedOffset != 20 {
		t.Errorf("expected offset=20, got %d", capturedOffset)
	}
}

func TestListProducts_SizeCappedAt100(t *testing.T) {
	// Cap defensivo: nadie debería poder pedir size=999999.
	var capturedLimit int
	repo := &mockRepository{
		listFn: func(offset, limit int) ([]domain.Product, int, error) {
			capturedLimit = limit
			return nil, 0, nil
		},
	}
	uc := NewListProductsUseCase(repo)

	res, _ := uc.Execute(1, 10000)

	if res.Size != 100 || capturedLimit != 100 {
		t.Errorf("expected size capped at 100, got size=%d limit=%d", res.Size, capturedLimit)
	}
}

func TestListProducts_ReturnsTotalForPagination(t *testing.T) {
	repo := &mockRepository{
		listFn: func(offset, limit int) ([]domain.Product, int, error) {
			return sampleProducts(), 42, nil
		},
	}
	uc := NewListProductsUseCase(repo)

	res, _ := uc.Execute(1, 20)

	if res.Total != 42 {
		t.Errorf("expected total=42, got %d", res.Total)
	}
}
