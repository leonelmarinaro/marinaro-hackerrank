// Package http contiene los handlers Gin que adaptan HTTP a casos de uso.
//
// Capa más externa de Clean Architecture. Responsabilidades:
//   - Parsear request (query/path params) en valores del dominio.
//   - Invocar el use case correspondiente.
//   - Mapear errores del dominio a status codes HTTP.
//   - Serializar la respuesta.
//
// NO hay lógica de negocio acá — solo plumbing HTTP.
package http

// errorResponse es el formato uniforme de error devuelto al cliente.
//
// Decisión: estructura única para todos los errores en lugar de strings
// libres. Beneficios: el frontend puede parsear consistentemente, y los
// errores ricos (como missing_ids) caben en el mismo contrato.
type errorResponse struct {
	Error      string   `json:"error"`
	MissingIDs []string `json:"missing_ids,omitempty"`
}

// healthResponse: respuesta simple del healthcheck.
type healthResponse struct {
	Status string `json:"status"`
}
