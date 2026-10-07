// Package api embeds the OpenAPI spec and the Swagger UI page served at /docs.
package api

import _ "embed"

//go:embed openapi.yaml
var Spec []byte

//go:embed docs.html
var DocsPage []byte
