# Desafío Técnico — Mercado Libre

> Challenge de HackerRank para entrevista técnica en Mercado Libre.

## Contexto

Hay que construir un **backend API** que provea detalles de productos para una feature de **comparación de items**. La implementación debe seguir buenas prácticas de backend, con endpoints claros y eficientes para recuperar la información necesaria para comparar productos.

## Requerimientos

### Desarrollo del API y endpoints
- Construir un **API RESTful** que retorne detalles de múltiples items para ser comparados.
- El API debe exponer campos como: nombre del producto, URL de imagen, descripción, precio, rating y especificaciones.
- Incluir **manejo de errores** y comentarios inline que expliquen la lógica.

### Stack
- Se puede usar cualquier tecnología o framework de backend.
- Simular la persistencia con archivos locales JSON/CSV o una base de datos en memoria (por ejemplo SQLite, H2). **No se requiere una base de datos real.**

### Requerimientos funcionales
- El **modelo de producto** debe encapsular información esencial, incluyendo (pero no limitado a) los siguientes atributos: `id`, `name`, `description`, `price`, `size`, `weight`, `color`.
- Algunos productos pueden requerir información especializada. Por ejemplo, un **smartphone** debe incluir detalles específicos como: capacidad de batería, especificaciones de cámara, memoria, capacidad de almacenamiento, marca, versión del modelo y sistema operativo.
- El usuario debe poder **consultar comparaciones específicas entre items e ignorar otros campos**. Esto le permite enfocarse en los detalles más relevantes para su análisis.

### Requerimientos no funcionales
Se valorará especialmente: buenas prácticas en manejo de errores, documentación, testing, y cualquier otro aspecto no funcional relevante que decidas demostrar.

### Documentación y visión estratégica
Incluir un breve **README** o diagrama (opcional) que explique:
- Diseño del API
- Endpoints principales
- Instrucciones de setup
- Decisiones arquitectónicas clave tomadas durante el desarrollo
