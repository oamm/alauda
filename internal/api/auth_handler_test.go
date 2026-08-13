package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	"github.com/company/service-registry/internal/auth"
	"github.com/company/service-registry/internal/storage"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestAuthLoginRouteReturnsToken(t *testing.T) {
	ctx := context.Background()
	db := newAPITestDB(t, ctx)
	repo := auth.NewRepository(db)
	_, err := repo.CreateUser(ctx, auth.CreateUserInput{
		Username:    "admin",
		Email:       "admin@example.test",
		DisplayName: "Admin",
		Password:    "password",
		Role:        auth.RoleAdministrator,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	mux := http.NewServeMux()
	registerAuthREST(mux, authHandler{service: auth.NewService(repo, time.Hour), repo: repo})

	body := bytes.NewBufferString(`{"username":"admin","password":"password"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var response struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Token == "" {
		t.Fatalf("expected token in response")
	}
}

func TestAuthLoginRouteRejectsBadRequests(t *testing.T) {
	ctx := context.Background()
	db := newAPITestDB(t, ctx)
	repo := auth.NewRepository(db)
	mux := http.NewServeMux()
	registerAuthREST(mux, authHandler{service: auth.NewService(repo, time.Hour), repo: repo})

	tests := []struct {
		name   string
		method string
		body   string
		want   int
	}{
		{name: "method", method: http.MethodGet, body: `{}`, want: http.StatusMethodNotAllowed},
		{name: "json", method: http.MethodPost, body: `{`, want: http.StatusBadRequest},
		{name: "credentials", method: http.MethodPost, body: `{"username":"missing","password":"wrong"}`, want: http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/v1/auth/login", bytes.NewBufferString(tt.body))
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d: %s", rec.Code, tt.want, rec.Body.String())
			}
		})
	}
}

func TestAuthRESTUserAndTokenManagement(t *testing.T) {
	ctx := context.Background()
	db := newAPITestDB(t, ctx)
	repo := auth.NewRepository(db)
	service := auth.NewService(repo, time.Hour)
	admin, err := repo.CreateUser(ctx, auth.CreateUserInput{
		Username:    "admin",
		Email:       "admin@example.test",
		DisplayName: "Admin",
		Password:    "password",
		Role:        auth.RoleAdministrator,
	})
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	mux := http.NewServeMux()
	registerAuthREST(mux, authHandler{service: service, repo: repo})

	principal := &auth.Principal{
		UserID:   admin.ID,
		Username: admin.Username,
		Role:     admin.Role,
		Scopes:   auth.RoleScopes(admin.Role),
	}
	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil).WithContext(auth.WithPrincipal(ctx, principal))
	meRec := httptest.NewRecorder()
	mux.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf("me status = %d, want %d: %s", meRec.Code, http.StatusOK, meRec.Body.String())
	}

	createUserBody := bytes.NewBufferString(`{"username":"ops","email":"ops@example.test","displayName":"Ops","password":"password","role":"Operator","tags":{"team":"sre"}}`)
	createUserReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/users", createUserBody)
	createUserRec := httptest.NewRecorder()
	mux.ServeHTTP(createUserRec, createUserReq)
	if createUserRec.Code != http.StatusCreated {
		t.Fatalf("create user status = %d, want %d: %s", createUserRec.Code, http.StatusCreated, createUserRec.Body.String())
	}

	listUsersReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/users", nil)
	listUsersRec := httptest.NewRecorder()
	mux.ServeHTTP(listUsersRec, listUsersReq)
	if listUsersRec.Code != http.StatusOK {
		t.Fatalf("list users status = %d, want %d: %s", listUsersRec.Code, http.StatusOK, listUsersRec.Body.String())
	}
	var listed struct {
		Users []auth.User `json:"users"`
	}
	if err := json.NewDecoder(listUsersRec.Body).Decode(&listed); err != nil {
		t.Fatalf("decode users: %v", err)
	}
	if len(listed.Users) != 2 {
		t.Fatalf("users = %d, want 2", len(listed.Users))
	}

	tokenBody := bytes.NewBufferString(`{"userId":"` + admin.ID + `","name":"ci","scopes":["read"],"environmentIds":["prod"]}`)
	tokenReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/tokens", tokenBody).WithContext(auth.WithPrincipal(ctx, principal))
	tokenRec := httptest.NewRecorder()
	mux.ServeHTTP(tokenRec, tokenReq)
	if tokenRec.Code != http.StatusCreated {
		t.Fatalf("create token status = %d, want %d: %s", tokenRec.Code, http.StatusCreated, tokenRec.Body.String())
	}
	var created auth.CreatedToken
	if err := json.NewDecoder(tokenRec.Body).Decode(&created); err != nil {
		t.Fatalf("decode token: %v", err)
	}
	if created.Secret == "" || created.Token.ID == "" {
		t.Fatalf("expected token secret and id, got %#v", created)
	}

	listTokenReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/tokens", nil).WithContext(auth.WithPrincipal(ctx, principal))
	listTokenRec := httptest.NewRecorder()
	mux.ServeHTTP(listTokenRec, listTokenReq)
	if listTokenRec.Code != http.StatusOK {
		t.Fatalf("list tokens status = %d, want %d: %s", listTokenRec.Code, http.StatusOK, listTokenRec.Body.String())
	}

	revokeReq := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/tokens/"+created.Token.ID, nil)
	revokeRec := httptest.NewRecorder()
	mux.ServeHTTP(revokeRec, revokeReq)
	if revokeRec.Code != http.StatusNoContent {
		t.Fatalf("revoke status = %d, want %d: %s", revokeRec.Code, http.StatusNoContent, revokeRec.Body.String())
	}
}

func TestEventHandlerListEventsAppliesFiltersAndPagination(t *testing.T) {
	ctx := context.Background()
	db := newAPITestDB(t, ctx)
	repo := storage.NewEventRepository(db)
	env, err := storage.NewEnvironmentRepository(db).Create(ctx, &registryv1.CreateEnvironmentRequest{Key: "prod", Name: "Production"})
	if err != nil {
		t.Fatalf("create environment: %v", err)
	}
	service, err := storage.NewServiceRepository(db).Create(ctx, &registryv1.CreateServiceRequest{Name: "svc", DisplayName: "Service"})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	now := time.Now().UTC()
	events := []*registryv1.Event{
		{Type: "catalog.updated", Timestamp: timestamppb.New(now.Add(-time.Minute)), ResourceType: "service", ResourceId: "svc-1", EnvironmentId: env.GetId(), ServiceId: service.GetId(), Actor: "test", Message: "first", Metadata: map[string]string{}},
		{Type: "catalog.updated", Timestamp: timestamppb.New(now), ResourceType: "service", ResourceId: "svc-2", EnvironmentId: env.GetId(), ServiceId: service.GetId(), Actor: "test", Message: "second", Metadata: map[string]string{}},
		{Type: "health.changed", Timestamp: timestamppb.New(now.Add(time.Minute)), ResourceType: "health", ResourceId: "hc-1", EnvironmentId: env.GetId(), ServiceId: service.GetId(), Actor: "test", Message: "third", Metadata: map[string]string{}},
	}
	for _, event := range events {
		if _, err := repo.Create(ctx, event); err != nil {
			t.Fatalf("create event: %v", err)
		}
	}

	handler := &eventHandler{repo: repo}
	resp, err := handler.ListEvents(ctx, connect.NewRequest(&registryv1.ListEventsRequest{
		EnvironmentId: env.GetId(),
		Type:          "catalog.updated",
		Pagination:    &registryv1.PaginationRequest{PageSize: 1},
	}))
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(resp.Msg.GetEvents()) != 1 || resp.Msg.GetEvents()[0].GetResourceId() != "svc-2" {
		t.Fatalf("events = %+v, want newest prod catalog event", resp.Msg.GetEvents())
	}
	if resp.Msg.GetPagination().GetNextPageToken() == "" {
		t.Fatalf("expected next page token")
	}
}

func newAPITestDB(t *testing.T, ctx context.Context) *storage.Database {
	t.Helper()
	db, err := storage.NewDatabase(ctx, ":memory:")
	if err != nil {
		t.Fatalf("new database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := storage.RunMigrations(ctx, db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	return db
}
