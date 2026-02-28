package security

import "github.com/nojyerac/go-lib/authz"

const (
	RoleReader = "flag_reader"
	RoleAdmin  = "flag_admin"
)

func HTTPPolicyMap() authz.PolicyMap {
	policies := authz.NewPolicyMap()

	readRequirement := authz.RequireAny(RoleReader, RoleAdmin)
	adminRequirement := authz.RequireAny(RoleAdmin)

	policies.Set(authz.HTTPOperation("GET", "/api/flags"), readRequirement)
	policies.Set(authz.HTTPOperation("GET", "/api/flags/{id}"), readRequirement)
	policies.Set(authz.HTTPOperation("POST", "/api/flags/{id}/evaluate"), readRequirement)

	policies.Set(authz.HTTPOperation("POST", "/api/flags"), adminRequirement)
	policies.Set(authz.HTTPOperation("PUT", "/api/flags/{id}"), adminRequirement)
	policies.Set(authz.HTTPOperation("DELETE", "/api/flags/{id}"), adminRequirement)

	return policies
}

func GRPCPolicyMap() authz.PolicyMap {
	policies := authz.NewPolicyMap()

	readRequirement := authz.RequireAny(RoleReader, RoleAdmin)
	adminRequirement := authz.RequireAny(RoleAdmin)

	policies.Set(authz.GRPCOperation("/flag.FlagService/GetFlag"), readRequirement)
	policies.Set(authz.GRPCOperation("/flag.FlagService/ListFlags"), readRequirement)
	policies.Set(authz.GRPCOperation("/flag.FlagService/Evaluate"), readRequirement)

	policies.Set(authz.GRPCOperation("/flag.FlagService/CreateFlag"), adminRequirement)
	policies.Set(authz.GRPCOperation("/flag.FlagService/UpdateFlag"), adminRequirement)
	policies.Set(authz.GRPCOperation("/flag.FlagService/DeleteFlag"), adminRequirement)

	return policies
}
