Este es un problema de hackerrank para un entrevista tecnica en mercado libre, el problema es el siguiente:

tengo que construir un backend api that suplpies products details for use in an item compararion feature. ypur implementation should follow established backend best practices, providinde clear and efficient endpoints to retrieve the required data for product comparisions

## Requirements:
backend api development
api endpoints
* build a restful api that returns details for multiples items to be compared
* the api shoul provide fields such as product name, image url, description, price, rating, and specifications
* include error handling and inline comments to explain your logic

## Stack:
* you can use any backend technology or framweork of your choice
* simulate data persistence using local json/csv files or an in-memory database (e.g., SQLite, H2 Database) to represent the inventory. A real database is not required

## Function requirements:
the product model should encapsulate essential information, including but not limited to the following attributes: ID, name, description, price,size , weight, and color. Additionally, certain products may requiere specialized information. For example, a smartphone shoul include specefic details such as battery capacity, camera specificacttions, memory, storage capacity, brand, model version, and operating system. A user should be able to query specefic comparisions between items and ignore other fields. This will help them focus on the most relevant details for their analysis

## Non-functional requirements:
special consideration will be given to goo practices in error handling, documentation, testing, and any other relevant relevant non-functional aspects you choose demonstrate.

## Documentation & strategic overview:
Please include a bried README or Diagram (optional) that explains your API design, main endpoints, setup instructions, and any key architectural decisions you made during development