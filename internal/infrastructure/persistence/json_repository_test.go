package persistence

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/lmarinaro/marinaro-hackerrank/internal/domain"
)

// Tests de integración del adapter JSON.
//
// A diferencia de los tests del use case (que mockean el repo), acá
// ejercitamos el adapter REAL con un archivo de prueba — verificamos
// el contrato completo del puerto.

const sampleJSON = `[
  {"id":"1","name":"A","category":"x","price":10},
  {"id":"2","name":"B","category":"y","price":20},
  {"id":"3","name":"C","category":"x","price":30}
]`

func writeTempJSON(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "products.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	return path
}

func TestNewJSONRepository_LoadsValidFile(t *testing.T) {
	path := writeTempJSON(t, sampleJSON)

	repo, err := NewJSONRepository(path)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, total, _ := repo.List(0, 100)
	if total != 3 {
		t.Errorf("expected 3 products loaded, got %d", total)
	}
}

func TestNewJSONRepository_FailsOnMissingFile(t *testing.T) {
	_, err := NewJSONRepository("/nonexistent/path/products.json")

	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestNewJSONRepository_FailsOnInvalidJSON(t *testing.T) {
	path := writeTempJSON(t, `{not valid json`)

	_, err := NewJSONRepository(path)

	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestNewJSONRepository_FailsOnDuplicateIDs(t *testing.T) {
	dup := `[{"id":"1","name":"A","category":"x"},{"id":"1","name":"B","category":"y"}]`
	path := writeTempJSON(t, dup)

	_, err := NewJSONRepository(path)

	if err == nil {
		t.Error("expected error for duplicate IDs")
	}
}

func TestNewJSONRepository_FailsOnEmptyID(t *testing.T) {
	// Un id vacío rompe lookups silenciosamente — el segundo registro con
	// id="" sobreescribiría al primero en el índice byID. Mejor fail-fast.
	empty := `[{"id":"","name":"A","category":"x"}]`
	path := writeTempJSON(t, empty)

	_, err := NewJSONRepository(path)

	if err == nil {
		t.Error("expected error for empty id")
	}
}

func TestFindByID_Found(t *testing.T) {
	repo := setupRepo(t)

	got, err := repo.FindByID("2")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "B" {
		t.Errorf("expected name=B, got %q", got.Name)
	}
}

func TestFindByID_NotFound(t *testing.T) {
	repo := setupRepo(t)

	_, err := repo.FindByID("999")

	if !errors.Is(err, domain.ErrProductNotFound) {
		t.Errorf("expected ErrProductNotFound, got %v", err)
	}
}

func TestFindByIDs_AllFound_PreservesOrder(t *testing.T) {
	repo := setupRepo(t)

	got, err := repo.FindByIDs([]string{"3", "1"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 products, got %d", len(got))
	}
	// Orden del input preservado: [3, 1]
	if got[0].ID != "3" || got[1].ID != "1" {
		t.Errorf("expected order [3,1], got [%s,%s]", got[0].ID, got[1].ID)
	}
}

func TestFindByIDs_PartialMissingReturnsError(t *testing.T) {
	repo := setupRepo(t)

	_, err := repo.FindByIDs([]string{"1", "999", "888"})

	var miss *domain.MissingIDsError
	if !errors.As(err, &miss) {
		t.Fatalf("expected MissingIDsError, got %T: %v", err, err)
	}
	if len(miss.Missing) != 2 {
		t.Errorf("expected 2 missing, got %d: %v", len(miss.Missing), miss.Missing)
	}
}

func TestList_Pagination(t *testing.T) {
	repo := setupRepo(t)

	page, total, err := repo.List(1, 1)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 3 {
		t.Errorf("expected total=3, got %d", total)
	}
	if len(page) != 1 || page[0].ID != "2" {
		t.Errorf("expected single product id=2, got %v", page)
	}
}

func TestList_OffsetBeyondTotalReturnsEmpty(t *testing.T) {
	repo := setupRepo(t)

	page, total, err := repo.List(100, 10)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page) != 0 || total != 3 {
		t.Errorf("expected empty page with total=3, got %d items total=%d", len(page), total)
	}
}

func TestCategories_DistinctAndSorted(t *testing.T) {
	repo := setupRepo(t)

	cats, err := repo.Categories()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// sample: x, y, x → distinct = [x, y] ordenado alfabéticamente
	if len(cats) != 2 || cats[0] != "x" || cats[1] != "y" {
		t.Errorf("expected [x, y], got %v", cats)
	}
}

func setupRepo(t *testing.T) *JSONRepository {
	t.Helper()
	path := writeTempJSON(t, sampleJSON)
	repo, err := NewJSONRepository(path)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	return repo
}
