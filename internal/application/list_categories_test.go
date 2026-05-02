package application

import "testing"

func TestListCategories_PassthroughToRepo(t *testing.T) {
	repo := &mockRepository{
		categoriesFn: func() ([]string, error) {
			return []string{"books", "smartphones"}, nil
		},
	}
	uc := NewListCategoriesUseCase(repo)

	got, err := uc.Execute()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != "books" {
		t.Errorf("expected [books, smartphones], got %v", got)
	}
}
