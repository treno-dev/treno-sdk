package provider

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	providerv1 "github.com/treno-dev/steward-sdk/go/gen/steward/provider/v1"
)

// Struct turns a map into the Struct the contract uses for open-ended values, such as a response's
// Outputs or an identity's Attrs. Values must be what JSON can hold (strings, numbers, booleans, nil,
// and maps and slices of those). Anything else is a mistake in the provider, so it panics, and the SDK
// reports a panic in a handler as an INTERNAL error.
func Struct(values map[string]any) *structpb.Struct {
	result, err := structpb.NewStruct(values)
	if err != nil {
		panic(fmt.Sprintf("provider.Struct: %v", err))
	}

	return result
}

// Value turns a Go value into the Value the contract uses for an input's Default. Same rules as
// Struct.
func Value(value any) *structpb.Value {
	result, err := structpb.NewValue(value)
	if err != nil {
		panic(fmt.Sprintf("provider.Value: %v", err))
	}

	return result
}

// The messages a provider works with, from the generated contract, so a provider imports one package.
// Requests and responses are named for what they handle (the entity, then the call).
type (
	Credential        = providerv1.Credential
	Organization      = providerv1.Organization
	SetupLink         = providerv1.SetupLink
	Service           = providerv1.Service
	Application       = providerv1.Application
	ApplicationSource = providerv1.ApplicationSource
	Variable          = providerv1.Variable
	Identity          = providerv1.Identity
	Role              = providerv1.Role
	Permission        = providerv1.Permission

	InputDefinition      = providerv1.InputDefinition
	SecretDefinition     = providerv1.SecretDefinition
	OutputDefinition     = providerv1.OutputDefinition
	VariadicVariables    = providerv1.VariadicVariables
	PermissionDefinition = providerv1.PermissionDefinition
	RoleDefinition       = providerv1.RoleDefinition

	EntitlementDefinition = providerv1.EntitlementDefinition

	ProviderGrantAccessRequest   = providerv1.ProviderServiceGrantAccessRequest
	ProviderGrantAccessResponse  = providerv1.ProviderServiceGrantAccessResponse
	ProviderRevokeAccessRequest  = providerv1.ProviderServiceRevokeAccessRequest
	ProviderRevokeAccessResponse = providerv1.ProviderServiceRevokeAccessResponse
	ProviderGetAccessRequest     = providerv1.ProviderServiceGetAccessRequest
	ProviderGetAccessResponse    = providerv1.ProviderServiceGetAccessResponse

	ServiceProvisionRequest     = providerv1.ServiceServiceProvisionRequest
	ServiceProvisionResponse    = providerv1.ServiceServiceProvisionResponse
	ServiceDeprovisionRequest   = providerv1.ServiceServiceDeprovisionRequest
	ServiceDeprovisionResponse  = providerv1.ServiceServiceDeprovisionResponse
	ServiceListRequest          = providerv1.ServiceServiceListRequest
	ServiceListResponse         = providerv1.ServiceServiceListResponse
	ServiceGrantAccessRequest   = providerv1.ServiceServiceGrantAccessRequest
	ServiceGrantAccessResponse  = providerv1.ServiceServiceGrantAccessResponse
	ServiceRevokeAccessRequest  = providerv1.ServiceServiceRevokeAccessRequest
	ServiceRevokeAccessResponse = providerv1.ServiceServiceRevokeAccessResponse
	ServiceGetAccessRequest     = providerv1.ServiceServiceGetAccessRequest
	ServiceGetAccessResponse    = providerv1.ServiceServiceGetAccessResponse

	ApplicationCreateRequest   = providerv1.ApplicationServiceCreateRequest
	ApplicationCreateResponse  = providerv1.ApplicationServiceCreateResponse
	ApplicationDestroyRequest  = providerv1.ApplicationServiceDestroyRequest
	ApplicationDestroyResponse = providerv1.ApplicationServiceDestroyResponse
	ApplicationDeployRequest   = providerv1.ApplicationServiceDeployRequest
	ApplicationDeployResponse  = providerv1.ApplicationServiceDeployResponse
)

// The types an InputDefinition or OutputDefinition can have.
const (
	TypeString  = "string"
	TypeNumber  = "number"
	TypeBoolean = "boolean"
	TypeSelect  = "select"
	TypeList    = "list"
	TypeMap     = "map"

	// TypeSource is a type that Steward defines: where something is fetched from, as `{ type, config }`. The input lists the source
	// types it accepts as its options, and the console asks for it with the same fields as the source of a provider.
	TypeSource = "steward::source"

	// TypeStringMap is a type that Steward defines: names and values, text to text, that the console asks for as a list of rows, such as
	// the settings of a backend.
	TypeStringMap = "map(string)"

	// TypeJSON is a type that Steward defines: any JSON value, which the console asks for in a dialog that checks it, so that a value that is
	// not a string, such as a map or a list of numbers, can be given.
	TypeJSON = "steward::json"
)
