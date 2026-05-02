package domain

// ProductRepository es el puerto que la capa de aplicación usa para acceder
// a productos.
//
// Decisión clave: la interfaz vive en `domain` (NO en application).
// Esto sigue el principio de Inversión de Dependencias de Clean Architecture:
// el dominio DEFINE lo que necesita, la infraestructura lo IMPLEMENTA.
//
// Beneficios concretos:
//   - Testeamos use cases con mocks del repo (ver application/*_test.go).
//   - Cambiar de JSON a SQLite o Postgres no requiere tocar use cases.
//   - El dominio no sabe (ni le importa) cómo se persisten los datos.
type ProductRepository interface {
	// FindByID retorna un producto puntual.
	// Error: ErrProductNotFound si no existe.
	FindByID(id string) (*Product, error)

	// FindByIDs retorna los productos solicitados, en el orden recibido.
	// Si algún ID falta retorna *MissingIDsError con la lista exacta —
	// semántica "todo o nada" pero con detalle del fallo.
	FindByIDs(ids []string) ([]Product, error)

	// List retorna productos paginados.
	//   - offset: 0-based, primer elemento a retornar
	//   - limit: cantidad máxima a retornar (>0)
	// Devuelve además el total de productos en el catálogo (para paginación).
	List(offset, limit int) ([]Product, int, error)

	// Categories retorna las categorías distintas presentes en el catálogo,
	// ordenadas alfabéticamente para output determinístico.
	Categories() ([]string, error)
}
