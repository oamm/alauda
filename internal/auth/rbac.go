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
	ScopeRead            Scope = "read"
	ScopeWrite           Scope = "write"
	ScopeAdmin           Scope = "admin"
	ScopeDiscoveryRead   Scope = "discovery.read"
	ScopeRegistryRead    Scope = "registry.read"
	ScopeRegistryWrite   Scope = "registry.write"
	ScopeHealthRead      Scope = "health.read"
	ScopeHealthWrite     Scope = "health.write"
	ScopeHealthExecute   Scope = "health.execute"
	ScopeIncidentRead    Scope = "incident.read"
	ScopeIncidentResolve Scope = "incident.resolve"
	ScopeEventsRead      Scope = "events.read"
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
		if scope == ScopeRead {
			switch required {
			case ScopeDiscoveryRead, ScopeRegistryRead, ScopeHealthRead, ScopeIncidentRead, ScopeEventsRead:
				return true
			}
		}
		if scope == ScopeWrite {
			switch required {
			case ScopeRegistryWrite, ScopeHealthWrite, ScopeHealthExecute, ScopeIncidentResolve:
				return true
			}
		}
	}
	return false
}
