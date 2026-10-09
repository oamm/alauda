package api

import (
	"encoding/json"
	"github.com/company/service-registry/internal/contract"
	"github.com/company/service-registry/internal/problem"
	"net/http"
	"reflect"
	"strings"
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

var openAPISpec = buildPublicOpenAPI()

type schemaMap = map[string]any

func schemaRef(name string) schemaMap { return schemaMap{"$ref": "#/components/schemas/" + name} }
func stringSchema() schemaMap         { return schemaMap{"type": "string"} }
func objectSchema(properties schemaMap, required ...string) schemaMap {
	return schemaMap{"type": "object", "additionalProperties": false, "properties": properties, "required": required}
}
func arraySchema(items schemaMap) schemaMap { return schemaMap{"type": "array", "items": items} }
func dtoSchema(t reflect.Type) schemaMap {
	if t.Kind() == reflect.Pointer {
		s := dtoSchema(t.Elem())
		s["nullable"] = true
		return s
	}
	switch t.Kind() {
	case reflect.String:
		return stringSchema()
	case reflect.Bool:
		return schemaMap{"type": "boolean"}
	case reflect.Int, reflect.Int32, reflect.Int64:
		return schemaMap{"type": "integer"}
	case reflect.Slice:
		s := arraySchema(dtoSchema(t.Elem()))
		s["nullable"] = true
		return s
	case reflect.Map:
		return schemaMap{"type": "object", "additionalProperties": dtoSchema(t.Elem()), "nullable": true}
	case reflect.Struct:
		props := schemaMap{}
		required := []string{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			tag := f.Tag.Get("json")
			if tag == "-" || tag == "" {
				continue
			}
			name := strings.Split(tag, ",")[0]
			props[name] = dtoSchema(f.Type)
			if !strings.Contains(tag, "omitempty") {
				required = append(required, name)
			}
		}
		return objectSchema(props, required...)
	}
	panic("unsupported public schema type: " + t.String())
}
func queryParameter(name string, required bool, schema schemaMap) schemaMap {
	return schemaMap{"name": name, "in": "query", "required": required, "schema": schema}
}
func publicSpecResponses(code, schema string) schemaMap {
	responses := schemaMap{}
	if code == "204" {
		responses[code] = schemaMap{"description": "Deregistered or already absent"}
	} else {
		responses[code] = schemaMap{"description": "Successful response", "content": schemaMap{"application/json": schemaMap{"schema": schemaRef(schema)}}}
	}
	for _, status := range []string{"400", "401", "403", "404", "409", "413", "415", "429", "500", "503"} {
		responses[status] = schemaMap{"description": "Problem Details with stable code", "content": schemaMap{"application/problem+json": schemaMap{"schema": schemaRef("ProblemDetails")}}}
	}
	return responses
}
func buildPublicOpenAPI() schemaMap {
	key := schemaMap{"type": "string", "minLength": 1, "maxLength": 128, "pattern": "^[A-Za-z0-9._-]+$"}
	stringMap := schemaMap{"type": "object", "additionalProperties": stringSchema(), "nullable": true}
	schemas := schemaMap{}
	for name, value := range map[string]any{"Registration": contract.Registration{}, "RegistrationResponse": publicRegistrationResponse{}, "Instance": publicInstance{}, "Endpoint": publicEndpoint{}, "Discovery": publicDiscoveryResponse{}, "HealthCheck": publicHealthCheck{}, "HealthResult": publicHealthResult{}, "ProblemDetails": problem.Details{}} {
		schemas[name] = dtoSchema(reflect.TypeOf(value))
	}
	registration := schemas["Registration"].(schemaMap)
	checkProps := schemas["HealthCheck"].(schemaMap)["properties"].(schemaMap)
	checkProps["type"] = schemaMap{"type": "string", "enum": []string{"http", "tcp", "dns"}}
	checkProps["intervalSeconds"] = schemaMap{"type": "integer", "minimum": 1, "maximum": 86400, "default": 30}
	checkProps["timeoutSeconds"] = schemaMap{"type": "integer", "minimum": 1, "maximum": 300, "default": 5}
	checkProps["failuresBeforeUnhealthy"] = schemaMap{"type": "integer", "minimum": 1, "default": 3, "deprecated": true, "description": "Compatibility field; current health follows the latest completed result."}
	checkProps["successesBeforeHealthy"] = schemaMap{"type": "integer", "minimum": 1, "default": 2, "deprecated": true, "description": "Compatibility field; current health follows the latest completed result."}
	schemas["HealthCheckCreate"] = objectSchema(checkProps, "name", "endpoint", "type")
	schemas["Instance"].(schemaMap)["properties"].(schemaMap)["healthState"] = schemaMap{"type": "string", "enum": []string{"Healthy", "Unknown", "Degraded", "Unhealthy", "Disabled"}}
	registration["required"] = []string{"environment", "instance"}
	rp := registration["properties"].(schemaMap)
	rp["environment"] = key
	rp["mode"] = schemaMap{"type": "string", "enum": []string{"upsert"}, "default": "upsert"}
	rp["replaceEndpoints"] = schemaMap{"type": "boolean", "default": false}
	ep := rp["endpoints"].(schemaMap)["items"].(schemaMap)["properties"].(schemaMap)
	ep["port"] = schemaMap{"type": "integer", "minimum": 1, "maximum": 65535, "nullable": true}
	ep["protocol"] = schemaMap{"type": "string", "enum": []string{"http", "https", "tcp", "udp", "grpc"}, "nullable": true}
	ep["name"] = key
	ep["path"] = schemaMap{"type": "string", "nullable": true, "description": "Optional HTTP/HTTPS path without query, fragment or authority. TCP/UDP/gRPC require empty or omitted path; incompatible values are validation errors."}
	ip := rp["instance"].(schemaMap)["properties"].(schemaMap)
	ip["name"] = key
	ip["address"] = schemaMap{"type": "string", "description": "Bare DNS hostname, IPv4 or IPv6; scheme and embedded port are rejected.", "nullable": true}
	registration["example"] = schemaMap{"environment": "stg", "instance": schemaMap{"name": "lynx-authentication.lynx", "address": "lynx-authentication.lynx"}, "endpoints": []any{schemaMap{"name": "default", "protocol": "http", "port": 81, "path": "/"}}}
	schemas["Resolve"] = objectSchema(schemaMap{"service": stringSchema(), "environment": stringSchema(), "instance": stringSchema(), "endpoint": stringSchema(), "address": schemaMap{"type": "string", "format": "uri"}}, "service", "environment", "instance", "endpoint", "address")
	healthStatus := schemaMap{"type": "string", "enum": []string{"Healthy", "Degraded", "Unhealthy", "Unknown", "Disabled"}}
	schemas["Service"] = objectSchema(schemaMap{"id": stringSchema(), "name": key, "displayName": stringSchema(), "description": stringSchema(), "tags": stringMap, "metadata": stringMap, "healthStatus": healthStatus}, "id", "name", "displayName", "description", "tags", "metadata", "healthStatus")
	schemas["HealthStatusSnapshot"] = objectSchema(schemaMap{"services": schemaMap{"type": "object", "additionalProperties": healthStatus}, "instances": schemaMap{"type": "object", "additionalProperties": healthStatus}, "monitored": schemaMap{"type": "object", "additionalProperties": schemaMap{"type": "boolean"}}}, "services", "instances", "monitored")
	schemas["ServiceCreate"] = objectSchema(schemaMap{"name": key, "displayName": stringSchema(), "description": stringSchema(), "tags": stringMap, "metadata": stringMap}, "name")
	schemas["EnvironmentCreate"] = objectSchema(schemaMap{"key": key, "name": stringSchema(), "description": stringSchema(), "tags": stringMap}, "key")
	schemas["Environment"] = objectSchema(schemaMap{"id": stringSchema(), "key": key, "name": stringSchema(), "description": stringSchema(), "enabled": schemaMap{"type": "boolean"}, "tier": stringSchema(), "tags": stringMap, "createdAt": schemaMap{"type": "string", "format": "date-time"}, "updatedAt": schemaMap{"type": "string", "format": "date-time"}}, "id", "key", "name", "enabled")
	schemas["Pagination"] = objectSchema(schemaMap{"nextPageToken": stringSchema(), "totalSize": schemaMap{"type": "integer"}})
	for name, item := range map[string]string{"Services": "Service", "Environments": "Environment", "Instances": "Instance", "Endpoints": "Endpoint", "Checks": "HealthCheck", "Results": "HealthResult"} {
		field := strings.ToLower(name)
		props := schemaMap{field: arraySchema(schemaRef(item)), "nextPageToken": stringSchema()}
		if name == "Instances" || name == "Endpoints" {
			props["service"] = stringSchema()
			props["environment"] = stringSchema()
		}
		if name == "Endpoints" {
			props["instance"] = stringSchema()
		}
		if name == "Environments" {
			props["pagination"] = schemaRef("Pagination")
		}
		if name == "Services" {
			props["pagination"] = schemaRef("Pagination")
		}
		schemas[name] = objectSchema(props, field)
	}
	schemas["HealthRun"] = objectSchema(schemaMap{"check": stringSchema(), "instance": stringSchema(), "success": schemaMap{"type": "boolean"}, "timestamp": schemaMap{"type": "string", "format": "date-time"}, "healthState": stringSchema()}, "check", "instance", "success", "timestamp", "healthState")
	paths := schemaMap{}
	add := func(path, method, summary, capability, response, code, body string, environment, pagination bool) {
		params := []any{}
		for _, part := range strings.Split(path, "/") {
			if strings.HasPrefix(part, "{") {
				params = append(params, schemaMap{"name": strings.Trim(part, "{}"), "in": "path", "required": true, "schema": key})
			}
		}
		if environment {
			params = append(params, queryParameter("environment", true, key))
		}
		if pagination {
			params = append(params, queryParameter("pageSize", false, schemaMap{"type": "integer", "minimum": 1, "maximum": 200, "default": 50}), queryParameter("pageToken", false, stringSchema()))
		}
		op := schemaMap{"summary": summary, "description": "Requires " + capability + ". Environment restrictions are enforced by the backend before pagination.", "x-capability": capability, "tags": []string{"PUBLIC"}, "parameters": params, "responses": publicSpecResponses(code, response)}
		if body != "" {
			op["requestBody"] = schemaMap{"required": true, "content": schemaMap{"application/json": schemaMap{"schema": schemaRef(body)}}}
		}
		if paths[path] == nil {
			paths[path] = schemaMap{}
		}
		paths[path].(schemaMap)[method] = op
	}
	add("/api/v1/services", "get", "List Services", "registry.read", "Services", "200", "", false, true)
	add("/api/v1/health/status", "get", "Read current Service and Instance health snapshot", "health.read", "HealthStatusSnapshot", "200", "", false, false)
	paths["/api/v1/health/status"].(schemaMap)["get"].(schemaMap)["parameters"] = []any{queryParameter("environment", false, key)}
	paths["/api/v1/services"].(schemaMap)["get"].(schemaMap)["parameters"] = append(paths["/api/v1/services"].(schemaMap)["get"].(schemaMap)["parameters"].([]any), queryParameter("environment", false, key))
	add("/api/v1/services", "post", "Explicit Service bootstrap", "registry.write", "Service", "201", "ServiceCreate", false, false)
	add("/api/v1/environments", "get", "List Environments, including disabled", "registry.read", "Environments", "200", "", false, true)
	add("/api/v1/environments", "post", "Explicit Environment bootstrap", "registry.write", "Environment", "201", "EnvironmentCreate", false, false)
	add("/api/v1/environments/{environmentKey}", "get", "Get Environment by key", "registry.read", "Environment", "200", "", false, false)
	add("/api/v1/environments/{environmentKey}/services", "get", "List Environment Services", "registry.read", "Services", "200", "", false, true)
	add("/api/v1/environments/{environmentKey}/services/{serviceName}", "get", "Get Environment Service", "registry.read", "Service", "200", "", false, false)
	base := "/api/v1/services/{serviceKey}"
	add(base, "get", "Get Service", "registry.read", "Service", "200", "", true, false)
	add(base+"/instances", "get", "List Instances regardless of discovery eligibility", "registry.read", "Instances", "200", "", true, true)
	add(base+"/instances", "post", "Atomic idempotent registration UPSERT", "registry.write", "RegistrationResponse", "200", "Registration", false, false)
	op := paths[base+"/instances"].(schemaMap)["post"].(schemaMap)
	op["description"] = "Identity: Service + Environment + Instance name. Omitted fields preserve values; explicit empty tags/metadata clear them. Matching Endpoint names are updated, new names created, omitted names preserved unless replaceEndpoints=true. Primary promotion atomically demotes the old Primary. A new singleton with Primary omitted defaults to Primary; existing incremental singletons do not. Only mode=upsert is supported. Service and Environment must exist and Environment must be enabled. Requires registry.write."
	add(base+"/instances/{instanceName}", "get", "Get Instance", "registry.read", "Instance", "200", "", true, false)
	add(base+"/instances/{instanceName}", "delete", "Idempotent deregistration preserving history and retained endpoints/checks", "registry.write", "", "204", "", true, false)
	add(base+"/instances/{instanceName}/endpoints", "get", "List registered Endpoints including disabled", "registry.read", "Endpoints", "200", "", true, true)
	hb := base + "/instances/{instanceName}/health-checks"
	add(hb, "get", "List Health Checks", "health.read", "Checks", "200", "", true, true)
	add(hb, "post", "Create named Health Check bound to Endpoint", "health.write", "HealthCheck", "201", "HealthCheckCreate", true, false)
	add(hb+"/{checkName}/run", "post", "Execute active Health Check", "health.execute", "HealthRun", "200", "", true, false)
	add(base+"/health-results", "get", "List Health history using public names", "health.read", "Results", "200", "", true, true)
	resultsOp := paths[base+"/health-results"].(schemaMap)["get"].(schemaMap)
	for _, name := range []string{"instance", "endpoint", "check", "from", "to"} {
		schema := stringSchema()
		if name == "from" || name == "to" {
			schema["format"] = "date-time"
		}
		resultsOp["parameters"] = append(resultsOp["parameters"].([]any), queryParameter(name, false, schema))
	}
	for _, resolve := range []bool{false, true} {
		path := "/api/v1/discovery/{serviceKey}"
		model := "Discovery"
		if resolve {
			path += "/resolve"
			model = "Resolve"
		}
		add(path, "get", "Discover usable endpoints without ID lookup", "discovery.read", model, "200", "", true, false)
		op := paths[path].(schemaMap)["get"].(schemaMap)
		op["parameters"] = append(op["parameters"].([]any), queryParameter("endpoint", false, key), queryParameter("health", false, schemaMap{"type": "string", "enum": []string{"usable", "healthy", "all"}, "default": "usable"}))
		op["description"] = "Enabled Environment/Instances/Endpoints only. usable excludes known Unhealthy/Disabled and allows Unknown/Degraded; healthy requires fresh Healthy; all includes health states but not disabled lifecycle resources. Missing checks, disabled monitoring and stale health become Unknown. Healthy candidates rank first, then name order. Resolve selects named Endpoint, otherwise Primary, then lexicographic Endpoint name. No load balancing. Cache-Control: no-store."
	}

	schemas["Resolve"].(schemaMap)["example"] = schemaMap{"service": "Authentication.Grpc", "environment": "stg", "instance": "lynx-authentication.lynx", "endpoint": "default", "address": "http://lynx-authentication.lynx:81/"}
	exampleInstance := publicInstance{Name: "lynx-authentication.lynx", Address: "lynx-authentication.lynx", Enabled: true, HealthState: "Unknown"}
	exampleEndpoint := publicEndpoint{Name: "default", Protocol: "http", Port: 81, Path: "/", Primary: true, Enabled: true, Address: "http://lynx-authentication.lynx:81/"}
	exampleInstance.Endpoints = []publicEndpoint{exampleEndpoint}
	schemas["RegistrationResponse"].(schemaMap)["example"] = publicRegistrationResponse{Service: "Authentication.Grpc", Environment: "stg", Instance: exampleInstance, Endpoints: []publicEndpoint{exampleEndpoint}}
	schemas["Discovery"].(schemaMap)["example"] = publicDiscoveryResponse{Service: "Authentication.Grpc", Environment: "stg", Instances: []publicInstance{exampleInstance}}
	schemas["ProblemDetails"].(schemaMap)["example"] = problem.Details{Type: "https://alauda.dev/problems/service-not-found", Title: "Not Found", Status: 404, Code: "service_not_found", Detail: "Service does not exist."}
	schemas["ServiceCreate"].(schemaMap)["example"] = schemaMap{"name": "Authentication.Grpc"}
	schemas["EnvironmentCreate"].(schemaMap)["example"] = schemaMap{"key": "stg", "name": "Staging"}
	return schemaMap{"openapi": "3.0.3", "info": schemaMap{"title": "Alauda Public API", "version": "1.0.0", "description": "Stable Service / Environment / Instance / Endpoint / Health Check contract. Bearer API tokens and Application Keys supported. Legacy read/write scopes map to documented capabilities; admin grants all. Scoped credentials cannot bootstrap global resources. Connect RPCs used by the UI are internal contracts. Removed catalog/discovery compatibility APIs are not supported; see docs/API.md."}, "servers": []any{schemaMap{"url": "/"}}, "security": []any{schemaMap{"bearerAuth": []string{}}}, "components": schemaMap{"securitySchemes": schemaMap{"bearerAuth": schemaMap{"type": "http", "scheme": "bearer"}}, "schemas": schemas}, "paths": paths}
}

const swaggerHTML = `<!doctype html>
<html><head><title>Alauda API</title><link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css"></head>
<body><div id="swagger-ui"></div><script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>window.onload=()=>SwaggerUIBundle({url:'/openapi.json',dom_id:'#swagger-ui'});</script></body></html>`
