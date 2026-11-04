// Package docs holds the OpenAPI specification served by the Swagger UI.
package docs

import _ "embed"

//go:embed swagger.json
var swaggerJSON []byte

// SwaggerJSON returns the embedded OpenAPI document.
func SwaggerJSON() []byte {
	return swaggerJSON
}
