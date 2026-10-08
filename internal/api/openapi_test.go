package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestOpenAPIValidatesAndExamplesMatch(t *testing.T) {
	data, err := json.Marshal(openAPISpec)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := openapi3.NewLoader().LoadFromData(data)
	if err != nil {
		t.Fatal(err)
	}
	if err = doc.Validate(context.Background()); err != nil {
		t.Fatal(err)
	}
	for name, ref := range doc.Components.Schemas {
		if ref.Value.Example != nil {
			data := mustEncode(t, ref.Value.Example)
			var example any
			if json.Unmarshal(data, &example) != nil {
				t.Fatal(name)
			}
			if err = ref.Value.VisitJSON(example); err != nil {
				t.Fatalf("%s example: %v", name, err)
			}
		}
	}
	reg := doc.Components.Schemas["Registration"].Value
	if err = reg.VisitJSON(reg.Example); err != nil {
		t.Fatalf("registration example: %v", err)
	}
	var dto publicRegistrationRequest
	if err = json.Unmarshal(mustEncode(t, reg.Example), &dto); err != nil {
		t.Fatal(err)
	}
	f := newPublicFixture(t)
	w := f.request("POST", registerRoute, initialRegistration)
	var response any
	if err = json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if err = doc.Components.Schemas["RegistrationResponse"].Value.VisitJSON(response); err != nil {
		t.Fatal(err)
	}
	requireRegistration(t, f.request("POST", registerRoute, `{"environment":"stg","instance":{"name":"auth-01","enabled":true},"endpoints":[{"name":"default","enabled":true}]}`))
	w = f.request("GET", "/api/v1/discovery/Authentication.Grpc/resolve?environment=stg", "")
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	response = nil
	if err = json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if err = doc.Components.Schemas["Resolve"].Value.VisitJSON(response); err != nil {
		t.Fatal(err)
	}
	if doc.Paths.Value("/api/v1/health/status") == nil {
		t.Fatal("missing canonical health status route")
	}
	w = f.request("GET", "/api/v1/health/status?environment=stg", "")
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	response = nil
	if err = json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if err = doc.Components.Schemas["HealthStatusSnapshot"].Value.VisitJSON(response); err != nil {
		t.Fatal(err)
	}
	if doc.Paths.Value("/api/v1/services/{serviceKey}/instances/{instanceName}").Delete.Responses.Value("204") == nil {
		t.Fatal("missing 204")
	}
	if doc.Paths.Value("/api/v1/services/{serviceKey}/instances").Post.Responses.Value("409") == nil {
		t.Fatal("missing 409")
	}
	for path, item := range doc.Paths.Map() {
		actual := strings.NewReplacer("{serviceKey}", "Authentication.Grpc", "{instanceName}", "auth-01", "{environmentKey}", "stg", "{serviceName}", "Authentication.Grpc", "{checkName}", "missing").Replace(path)
		for method, op := range item.Operations() {
			if op == nil {
				continue
			}
			if method == http.MethodGet {
				res := f.request(method, actual+"?environment=stg", "")
				if res.Code == 405 {
					t.Fatalf("unmounted operation %s %s", method, path)
				}
			}
		}
	}
}
func mustEncode(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
