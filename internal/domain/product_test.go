package domain

import (
	"errors"
	"testing"
)

// Tests del entity Product.
//
// Filosofía TDD aplicada: estos tests fueron escritos antes que la implementación.
// Diseñados sin dependencias externas — solo struct + métodos puros. Verifican
// la lógica de negocio pura: validación de campos, proyección selectiva.

func TestIsAllowedField(t *testing.T) {
	cases := []struct {
		field string
		want  bool
	}{
		{"name", true},
		{"price", true},
		{"specs", true},
		{"image_url", true},
		// Campos hipotéticos internos que no deberían filtrarse vía API:
		{"cost_price", false},
		{"internal_sku", false},
		// Edge cases:
		{"", false},
		{"NAME", false}, // case-sensitive intencional para evitar ambigüedad
	}
	for _, c := range cases {
		if got := IsAllowedField(c.field); got != c.want {
			t.Errorf("IsAllowedField(%q) = %v, want %v", c.field, got, c.want)
		}
	}
}

func TestSelectFields_FullProductWhenFieldsEmpty(t *testing.T) {
	p := newSampleProduct()

	out := p.SelectFields(nil)

	if out["id"] != "p1" {
		t.Errorf("expected full product with id=p1, got %v", out["id"])
	}
	if out["name"] != "iPhone 15" {
		t.Errorf("expected name=iPhone 15, got %v", out["name"])
	}
	// Specs debe estar presente porque tiene valor
	if _, ok := out["specs"]; !ok {
		t.Errorf("specs should be included when non-empty")
	}
}

func TestSelectFields_SubsetReturnsOnlyRequested(t *testing.T) {
	p := newSampleProduct()

	out := p.SelectFields([]string{"name", "price"})

	if len(out) != 2 {
		t.Errorf("expected 2 keys, got %d: %v", len(out), out)
	}
	if out["name"] != "iPhone 15" {
		t.Errorf("expected name, got %v", out["name"])
	}
	if _, ok := out["description"]; ok {
		t.Errorf("description should be excluded, got %v", out)
	}
	if _, ok := out["specs"]; ok {
		t.Errorf("specs should be excluded, got %v", out)
	}
}

func TestSelectFields_UnknownFieldSilentlyIgnored(t *testing.T) {
	p := newSampleProduct()

	out := p.SelectFields([]string{"name", "totally_invented"})

	if _, ok := out["totally_invented"]; ok {
		t.Errorf("unknown field should be silently ignored at entity level")
	}
	if out["name"] != "iPhone 15" {
		t.Errorf("known field should still be included")
	}
}

func TestSelectFields_OptionalZeroFieldsOmitted(t *testing.T) {
	// Producto minimal — solo campos requeridos. Los opcionales en cero
	// no deben aparecer en el output (consistente con omitempty del JSON).
	p := Product{ID: "x", Name: "minimal"}

	out := p.SelectFields(nil)

	for _, f := range []string{"size", "weight", "color", "specs"} {
		if _, ok := out[f]; ok {
			t.Errorf("optional zero-value field %q should be omitted, got %v", f, out)
		}
	}
}

func TestMissingIDsError_UnwrapToProductNotFound(t *testing.T) {
	err := &MissingIDsError{Missing: []string{"99", "100"}}

	// errors.Is debe matchear ErrProductNotFound vía Unwrap —
	// permite a los handlers tratar ambos errores como 404.
	if !errors.Is(err, ErrProductNotFound) {
		t.Errorf("MissingIDsError should unwrap to ErrProductNotFound")
	}

	if err.Error() == "" {
		t.Errorf("error message should not be empty")
	}
}

// newSampleProduct retorna un producto de prueba para reutilizar en tests.
// Helper privado del paquete de test.
func newSampleProduct() Product {
	return Product{
		ID:          "p1",
		Name:        "iPhone 15",
		Description: "Latest iPhone",
		ImageURL:    "https://example.com/iphone15.jpg",
		Price:       1299.99,
		Rating:      4.7,
		Category:    "smartphones",
		Color:       "black",
		Weight:      0.171,
		Specs: map[string]any{
			"battery": "3349mAh",
			"ram":     "8GB",
			"storage": "256GB",
			"os":      "iOS 17",
		},
	}
}
