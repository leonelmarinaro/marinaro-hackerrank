package http

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// requestIDHeader es el header donde leemos/escribimos el ID de correlación.
// Convención de facto en la industria (Heroku, AWS, Datadog, etc.).
const requestIDHeader = "X-Request-Id"

// RequestIDMiddleware adjunta un ID único a cada request.
//
// Honra el header entrante (útil cuando hay un edge proxy que ya generó uno
// para correlacionar logs entre servicios) y lo genera si no viene. El ID se
// expone también en el response header para que el cliente pueda referenciarlo
// al reportar bugs.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader(requestIDHeader)
		if rid == "" {
			rid = newRequestID()
		}
		c.Set("request_id", rid)
		c.Writer.Header().Set(requestIDHeader, rid)
		c.Next()
	}
}

// LoggingMiddleware loguea cada request en formato JSON estructurado.
//
// Por qué no usamos gin.Logger() default: emite texto libre, sin request_id,
// y mete query string en plano (riesgo de leak si en el futuro hubiera tokens).
// slog (stdlib desde Go 1.21) nos da JSON estructurado sin dependencias.
func LoggingMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		// Loguear DESPUÉS de procesar — necesitamos status y duración.
		logger.Info("http_request",
			slog.String("request_id", c.GetString("request_id")),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("duration", time.Since(start)),
			slog.String("client_ip", c.ClientIP()),
		)
	}
}

// SecurityHeadersMiddleware agrega headers HTTP de hardening básico.
//
// Para una API JSON pura no aplica casi nada de lo "web" (XSS, clickjacking),
// pero seteamos los que sí tienen sentido como defensa en profundidad y como
// señal explícita ante un auditor de que pensamos en el tema.
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// nosniff: bloquea content-type sniffing del browser. Cero coste.
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		// Referrer-Policy: no enviar el path de la API si alguien lo abre vía link.
		c.Writer.Header().Set("Referrer-Policy", "no-referrer")
		c.Next()
	}
}

// newRequestID genera un ID corto pero suficientemente único para correlación.
// 8 bytes hex (16 chars) es ~10^19 combinaciones — sobra para distinguir
// requests en logs. Si necesitáramos cross-service tracing real iríamos a
// W3C Trace Context o OpenTelemetry, pero eso es over-engineering acá.
func newRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fallback nanoseg-based si crypto/rand falla (extremadamente raro).
		// No bloqueamos la request por algo tan accesorio como un ID.
		return time.Now().Format("150405.000000000")
	}
	return hex.EncodeToString(b)
}
