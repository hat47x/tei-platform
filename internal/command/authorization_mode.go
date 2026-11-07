package command

// AuthorizationMode names how the Command boundary reached its authorization
// decision. It is evidence about which check ran, not a grant of authority.
type AuthorizationMode string

const (
	// AuthorizationReferenceAllowAll means the composition explicitly chose
	// ReferenceAuthorizer: every request is allowed and no subject, rule or
	// resource was evaluated.
	AuthorizationReferenceAllowAll AuthorizationMode = "reference-allow-all"
	// AuthorizationAllowList means AllowListAuthorizer ran: only the named
	// capabilities were allowed. It is still a reference decision, not a
	// permission source.
	AuthorizationAllowList AuthorizationMode = "reference-allow-list"
	// AuthorizationExternal means a non-reference Authorizer decided. The
	// boundary cannot tell which rules or grants it consulted.
	AuthorizationExternal AuthorizationMode = "external"
)

// authorizationModeOf reports the mode of an Authorizer the Invoker was given.
// It is a type switch over the two reference authorizers this package defines;
// any other implementation is called external without claiming more.
func authorizationModeOf(authorizer Authorizer) AuthorizationMode {
	switch authorizer.(type) {
	case ReferenceAuthorizer:
		return AuthorizationReferenceAllowAll
	case AllowListAuthorizer:
		return AuthorizationAllowList
	default:
		return AuthorizationExternal
	}
}
