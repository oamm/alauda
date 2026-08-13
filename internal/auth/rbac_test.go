package auth

import "testing"

func TestRoleScopesAndValidation(t *testing.T) {
	tests := []struct {
		role   Role
		scopes []Scope
		valid  bool
	}{
		{RoleAdministrator, []Scope{ScopeRead, ScopeWrite, ScopeAdmin}, true},
		{RoleOperator, []Scope{ScopeRead, ScopeWrite}, true},
		{RoleAutomation, []Scope{ScopeRead, ScopeWrite}, true},
		{RoleViewer, []Scope{ScopeRead}, true},
		{Role("Other"), nil, false},
	}
	for _, tc := range tests {
		t.Run(string(tc.role), func(t *testing.T) {
			if got := ValidRole(tc.role); got != tc.valid {
				t.Fatalf("ValidRole() = %v, want %v", got, tc.valid)
			}
			got := RoleScopes(tc.role)
			if len(got) != len(tc.scopes) {
				t.Fatalf("RoleScopes() = %v, want %v", got, tc.scopes)
			}
			for i := range got {
				if got[i] != tc.scopes[i] {
					t.Fatalf("RoleScopes() = %v, want %v", got, tc.scopes)
				}
			}
		})
	}
}

func TestHasScopeAllowsAdminAndExactScope(t *testing.T) {
	if !HasScope([]Scope{ScopeRead}, ScopeRead) {
		t.Fatalf("expected exact scope to satisfy requirement")
	}
	if !HasScope([]Scope{ScopeAdmin}, ScopeWrite) {
		t.Fatalf("expected admin scope to satisfy write requirement")
	}
	if HasScope([]Scope{ScopeRead}, ScopeWrite) {
		t.Fatalf("read scope should not satisfy write requirement")
	}
}
