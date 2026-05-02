// Package domain contiene las entidades de negocio y los puertos (interfaces)
// que el dominio necesita del mundo externo.
//
// Capa más interna de la Clean Architecture: NO depende de paquetes de
// infraestructura (HTTP, JSON, DB). Solo stdlib. Esto la hace 100% testeable
// sin levantar nada externo y la blinda contra cambios en frameworks.
package domain

// Product representa un item del catálogo de Mercado Libre.
//
// Decisión arquitectónica: campos comunes tipados + Specs como map[string]any.
// Trade-off: perdemos type-safety en specs (no detectamos typos en compile time)
// pero ganamos extensibilidad — agregar una categoría nueva (smartphone, libro,
// ropa, electrodoméstico) NO requiere cambiar el schema. Para un catálogo de
// e-commerce real con miles de SKUs heterogéneos, la flexibilidad gana.
//
// Alternativas consideradas y descartadas:
//  1. Structs por categoría (Smartphone, Book) con embedding: rígido,
//     cada categoría nueva requiere código nuevo.
//  2. JSONB-like con todo en map: pierde el contrato de los campos comunes.
type Product struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	ImageURL    string         `json:"image_url"`
	Price       float64        `json:"price"`
	Rating      float64        `json:"rating"`
	Category    string         `json:"category"`
	Size        string         `json:"size,omitempty"`
	Weight      float64        `json:"weight,omitempty"`
	Color       string         `json:"color,omitempty"`
	Specs       map[string]any `json:"specs,omitempty"`
}

// allowedFields es la whitelist de campos seleccionables vía API.
//
// Por qué whitelist (no blacklist): si mañana agregamos un campo interno
// (ej: cost_price, internal_sku) no se filtrará accidentalmente al cliente.
// La whitelist es explícita y segura por defecto.
var allowedFields = map[string]struct{}{
	"id":          {},
	"name":        {},
	"description": {},
	"image_url":   {},
	"price":       {},
	"rating":      {},
	"category":    {},
	"size":        {},
	"weight":      {},
	"color":       {},
	"specs":       {},
}

// IsAllowedField indica si un nombre de campo forma parte de la API pública.
// Usado por la capa de aplicación para validar el query param `fields`.
func IsAllowedField(name string) bool {
	_, ok := allowedFields[name]
	return ok
}

// SelectFields retorna una proyección del producto con solo los campos pedidos.
// Si fields está vacío o nil, retorna el producto completo.
//
// Decisión: devolver map[string]any en lugar de un *Product proyectado.
// Razón: el requerimiento es OMITIR del JSON los campos no pedidos. Un struct
// con campos opcionales serializaría como `null` o como cero — no como ausentes.
// El map nos permite control total sobre las claves del output.
func (p Product) SelectFields(fields []string) map[string]any {
	full := p.toMap()
	if len(fields) == 0 {
		return full
	}
	out := make(map[string]any, len(fields))
	for _, f := range fields {
		if v, ok := full[f]; ok {
			out[f] = v
		}
		// Campo desconocido se ignora silenciosamente. La validación contra
		// la whitelist ocurre antes (en el use case) — si llegamos acá con
		// un campo inválido es porque el llamador eligió ignorarlo.
	}
	return out
}

// toMap convierte el producto a map para proyección dinámica.
// Privado intencional: la API pública del entity es SelectFields.
//
// Los campos opcionales (size, weight, color, specs) solo se incluyen si
// tienen valor — coherente con `omitempty` del JSON tag y con la idea de
// que omitir > mostrar vacío para campos no aplicables.
func (p Product) toMap() map[string]any {
	m := map[string]any{
		"id":          p.ID,
		"name":        p.Name,
		"description": p.Description,
		"image_url":   p.ImageURL,
		"price":       p.Price,
		"rating":      p.Rating,
		"category":    p.Category,
	}
	if p.Size != "" {
		m["size"] = p.Size
	}
	if p.Weight != 0 {
		m["weight"] = p.Weight
	}
	if p.Color != "" {
		m["color"] = p.Color
	}
	if len(p.Specs) > 0 {
		m["specs"] = p.Specs
	}
	return m
}
