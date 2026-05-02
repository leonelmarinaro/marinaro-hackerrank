// Package main es el composition root del API.
//
// Responsabilidad única: instanciar y conectar los componentes de las capas
// internas. NO contiene lógica de negocio. Si tuvieras que cambiar todo
// (framework HTTP, persistencia, etc.) este es uno de los pocos archivos que
// se tocan junto con los adapters concretos.
//
// Patrón: Manual Dependency Injection. Sin framework de DI (Wire, Fx).
// Razón: para 4 use cases + 1 repo + 1 router, hacerlo a mano es 10 líneas
// y se lee mejor que cualquier mágica de generación de código.
package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/lmarinaro/marinaro-hackerrank/internal/application"
	httpadapter "github.com/lmarinaro/marinaro-hackerrank/internal/infrastructure/http"
	"github.com/lmarinaro/marinaro-hackerrank/internal/infrastructure/persistence"
)

func main() {
	// Path del catálogo: configurable por env para portabilidad (HackerRank,
	// containers, tests). Default razonable apunta al fixture del repo.
	dataPath := os.Getenv("PRODUCTS_FILE")
	if dataPath == "" {
		dataPath = "testdata/products.json"
	}

	// Validación defensiva del path. Aunque el catálogo se carga al boot
	// (no por request, no hay riesgo directo de path traversal vía API),
	// rechazamos paths sospechosos para evitar que una mala configuración
	// (env var inyectada, typo) lea archivos no deseados. Reglas:
	//   - extensión .json obligatoria — leer cualquier otro archivo es accidente.
	//   - sin componentes ".." después de Clean — bloquea escape básico.
	if err := validateProductsPath(dataPath); err != nil {
		log.Fatalf("startup: invalid PRODUCTS_FILE %q: %v", dataPath, err)
	}

	// Carga del catálogo. Si falla, abortamos: un servicio sin datos no
	// debería arrancar pretendiendo estar sano (fail-fast).
	repo, err := persistence.NewJSONRepository(dataPath)
	if err != nil {
		log.Fatalf("startup: failed loading products from %q: %v", dataPath, err)
	}
	log.Printf("startup: loaded products from %q", dataPath)

	// Use cases — todos toman el mismo repo. La inyección por constructor
	// hace explícita la dependencia.
	compareUC := application.NewCompareProductsUseCase(repo)
	listUC := application.NewListProductsUseCase(repo)
	getUC := application.NewGetProductUseCase(repo)
	categoriesUC := application.NewListCategoriesUseCase(repo)

	// Handler agrupa los use cases que comparten un mismo recurso (productos).
	// Si más adelante hay otros recursos (orders, users), tendrían sus propios
	// handlers — manteniendo bajo acoplamiento.
	handler := httpadapter.NewProductHandler(compareUC, listUC, getUC, categoriesUC)
	router := httpadapter.NewRouter(handler)

	// Puerto configurable por env (convención 12-factor). Default :8080.
	addr := ":" + portFromEnv()
	log.Printf("startup: listening on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// portFromEnv devuelve el puerto desde la env var PORT o 8080 por default.
// Helper local para mantener main() legible.
func portFromEnv() string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return "8080"
}

// validateProductsPath sanitiza el path del catálogo.
//
// No es defensa contra atacante avanzado — el binario corre con permisos del
// usuario que lo ejecuta — pero filtra el 99% de errores accidentales (typos,
// configs malas, paths que apuntan a /etc/*).
func validateProductsPath(path string) error {
	if path == "" {
		return errEmptyPath
	}
	if !strings.HasSuffix(strings.ToLower(path), ".json") {
		return errBadExtension
	}
	// filepath.Clean colapsa ".." — si después de limpiar el path empieza con
	// ".." es porque escapaba del directorio de trabajo.
	clean := filepath.Clean(path)
	if strings.HasPrefix(clean, "..") {
		return errPathTraversal
	}
	return nil
}

var (
	errEmptyPath     = stringError("path is empty")
	errBadExtension  = stringError("path must end in .json")
	errPathTraversal = stringError("path contains traversal segments")
)

// stringError es un error inmutable basado en string. Usamos esto en lugar
// de errors.New para mantener los mensajes como const-like (permite reusar
// vía == en tests sin asignaciones globales).
type stringError string

func (e stringError) Error() string { return string(e) }
