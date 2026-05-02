package domain

import (
	"errors"
	"fmt"
	"strings"
)

// Errores tipados del dominio.
//
// Patrón Go idiomático con sentinel errors: las capas externas (handlers HTTP)
// usan errors.Is() para mapear estos errores a status codes sin acoplarse a
// strings de mensaje. Cambiar el texto de un error nunca rompe el mapeo.
var (
	// ErrProductNotFound se retorna cuando un producto puntual no existe.
	ErrProductNotFound = errors.New("product not found")

	// ErrInvalidField indica que el cliente pidió un campo fuera de la whitelist.
	ErrInvalidField = errors.New("invalid field requested")

	// ErrEmptyIDs indica que se invocó una operación que requiere IDs sin pasar ninguno.
	ErrEmptyIDs = errors.New("at least one id is required")

	// ErrInvalidPagination cubre offset/limit fuera de rango.
	ErrInvalidPagination = errors.New("invalid pagination parameters")
)

// MissingIDsError representa el caso donde algunos IDs solicitados no existen.
//
// A diferencia de ErrProductNotFound (un error simple), este error CARGA datos:
// la lista exacta de IDs faltantes. El cliente necesita saber QUÉ falta para
// corregir su request — comparar productos parcialmente sería un resultado
// degradado y silencioso, lo cual es peor que fallar explícitamente.
//
// Implementa la convención `Unwrap` para permitir errors.Is(err, ErrProductNotFound).
type MissingIDsError struct {
	Missing []string
}

func (e *MissingIDsError) Error() string {
	return fmt.Sprintf("products not found: %s", strings.Join(e.Missing, ", "))
}

// Unwrap permite que errors.Is(missingErr, ErrProductNotFound) funcione.
// Útil para handlers que quieran tratar ambos casos como 404.
func (e *MissingIDsError) Unwrap() error {
	return ErrProductNotFound
}
