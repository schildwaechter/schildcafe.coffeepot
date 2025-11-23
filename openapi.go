package main

import _ "embed"

// openAPISpec contains the OpenAPI document served at /openapi.yaml.
//
//go:embed openapi.yaml
var openAPISpec []byte
