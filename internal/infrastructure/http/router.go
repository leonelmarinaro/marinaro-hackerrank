package http

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

// NewRouter arma el engine Gin con middlewares y rutas registradas.
//
// Decisión: separar router de handler permite testear handlers con un router
// minimal en tests + tener un único punto donde se ven TODAS las rutas (útil
// para auditar la superficie del API).
//
// gin.New() (no gin.Default()): queremos elegir nuestro stack de middlewares
// explícito. gin.Default() trae Logger() (texto plano sin estructura) y
// Recovery() — nos quedamos con Recovery, reemplazamos Logger por uno
// estructurado con slog + request_id.
//
// Trusted proxies fijado a nil: el binario corre detrás de un load balancer
// (en MELI sería el Mesh), no debe confiar en X-Forwarded-* de clientes
// directos. En despliegue real configuraríamos la lista del LB.
//
// Orden de rutas: las rutas estáticas (/products/categories, /products/compare)
// SE REGISTRAN ANTES que la dinámica (/products/:id). Gin matchea por orden
// y prefijo — si registramos :id primero, "categories" caería ahí como id.
func NewRouter(h *ProductHandler, logger *slog.Logger) *gin.Engine {
	r := gin.New()
	_ = r.SetTrustedProxies(nil)

	r.Use(
		gin.Recovery(),
		RequestIDMiddleware(),
		LoggingMiddleware(logger),
		SecurityHeadersMiddleware(),
	)

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
