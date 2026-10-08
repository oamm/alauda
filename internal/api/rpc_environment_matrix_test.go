package api

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	v1 "github.com/company/service-registry/gen/go/api/registry/v1"
	"github.com/company/service-registry/internal/auth"
	"github.com/company/service-registry/internal/storage"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

func TestAllRPCResourceReferencesEnforceEnvironment(t *testing.T) {
	f := newPublicFixture(t)
	out := requireRegistration(t, f.request("POST", registerRoute, `{"environment":"stg","instance":{"name":"auth-01","address":"host"},"endpoints":[{"name":"default","protocol":"http","port":81}]}`))
	var deploymentID string
	if err := f.db.QueryRow(context.Background(), "SELECT deployment_id FROM service_instances WHERE id=?", out.Instance.ID).Scan(&deploymentID); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := f.db.Exec(context.Background(), `INSERT INTO health_checks(id,instance_id,endpoint_id,name,type,enabled,interval_seconds,timeout_seconds,failures_before_unhealthy,successes_before_healthy,tags,metadata,created_at,updated_at) VALUES('matrix-check',?,?,'check',1,1,30,5,3,2,'{}','{}',?,?)`, out.Instance.ID, out.Endpoints[0].ID, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(context.Background(), `INSERT INTO incidents(id,instance_id,deployment_id,environment_id,service_id,state,opened_at,reason,created_at) VALUES('matrix-incident',?,?,?,?,'OPEN',?,'test',?)`, out.Instance.ID, deploymentID, f.environmentID, f.serviceID, now, now); err != nil {
		t.Fatal(err)
	}
	env, err := storage.NewEnvironmentRepository(f.db).Create(context.Background(), &v1.CreateEnvironmentRequest{Key: "only-allowed", Name: "Allowed"})
	if err != nil {
		t.Fatal(err)
	}
	key, err := auth.NewRepository(f.db).CreateApplicationKey(context.Background(), auth.CreateApplicationKeyInput{Name: "matrix", Scopes: []auth.Scope{auth.ScopeAdmin}, EnvironmentIDs: []string{env.Id}, CreatedBy: "test"})
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]string{"environment_id": f.environmentID, "environment_key": "stg", "service_id": f.serviceID, "deployment_id": deploymentID, "instance_id": out.Instance.ID, "endpoint_id": out.Endpoints[0].ID, "health_check_id": "matrix-check"}
	tested := 0
	protoregistry.GlobalFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		if file.Package() != "registry.v1" {
			return true
		}
		services := file.Services()
		for i := 0; i < services.Len(); i++ {
			svc := services.Get(i)
			methods := svc.Methods()
			for j := 0; j < methods.Len(); j++ {
				method := methods.Get(j)
				msg := dynamicpb.NewMessage(method.Input())
				referenced := false
				fields := method.Input().Fields()
				for k := 0; k < fields.Len(); k++ {
					field := fields.Get(k)
					if field.Kind() != protoreflect.StringKind || field.IsList() {
						continue
					}
					value := ids[string(field.Name())]
					if field.Name() == "id" {
						switch string(svc.Name()) {
						case "EnvironmentService":
							value = f.environmentID
						case "CatalogService":
							value = f.serviceID
						case "DeploymentService":
							value = deploymentID
						case "InstanceService":
							value = out.Instance.ID
						case "EndpointService":
							value = out.Endpoints[0].ID
						case "HealthService":
							value = "matrix-check"
						case "IncidentService":
							value = "matrix-incident"
						}
					}
					if value != "" {
						msg.Set(field, protoreflect.ValueOfString(value))
						referenced = true
					}
				}
				if !referenced {
					continue
				}
				path := "/" + string(svc.FullName()) + "/" + string(method.Name())
				for _, typ := range []string{"application/json", "application/proto", "application/connect+json", "application/connect+proto"} {
					var body []byte
					var err error
					if strings.HasSuffix(typ, "proto") {
						body, err = proto.Marshal(msg)
					} else {
						data := map[string]string{}
						msg.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
							data[field.JSONName()] = value.String()
							return true
						})
						body, err = json.Marshal(data)
					}
					if err != nil {
						t.Fatal(err)
					}
					if strings.Contains(typ, "connect+") {
						frame := make([]byte, 5, len(body)+5)
						binary.BigEndian.PutUint32(frame[1:], uint32(len(body)))
						body = append(frame, body...)
					}
					req := httptest.NewRequest("POST", path, bytes.NewReader(body))
					req.Header.Set("Content-Type", typ)
					req.Header.Set("Authorization", "Bearer "+key.Secret)
					res := httptest.NewRecorder()
					f.mux.ServeHTTP(res, req)
					if res.Code != 403 {
						t.Errorf("%s %s bypass: %d %s", path, typ, res.Code, res.Body.String())
					}
					tested++
				}
			}
		}
		return true
	})
	if tested < 40 {
		t.Fatalf("matrix covered only %d transport cases", tested)
	}
	t.Logf("Environment boundary denied %d resource-reference transport cases", tested)
}

func TestScopedConnectJSONFramePreservesAllowedRequest(t *testing.T) {
	f := newPublicFixture(t)
	payload := []byte(`{"environmentId":"` + f.environmentID + `"}`)
	frame := make([]byte, 5, len(payload)+5)
	binary.BigEndian.PutUint32(frame[1:], uint32(len(payload)))
	frame = append(frame, payload...)
	req := httptest.NewRequest("POST", "/registry.v1.EventService/WatchEvents", bytes.NewReader(frame))
	req.Header.Set("Content-Type", "application/connect+json")
	req = req.WithContext(auth.WithPrincipal(req.Context(), &auth.Principal{EnvironmentIDs: []string{f.environmentID}}))
	called := false
	handler := environmentAuthorization(f.db, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		body, err := io.ReadAll(r.Body)
		if err != nil || !bytes.Equal(body, frame) {
			t.Fatalf("request frame changed: %q, %v", body, err)
		}
		w.WriteHeader(200)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if !called || response.Code != 200 {
		t.Fatalf("allowed streaming frame rejected: %d %s", response.Code, response.Body.String())
	}
}
