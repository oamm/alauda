package api

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/company/service-registry/internal/auth"
	"github.com/company/service-registry/internal/problem"
	"github.com/company/service-registry/internal/storage"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

func environmentAuthorization(db *storage.Database, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := auth.PrincipalFromContext(r.Context())
		if !ok || len(p.EnvironmentIDs) == 0 {
			next.ServeHTTP(w, r)
			return
		}
		r = r.WithContext(storage.WithEnvironmentAccess(r.Context(), p.EnvironmentIDs))
		deny := func() {
			problem.Write(w, 403, "permission_denied", "Credential is not authorized for this resource or requires an explicit environment filter.", nil)
		}
		path := r.URL.Path
		if strings.HasPrefix(path, "/api/v1/auth/") {
			if path == "/api/v1/auth/me" || path == "/api/v1/auth/logout" || path == "/api/v1/auth/password" {
				next.ServeHTTP(w, r)
			} else {
				deny()
			}
			return
		}
		if strings.Contains(path, "AlertService/") || strings.HasPrefix(path, "/api/v1/alerts/") || strings.HasPrefix(path, "/api/v1/audit-logs") {
			deny()
			return
		}
		values := map[string]string{}
		for key, items := range r.URL.Query() {
			if len(items) > 0 {
				values[key] = items[0]
			}
		}
		operation := path[strings.LastIndex(path, "/")+1:]
		if r.Body != nil && r.Method == http.MethodPost {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				problem.Write(w, 413, "request_too_large", "Request body exceeds the configured limit.", nil)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			var fields map[string]any
			if strings.HasPrefix(path, "/registry.v1.") && (strings.Contains(r.Header.Get("Content-Type"), "grpc") || strings.Contains(r.Header.Get("Content-Type"), "connect+")) {
				if len(body) < 5 || body[0] != 0 || uint64(binary.BigEndian.Uint32(body[1:5])) != uint64(len(body)-5) {
					problem.Write(w, 400, "validation_failed", "Invalid or unsupported request frame.", nil)
					return
				}
				body = body[5:]
			}
			if strings.HasPrefix(path, "/registry.v1.") && !strings.Contains(r.Header.Get("Content-Type"), "json") {
				mt, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName("registry.v1." + operation + "Request"))
				if err != nil {
					deny()
					return
				}
				msg := mt.New().Interface()
				if proto.Unmarshal(body, msg) != nil {
					problem.Write(w, 400, "validation_failed", "Invalid request body.", nil)
					return
				}
				body, _ = protojson.Marshal(msg)
			}
			if len(body) > 0 && json.Unmarshal(body, &fields) != nil {
				problem.Write(w, 400, "validation_failed", "Invalid request body.", nil)
				return
			}
			for key, value := range fields {
				if v, ok := value.(string); ok {
					values[key] = v
				}
			}
		}
		// Global mutations cannot be restricted to one environment safely.
		if strings.Contains(path, "CatalogService/") && !(strings.HasPrefix(operation, "Get") || strings.HasPrefix(operation, "List")) || strings.Contains(path, "EnvironmentService/Create") || (path == "/api/v1/services" || path == "/api/v1/environments") && r.Method != http.MethodGet {
			deny()
			return
		}
		checked := false
		check := func(query string, arg string) bool {
			if arg == "" {
				return true
			}
			checked = true
			var env string
			err := db.QueryRow(r.Context(), query, arg).Scan(&env)
			if errors.Is(err, sql.ErrNoRows) {
				problem.Write(w, 404, "resource_not_found", "Resource does not exist.", nil)
				return false
			}
			if err != nil {
				problem.Write(w, 500, "server_error", "Unable to authorize resource.", nil)
				return false
			}
			if !environmentAllowed(r, env) {
				deny()
				return false
			}
			return true
		}
		joins := "SELECT d.environment_id FROM service_instances i JOIN service_deployments d ON d.id=i.deployment_id "
		rules := []struct {
			keys  []string
			query string
		}{
			{[]string{"environmentId", "environment_id"}, "SELECT id FROM environments WHERE id=? AND deleted_at IS NULL"},
			{[]string{"environment", "environmentKey", "environment_key"}, "SELECT id FROM environments WHERE key=? AND deleted_at IS NULL"},
			{[]string{"deploymentId", "deployment_id"}, "SELECT environment_id FROM service_deployments WHERE id=?"},
			{[]string{"instanceId", "instance_id"}, joins + "WHERE i.id=?"},
			{[]string{"endpointId", "endpoint_id"}, joins + "JOIN endpoints e ON e.instance_id=i.id WHERE e.id=?"},
			{[]string{"healthCheckId", "health_check_id", "checkId"}, joins + "JOIN health_checks h ON h.instance_id=i.id WHERE h.id=?"},
		}
		if strings.HasPrefix(path, "/api/v1/environments/") {
			parts := resourcePathParts(path, "/api/v1/environments/")
			if len(parts) > 0 {
				values["environment"] = parts[0]
			}
		}
		for _, rule := range rules {
			for _, key := range rule.keys {
				if !check(rule.query, values[key]) {
					return
				}
			}
		}
		if id := values["id"]; id != "" {
			var query string
			switch {
			case strings.Contains(path, "EnvironmentService/"):
				query = "SELECT id FROM environments WHERE id=?"
			case strings.Contains(path, "DeploymentService/"):
				query = "SELECT environment_id FROM service_deployments WHERE id=?"
			case strings.Contains(path, "InstanceService/"):
				query = joins + "WHERE i.id=?"
			case strings.Contains(path, "EndpointService/"):
				query = joins + "JOIN endpoints e ON e.instance_id=i.id WHERE e.id=?"
			case strings.Contains(path, "HealthService/"):
				query = joins + "JOIN health_checks h ON h.instance_id=i.id WHERE h.id=?"
			case strings.Contains(path, "IncidentService/"):
				query = joins + "JOIN incidents incident ON incident.instance_id=i.id WHERE incident.id=?"
			case strings.Contains(path, "CatalogService/"):
				var count int
				args := []any{id}
				q := "SELECT COUNT(*) FROM service_deployments WHERE service_id=? AND deleted_at IS NULL AND environment_id IN ("
				for i, env := range p.EnvironmentIDs {
					if i > 0 {
						q += ","
					}
					q += "?"
					args = append(args, env)
				}
				if err := db.QueryRow(r.Context(), q+")", args...).Scan(&count); err != nil {
					problem.Write(w, 500, "server_error", "Unable to authorize resource.", nil)
					return
				}
				if count == 0 {
					deny()
					return
				}
				checked = true
			}
			if query != "" && !check(query, id) {
				return
			}
		}
		safeCatalog := path == "/api/v1/services" || path == "/api/v1/environments" || path == "/api/v1/catalog/environments" || path == "/api/v1/catalog/services" || strings.Contains(path, "EnvironmentService/List") || strings.Contains(path, "CatalogService/List")
		publicResource := strings.HasPrefix(path, "/api/v1/services/") || strings.HasPrefix(path, "/api/v1/discovery/")
		healthProjection := path == "/api/v1/health/checks" || path == "/api/v1/health/results"
		if !checked && !safeCatalog && !publicResource && !healthProjection && path != "/openapi.json" && path != "/swagger" {
			deny()
			return
		}
		next.ServeHTTP(w, r)
	})
}
