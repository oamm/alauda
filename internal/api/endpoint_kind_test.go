package api

import (
	"connectrpc.com/connect"
	"context"
	"fmt"
	v1 "github.com/company/service-registry/gen/go/api/registry/v1"
	"github.com/company/service-registry/internal/storage"
	"strings"
	"testing"
)

func TestEndpointKindPathWritesAndStructuredDiscovery(t *testing.T) {
	for _, kind := range []string{"tcp", "udp", "grpc"} {
		t.Run(kind, func(t *testing.T) {
			f := newPublicFixture(t)
			invalid := fmt.Sprintf(`{"environment":"stg","instance":{"name":"node","address":"host"},"endpoints":[{"name":"default","kind":%q,"port":81,"path":"/bad"}]}`, kind)
			rejected := f.request("POST", registerRoute, invalid)
			if rejected.Code != 400 || !strings.Contains(rejected.Body.String(), "endpoints[0].path") {
				t.Fatalf("%d %s", rejected.Code, rejected.Body.String())
			}
			valid := strings.Replace(invalid, `,"path":"/bad"`, "", 1)
			out := requireRegistration(t, f.request("POST", registerRoute, valid))
			ep := out.Endpoints[0]
			if ep.Path != "" || strings.HasSuffix(ep.Address, "/") || strings.Contains(ep.Address, ":0") {
				t.Fatalf("%+v", ep)
			}
			handler := &endpointHandler{repo: storage.NewEndpointRepository(f.db)}
			enum := v1.EndpointKind(v1.EndpointKind_value["ENDPOINT_KIND_"+strings.ToUpper(kind)])
			_, err := handler.CreateEndpoint(context.Background(), connect.NewRequest(&v1.CreateEndpointRequest{InstanceId: out.Instance.ID, Name: "invalid", Kind: enum, Port: 81, Path: "/bad"}))
			if connect.CodeOf(err) != connect.CodeInvalidArgument {
				t.Fatalf("create: %v", err)
			}
			_, err = handler.UpdateEndpoint(context.Background(), connect.NewRequest(&v1.UpdateEndpointRequest{Id: ep.ID, Kind: enum, Path: "/bad", Enabled: true}))
			if connect.CodeOf(err) != connect.CodeInvalidArgument {
				t.Fatalf("update: %v", err)
			}
			if _, err = f.db.Exec(context.Background(), `UPDATE endpoints SET path='/legacy' WHERE id=?`, ep.ID); err != nil {
				t.Fatal(err)
			}
			for _, route := range []string{"/api/v1/discovery/Authentication.Grpc/resolve?environment=stg", "/api/v1/services/Authentication.Grpc/instances/node/endpoints?environment=stg"} {
				w := f.request("GET", route, "")
				if w.Code != 200 || strings.Contains(w.Body.String(), `"path"`) || strings.Contains(w.Body.String(), "/legacy") {
					t.Fatalf("legacy output: %d %s", w.Code, w.Body.String())
				}
			}
			updated, err := handler.UpdateEndpoint(context.Background(), connect.NewRequest(&v1.UpdateEndpointRequest{Id: ep.ID, Enabled: true}))
			if err != nil || updated.Msg.Endpoint.Path != "" {
				t.Fatalf("legacy edit: %v %v", updated, err)
			}
		})
	}
}

func TestEndpointKindTransitionClearsPath(t *testing.T) {
	f := newPublicFixture(t)
	out := requireRegistration(t, f.request("POST", registerRoute, initialRegistration))
	handler := &endpointHandler{repo: storage.NewEndpointRepository(f.db)}
	preserved, err := handler.UpdateEndpoint(context.Background(), connect.NewRequest(&v1.UpdateEndpointRequest{Id: out.Endpoints[0].ID, Kind: v1.EndpointKind_ENDPOINT_KIND_HTTPS, Enabled: true}))
	if err != nil || preserved.Msg.Endpoint.Path != "/original" {
		t.Fatalf("omitted HTTP path changed: %v %v", preserved, err)
	}
	updated, err := handler.UpdateEndpoint(context.Background(), connect.NewRequest(&v1.UpdateEndpointRequest{Id: out.Endpoints[0].ID, Kind: v1.EndpointKind_ENDPOINT_KIND_TCP, Port: 5432, Enabled: true}))
	if err != nil || updated.Msg.Endpoint.Path != "" {
		t.Fatalf("transition: %v %v", updated, err)
	}
	// A partial update without kind resolves the stored kind before validating Path.
	_, err = handler.UpdateEndpoint(context.Background(), connect.NewRequest(&v1.UpdateEndpointRequest{Id: out.Endpoints[0].ID, Path: "/invalid", Enabled: true}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("partial update: %v", err)
	}
}
