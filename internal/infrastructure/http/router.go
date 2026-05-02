package http

import "github.com/gin-gonic/gin"

// NewRouter arma el engine Gin con las rutas registradas.
//
// Decisión: separar router de handler permite testear handlers con un router
// minimal en tests + tener un único punto donde se ven TODAS las rutas (útil
// para auditar la superficie del API).
//
// Orden de rutas: las rutas estáticas (/products/categories, /products/compare)
// SE REGISTRAN ANTES que la dinámica (/products/:id). Gin matchea por orden
// y prefijo — si registramos :id primero, "categories" caería ahí como id.
func NewRouter(h *ProductHandler) *gin.Engine {
	r := gin.Default()

	r.GET("/health", h.Health)

	products := r.Group("/products")
	{
		products.GET("", h.List)
		products.GET("/compare", h.Compare)       // ANTES que /:id
		products.GET("/categories", h.Categories) // ANTES que /:id
		products.GET("/:id", h.Get)
	}

	return r
}
