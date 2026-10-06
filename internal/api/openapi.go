package api

import (
	"encoding/json"
	"net/http"
)

func RegisterOpenAPI(mux *http.ServeMux, wrap func(http.Handler) http.Handler) {
	openAPI := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(openAPISpec)
	})
	swagger := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(swaggerHTML))
	})
	mux.Handle("/openapi.json", wrap(openAPI))
	mux.Handle("/swagger", wrap(swagger))
}

var openAPISpec = map[string]any{
	"openapi": "3.0.3",
	"info": map[string]any{
		"title":       "Alauda Service Registry API",
		"version":     "1.0.0",
		"description": "Read-only catalog and service discovery API for external clients.",
	},
	"servers": []map[string]string{{"url": "/"}},
	"components": map[string]any{
		"securitySchemes": map[string]any{
			"bearerAuth": map[string]string{"type": "http", "scheme": "bearer"},
		},
		"schemas": map[string]any{
			"Pagination":      map[string]any{"type": "object", "properties": map[string]any{"nextPageToken": map[string]string{"type": "string"}, "totalSize": map[string]string{"type": "integer", "format": "int32"}}},
			"CatalogResponse": map[string]any{"type": "object", "additionalProperties": true},
			"Error":           map[string]any{"type": "object", "required": []string{"error"}, "properties": map[string]string{"error": "string"}},
		},
	},
	"security": []map[string][]string{{"bearerAuth": {}}},
	"paths": map[string]any{
		"/api/v1/environments":                                         canonicalPath("List enabled environments."),
		"/api/v1/environments/{environmentKey}":                        canonicalPath("Get an environment by its public key.", "environmentKey"),
		"/api/v1/environments/{environmentKey}/services":               canonicalPath("List services deployed in an environment.", "environmentKey"),
		"/api/v1/environments/{environmentKey}/services/{serviceName}": canonicalPath("Get a service deployed in an environment.", "environmentKey", "serviceName"),
		"/api/v1/catalog/environments":                                 catalogPath("List enabled or all environments.", "includeDisabled"),
		"/api/v1/catalog/services":                                     catalogPath("List services, optionally filtered by environment.", "environmentId", "environment"),
		"/api/v1/catalog/deployments":                                  catalogPath("List service deployments.", "serviceId", "environmentId", "environment"),
		"/api/v1/catalog/instances":                                    catalogPath("List service instances.", "deploymentId"),
		"/api/v1/catalog/endpoints":                                    catalogPath("List instance endpoints.", "instanceId"),
		"/api/v1/discovery/services/{serviceName}/resolve": map[string]any{"get": map[string]any{
			"summary":    "Resolve a service in an environment",
			"parameters": []map[string]any{{"name": "serviceName", "in": "path", "required": true, "schema": map[string]string{"type": "string"}}, {"name": "environment", "in": "query", "required": true, "schema": map[string]string{"type": "string"}}, {"name": "healthyOnly", "in": "query", "schema": map[string]string{"type": "boolean", "default": "false"}}},
			"responses":  standardResponses(),
		}},
	},
}

func catalogPath(summary string, filters ...string) map[string]any {
	parameters := []map[string]any{{"name": "pageSize", "in": "query", "schema": map[string]any{"type": "integer", "minimum": 1, "maximum": 200, "default": 50}}, {"name": "pageToken", "in": "query", "schema": map[string]string{"type": "string"}}}
	for _, filter := range filters {
		parameters = append(parameters, map[string]any{"name": filter, "in": "query", "schema": map[string]string{"type": "string"}})
	}
	return map[string]any{"get": map[string]any{"summary": summary, "parameters": parameters, "responses": standardResponses()}}
}

func canonicalPath(summary string, pathParameters ...string) map[string]any {
	parameters := make([]map[string]any, 0, len(pathParameters))
	for _, name := range pathParameters {
		parameters = append(parameters, map[string]any{"name": name, "in": "path", "required": true, "schema": map[string]string{"type": "string"}})
	}
	return map[string]any{"get": map[string]any{"summary": summary, "parameters": parameters, "responses": standardResponses()}}
}

func standardResponses() map[string]any {
	return map[string]any{
		"200": map[string]any{"description": "Successful response", "content": map[string]any{"application/json": map[string]any{"schema": map[string]string{"$ref": "#/components/schemas/CatalogResponse"}}}},
		"400": map[string]string{"description": "Invalid request"},
		"401": map[string]string{"description": "Authentication required"},
		"403": map[string]string{"description": "Insufficient permissions"},
		"404": map[string]string{"description": "Resource not found"},
		"500": map[string]string{"description": "Server error"},
	}
}

const swaggerHTML = `<!doctype html>
<html><head><title>Alauda API</title><link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css"></head>
<body><div id="swagger-ui"></div><script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>window.onload=()=>SwaggerUIBundle({url:'/openapi.json',dom_id:'#swagger-ui'});</script></body></html>`
