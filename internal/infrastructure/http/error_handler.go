package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lmarinaro/marinaro-hackerrank/internal/domain"
)

// writeError centraliza el mapeo error de dominio → HTTP status.
//
// Por qué centralizar: si cada handler hiciera el mapeo a mano, el día que
// agreguemos un nuevo error tipado tendríamos que recordar tocar todos los
// handlers. Acá el mapeo es único y explícito.
//
// Estrategia: errors.As para errores con datos (MissingIDsError), errors.Is
// para errores sentinel (ErrProductNotFound, ErrInvalidField, etc.).
func writeError(c *gin.Context, err error) {
	// Caso especial: missing IDs — devolvemos 404 + lista de qué falta.
	var miss *domain.MissingIDsError
	if errors.As(err, &miss) {
		c.JSON(http.StatusNotFound, errorResponse{
			Error:      err.Error(),
			MissingIDs: miss.Missing,
		})
		return
	}

	switch {
	case errors.Is(err, domain.ErrProductNotFound):
		c.JSON(http.StatusNotFound, errorResponse{Error: err.Error()})
	case errors.Is(err, domain.ErrEmptyIDs),
		errors.Is(err, domain.ErrInvalidField),
		errors.Is(err, domain.ErrInvalidPagination),
		errors.Is(err, domain.ErrTooManyIDs):
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
	default:
		// Registramos la causa para observabilidad interna: el payload al cliente
		// sigue genérico, pero queda disponible para el logger en el contexto.
		_ = c.Error(err).SetType(gin.ErrorTypePrivate)
		// Default: 500. NO exponemos el error interno al cliente —
		// log lo loguea (en main), respuesta es genérica para no filtrar
		// detalles de implementación. Trade-off: el cliente pierde detalle,
		// gana seguridad.
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}
