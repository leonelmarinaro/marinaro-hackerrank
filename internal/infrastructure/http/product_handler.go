package http

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lmarinaro/marinaro-hackerrank/internal/application"
	"github.com/lmarinaro/marinaro-hackerrank/internal/domain"
)

// ProductHandler agrupa los handlers que comparten dependencias.
//
// Decisión: handler como método del struct (no funciones sueltas con closures).
// Razón: tests más limpios — instanciás un handler con mocks de los use cases
// y testeás cada método. Closures hacen tests más enredados.
type ProductHandler struct {
	compareUC    *application.CompareProductsUseCase
	listUC       *application.ListProductsUseCase
	getUC        *application.GetProductUseCase
	categoriesUC *application.ListCategoriesUseCase
}

func NewProductHandler(
	compareUC *application.CompareProductsUseCase,
	listUC *application.ListProductsUseCase,
	getUC *application.GetProductUseCase,
	categoriesUC *application.ListCategoriesUseCase,
) *ProductHandler {
	return &ProductHandler{
		compareUC:    compareUC,
		listUC:       listUC,
		getUC:        getUC,
		categoriesUC: categoriesUC,
	}
}

// Health: GET /health → 200 OK.
// Endpoint trivial pero estándar — ops, k8s liveness, monitores.
func (h *ProductHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, healthResponse{Status: "ok"})
}

// Compare: GET /products/compare?ids=1,2,3&fields=name,price
//
// Parsea CSV de los query params, invoca el use case, mapea errores.
// CSV en query string es la convención RESTful (vs. múltiples ?ids=1&ids=2 que
// también es válido pero menos compacto y peor con caches HTTP).
//
// La deduplicación, el cap de cantidad y la validación de fields ocurren en
// el use case — el handler es puro transporte, sin lógica de negocio.
func (h *ProductHandler) Compare(c *gin.Context) {
	ids := splitCSV(c.Query("ids"))
	fields := splitCSV(c.Query("fields"))

	res, err := h.compareUC.Execute(ids, fields)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// List: GET /products?page=1&size=20
//
// Validación strict en el handler: si el cliente PASA explícitamente page/size
// con valor inválido (negativo, no parseable), retornamos 400. Si NO los pasa,
// aplicamos defaults silenciosamente. Esto evita el "silent clamp" — un cliente
// que mande page=-1 esperando algo va a recibir un 400 en vez de la página 1
// inesperadamente.
//
// El use case mantiene clamp como salvaguarda — defensa en profundidad.
func (h *ProductHandler) List(c *gin.Context) {
	page, err := parsePositiveInt(c, "page", 1)
	if err != nil {
		writeError(c, err)
		return
	}
	size, err := parsePositiveInt(c, "size", 20)
	if err != nil {
		writeError(c, err)
		return
	}

	res, err := h.listUC.Execute(page, size)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// Get: GET /products/:id
func (h *ProductHandler) Get(c *gin.Context) {
	id := c.Param("id")

	p, err := h.getUC.Execute(id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

// Categories: GET /products/categories
func (h *ProductHandler) Categories(c *gin.Context) {
	cats, err := h.categoriesUC.Execute()
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"categories": cats})
}

// splitCSV parsea un CSV de query param, ignorando espacios y vacíos.
//
// Helper local porque es trivial y específico de la layer HTTP. Si crece,
// podríamos extraerlo a un paquete utils — por ahora YAGNI.
func splitCSV(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// parsePositiveInt lee un query param numérico positivo.
// Si está ausente, retorna defaultValue. Si está presente pero no parseable
// o no es positivo, retorna ErrInvalidPagination wrapeado con el campo.
func parsePositiveInt(c *gin.Context, name string, defaultValue int) (int, error) {
	raw := c.Query(name)
	if raw == "" {
		return defaultValue, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 1 {
		return 0, fmt.Errorf("%w: %s=%q", domain.ErrInvalidPagination, name, raw)
	}
	return v, nil
}
