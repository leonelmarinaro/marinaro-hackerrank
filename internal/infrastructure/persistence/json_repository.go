// Package persistence contiene los adapters de persistencia que implementan
// el puerto domain.ProductRepository.
//
// Capa externa de Clean Architecture: depende de domain pero domain no la
// conoce (inversión de dependencias).
package persistence

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"

	"github.com/lmarinaro/marinaro-hackerrank/internal/domain"
)

// JSONRepository implementa ProductRepository sobre un archivo JSON cargado
// en memoria al boot.
//
// Decisión: cargar TODO al arranque en lugar de leer el archivo en cada query.
// Razón: el dataset es chico y estático; leer el archivo en cada request sería
// I/O innecesario. Trade-off aceptado: si el archivo cambia en runtime no se
// reflejan los cambios sin reiniciar el servicio. Para un challenge con datos
// fijos es una decisión correcta — para producción cambiaríamos a una DB real.
//
// Concurrencia: usamos sync.RWMutex aunque hoy nunca escribimos. Por qué:
//  1. Si mañana agregamos un endpoint POST /products, ya está cubierto.
//  2. RLock es prácticamente free en lecturas concurrentes.
//  3. Cero coste cognitivo, ganamos seguridad por defecto.
type JSONRepository struct {
	mu       sync.RWMutex
	products []domain.Product
	byID     map[string]domain.Product // index para FindByID O(1)
}

// NewJSONRepository carga el archivo JSON desde path y devuelve un repo listo.
//
// Errores: si el archivo no existe o tiene JSON inválido, retorna error con
// contexto. NO usamos fallbacks silenciosos — un servicio sin catálogo no
// debería arrancar pretendiendo estar sano.
func NewJSONRepository(path string) (*JSONRepository, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading products file %q: %w", path, err)
	}

	var products []domain.Product
	if err := json.Unmarshal(data, &products); err != nil {
		return nil, fmt.Errorf("parsing products JSON: %w", err)
	}

	byID := make(map[string]domain.Product, len(products))
	for _, p := range products {
		if _, dup := byID[p.ID]; dup {
			return nil, fmt.Errorf("duplicate product id in dataset: %q", p.ID)
		}
		byID[p.ID] = p
	}

	return &JSONRepository{
		products: products,
		byID:     byID,
	}, nil
}

// FindByID retorna un producto por ID o ErrProductNotFound.
func (r *JSONRepository) FindByID(id string) (*domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.byID[id]
	if !ok {
		return nil, domain.ErrProductNotFound
	}
	return &p, nil
}

// FindByIDs retorna los productos solicitados o *MissingIDsError con los faltantes.
//
// Mantiene el orden del input — el cliente pidió comparar [3, 1, 2] y eso es
// lo que recibe. Útil porque la UI suele renderizar en el orden recibido.
func (r *JSONRepository) FindByIDs(ids []string) ([]domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	found := make([]domain.Product, 0, len(ids))
	missing := make([]string, 0)

	for _, id := range ids {
		p, ok := r.byID[id]
		if !ok {
			missing = append(missing, id)
			continue
		}
		found = append(found, p)
	}

	if len(missing) > 0 {
		return nil, &domain.MissingIDsError{Missing: missing}
	}
	return found, nil
}

// List retorna una página de productos.
func (r *JSONRepository) List(offset, limit int) ([]domain.Product, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	total := len(r.products)
	if offset >= total {
		// Página fuera de rango: retornamos vacío + total.
		// Decisión: no es un error — es información válida ("ya iteraste todo").
		return []domain.Product{}, total, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	// Copiamos el slice para no exponer el array interno (defensive copy).
	page := make([]domain.Product, end-offset)
	copy(page, r.products[offset:end])

	return page, total, nil
}

// Categories retorna las categorías únicas ordenadas alfabéticamente.
//
// Output ordenado: facilita testing determinístico y la UI puede renderizarlo
// directamente sin re-ordenar.
func (r *JSONRepository) Categories() ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	seen := make(map[string]struct{})
	for _, p := range r.products {
		seen[p.Category] = struct{}{}
	}

	cats := make([]string, 0, len(seen))
	for c := range seen {
		cats = append(cats, c)
	}
	sort.Strings(cats)

	return cats, nil
}
