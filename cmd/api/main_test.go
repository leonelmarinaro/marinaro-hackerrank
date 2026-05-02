package main

import "testing"

func TestPortFromEnv(t *testing.T) {
	t.Run("uses PORT when set", func(t *testing.T) {
		t.Setenv("PORT", "9090")
		if got := portFromEnv(); got != "9090" {
			t.Fatalf("expected 9090, got %q", got)
		}
	})

	t.Run("defaults to 8080 when PORT missing", func(t *testing.T) {
		t.Setenv("PORT", "")
		if got := portFromEnv(); got != "8080" {
			t.Fatalf("expected 8080, got %q", got)
		}
	})
}

func TestValidateProductsPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{name: "valid relative json", path: "testdata/products.json", wantErr: false},
		{name: "valid uppercase extension", path: "testdata/PRODUCTS.JSON", wantErr: false},
		{name: "valid path with surrounding spaces", path: "  testdata/products.json  ", wantErr: false},
		{name: "empty path", path: "", wantErr: true},
		{name: "wrong extension", path: "testdata/products.yaml", wantErr: true},
		{name: "traversal path", path: "../secrets.json", wantErr: true},
		{name: "nested traversal path", path: "testdata/../../secrets.json", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateProductsPath(tc.path)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for path %q, got nil", tc.path)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected nil error for path %q, got %v", tc.path, err)
			}
		})
	}
}
