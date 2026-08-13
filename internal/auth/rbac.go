package auth

type Role string

const (
	RoleAdministrator Role = "Administrator"
	RoleOperator      Role = "Operator"
	RoleViewer        Role = "Viewer"
	RoleAutomation    Role = "Automation"
)

type Scope string

const (
	ScopeRead  Scope = "read"
	ScopeWrite Scope = "write"
	ScopeAdmin Scope = "admin"
)

func RoleScopes(role Role) []Scope {
	switch role {
	case RoleAdministrator:
		return []Scope{ScopeRead, ScopeWrite, ScopeAdmin}
	case RoleOperator, RoleAutomation:
		return []Scope{ScopeRead, ScopeWrite}
	case RoleViewer:
		return []Scope{ScopeRead}
	default:
		return nil
	}
}

func ValidRole(role Role) bool {
	switch role {
	case RoleAdministrator, RoleOperator, RoleViewer, RoleAutomation:
		return true
	default:
		return false
	}
}

func HasScope(scopes []Scope, required Scope) bool {
	for _, scope := range scopes {
		if scope == ScopeAdmin || scope == required {
			return true
		}
	}
	return false
}
