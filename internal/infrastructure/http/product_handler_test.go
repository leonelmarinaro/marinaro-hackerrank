package http

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lmarinaro/marinaro-hackerrank/internal/application"
	"github.com/lmarinaro/marinaro-hackerrank/internal/infrastructure/persistence"
)

// Tests de integración del HTTP layer.
//
// Estrategia: levantamos el stack completo (repo real con fixture + use cases
// + handlers + router) y ejercitamos vía httptest. Esto verifica el contrato
// público del API extremo a extremo, que es lo que el cliente realmente ve.
//
// Trade-off: más lento que mocks puros (carga JSON, serializa, etc.) pero
// detecta errores de wiring que los mocks ocultan.

const fixtureJSON = `[
  {"id":"1","name":"iPhone","category":"smartphones","price":1000,"rating":4.5,"specs":{"ram":"8GB"}},
  {"id":"2","name":"Galaxy","category":"smartphones","price":900,"rating":4.4,"specs":{"ram":"12GB"}},
  {"id":"3","name":"Book","category":"books","price":30,"rating":4.8}
]`

func setupRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dir := t.TempDir()
	path := filepath.Join(dir, "products.json")
	if err := os.WriteFile(path, []byte(fixtureJSON), 0644); err != nil {
		t.Fatalf("fixture: %v", err)
	}

	repo, err := persistence.NewJSONRepository(path)
	if err != nil {
		t.Fatalf("repo: %v", err)
	}

	h := NewProductHandler(
		application.NewCompareProductsUseCase(repo),
		application.NewListProductsUseCase(repo),
		application.NewGetProductUseCase(repo),
		application.NewListCategoriesUseCase(repo),
	)
	// Logger silencioso en tests: io.Discard descarta los JSON logs del middleware
	// para no contaminar la salida de `go test`.
	silentLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	return NewRouter(h, silentLogger)
}

func doRequest(t *testing.T, router *gin.Engine, method, url string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, url, nil)
	router.ServeHTTP(w, req)

	var body map[string]any
	if w.Body.Len() > 0 {
		_ = json.Unmarshal(w.Body.Bytes(), &body)
	}
	return w, body
}

func TestHealth(t *testing.T) {
	router := setupRouter(t)

	w, body := doRequest(t, router, "GET", "/health")

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", body["status"])
	}
}

func TestListProducts_Default(t *testing.T) {
	router := setupRouter(t)

	w, body := doRequest(t, router, "GET", "/products")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if body["total"].(float64) != 3 {
		t.Errorf("expected total=3, got %v", body["total"])
	}
}

func TestGetProduct_Found(t *testing.T) {
	router := setupRouter(t)

	w, body := doRequest(t, router, "GET", "/products/1")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if body["name"] != "iPhone" {
		t.Errorf("expected iPhone, got %v", body["name"])
	}
}

func TestGetProduct_NotFound(t *testing.T) {
	router := setupRouter(t)

	w, body := doRequest(t, router, "GET", "/products/999")

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
	if body["error"] == nil {
		t.Errorf("expected error message, got %v", body)
	}
}

func TestCategories(t *testing.T) {
	router := setupRouter(t)

	w, body := doRequest(t, router, "GET", "/products/categories")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	cats, _ := body["categories"].([]any)
	if len(cats) != 2 {
		t.Errorf("expected 2 categories, got %v", cats)
	}
}

func TestRouter_StaticCategoriesRouteTakesPrecedenceOverDynamicID(t *testing.T) {
	router := setupRouter(t)

	w, body := doRequest(t, router, "GET", "/products/categories")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on /products/categories, got %d body=%s", w.Code, w.Body.String())
	}
	if _, ok := body["categories"]; !ok {
		t.Fatalf("expected categories payload, got %v", body)
	}
}

func TestCompare_HappyPath(t *testing.T) {
	router := setupRouter(t)

	w, body := doRequest(t, router, "GET", "/products/compare?ids=1,2&fields=name,price")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	items, _ := body["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	first, _ := items[0].(map[string]any)
	if _, ok := first["name"]; !ok {
		t.Errorf("expected name in projection, got %v", first)
	}
	if _, ok := first["category"]; ok {
		t.Errorf("category should NOT be in projection, got %v", first)
	}
}

func TestCompare_MissingIDsReturns404(t *testing.T) {
	router := setupRouter(t)

	w, body := doRequest(t, router, "GET", "/products/compare?ids=1,99")

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
	missing, _ := body["missing_ids"].([]any)
	if len(missing) != 1 || missing[0] != "99" {
		t.Errorf("expected missing_ids=[99], got %v", missing)
	}
}

func TestCompare_EmptyIDsReturns400(t *testing.T) {
	router := setupRouter(t)

	w, _ := doRequest(t, router, "GET", "/products/compare")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCompare_InvalidFieldReturns400(t *testing.T) {
	router := setupRouter(t)

	w, _ := doRequest(t, router, "GET", "/products/compare?ids=1&fields=cost_price")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestList_PageOutOfRangeReturnsEmptyItemsWithTotal(t *testing.T) {
	// Comportamiento documentado: pedir una página fuera de rango NO es error.
	// Retorna items=[] + total real → el cliente sabe que ya iteró todo.
	router := setupRouter(t)

	w, body := doRequest(t, router, "GET", "/products?page=999&size=10")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if body["total"].(float64) != 3 {
		t.Errorf("expected total=3 (real catalog size), got %v", body["total"])
	}
	items, _ := body["items"].([]any)
	if len(items) != 0 {
		t.Errorf("expected empty items for out-of-range page, got %v", items)
	}
}

func TestList_NegativePageReturns400(t *testing.T) {
	// Validación strict: page negativa explícita es un error del cliente.
	router := setupRouter(t)

	w, body := doRequest(t, router, "GET", "/products?page=-1")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	if body["error"] == nil {
		t.Errorf("expected error message, got %v", body)
	}
}

func TestList_NonNumericSizeReturns400(t *testing.T) {
	router := setupRouter(t)

	w, _ := doRequest(t, router, "GET", "/products?size=abc")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCompare_DuplicateIDsAreDeduplicated(t *testing.T) {
	// ids=1,1,2 NO debe retornar 3 items con id=1 duplicado — debe retornar
	// 2 items distintos preservando el orden de primera aparición.
	router := setupRouter(t)

	w, body := doRequest(t, router, "GET", "/products/compare?ids=1,1,2&fields=name")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	items, _ := body["items"].([]any)
	if len(items) != 2 {
		t.Errorf("expected 2 items after dedup, got %d: %v", len(items), items)
	}
}

func TestCompare_FullProjectionWhenFieldsOmitted(t *testing.T) {
	router := setupRouter(t)

	w, body := doRequest(t, router, "GET", "/products/compare?ids=1")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	items, _ := body["items"].([]any)
	first, _ := items[0].(map[string]any)
	// Sin fields → todos los campos del producto deben estar
	for _, expected := range []string{"id", "name", "price", "category"} {
		if _, ok := first[expected]; !ok {
			t.Errorf("expected %q in full projection, got %v", expected, first)
		}
	}
}

func TestCompare_EmptyCSVTokensOnlyReturns400(t *testing.T) {
	router := setupRouter(t)

	w, _ := doRequest(t, router, "GET", "/products/compare?ids=,%20,%20,%20,,")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestCompare_WeirdSpacesInCSVAreTrimmed(t *testing.T) {
	router := setupRouter(t)

	w, body := doRequest(t, router, "GET", "/products/compare?ids=%201%20,%20%202%20,,&fields=%20name%20,%20price%20")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	items, _ := body["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	fields, _ := body["fields"].([]any)
	if len(fields) != 2 || fields[0] != "name" || fields[1] != "price" {
		t.Fatalf("expected fields [name price], got %v", fields)
	}
}

func TestCompare_RepeatedIDsParamUsesFirstValueOnly(t *testing.T) {
	router := setupRouter(t)

	w, body := doRequest(t, router, "GET", "/products/compare?ids=1&ids=2&fields=name")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	items, _ := body["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected 1 item (first ids param wins), got %d", len(items))
	}
}

func TestList_ExtremeAndUnparseablePaginationReturn400(t *testing.T) {
	router := setupRouter(t)

	t.Run("page overflow", func(t *testing.T) {
		w, _ := doRequest(t, router, "GET", "/products?page=999999999999999999999999")
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("size scientific notation", func(t *testing.T) {
		w, _ := doRequest(t, router, "GET", "/products?size=1e3")
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})
}
