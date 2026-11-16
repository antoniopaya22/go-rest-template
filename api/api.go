// Package api embeds the OpenAPI contract, the source of truth of the HTTP surface.
package api

import _ "embed"

// Spec is the OpenAPI document served at /openapi.yaml and used to validate traffic.
//
//go:embed openapi.yaml
var Spec []byte
