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
	Resource          = providerv1.Resource
	Application       = providerv1.Application
	ApplicationSource = providerv1.ApplicationSource
	Variable          = providerv1.Variable
	Identity          = providerv1.Identity
	Role              = providerv1.Role
	Permission        = providerv1.Permission

	InputDefinition      = providerv1.InputDefinition
	SecretDefinition     = providerv1.SecretDefinition
	OutputDefinition     = providerv1.OutputDefinition
	PermissionDefinition = providerv1.PermissionDefinition
	RoleDefinition       = providerv1.RoleDefinition

	CredentialDefinition = providerv1.CredentialDefinition
	CredentialFile       = providerv1.CredentialFile
	GrantDefinition      = providerv1.GrantDefinition

	ProviderGrantAccessRequest   = providerv1.ProviderServiceGrantAccessRequest
	ProviderGrantAccessResponse  = providerv1.ProviderServiceGrantAccessResponse
	ProviderRevokeAccessRequest  = providerv1.ProviderServiceRevokeAccessRequest
	ProviderRevokeAccessResponse = providerv1.ProviderServiceRevokeAccessResponse
	ProviderGetAccessRequest     = providerv1.ProviderServiceGetAccessRequest
	ProviderGetAccessResponse    = providerv1.ProviderServiceGetAccessResponse

	ResourceProvisionRequest     = providerv1.ResourceServiceProvisionRequest
	ResourceProvisionResponse    = providerv1.ResourceServiceProvisionResponse
	ResourceDeprovisionRequest   = providerv1.ResourceServiceDeprovisionRequest
	ResourceDeprovisionResponse  = providerv1.ResourceServiceDeprovisionResponse
	ResourceListRequest          = providerv1.ResourceServiceListRequest
	ResourceListResponse         = providerv1.ResourceServiceListResponse
	ResourceGrantAccessRequest   = providerv1.ResourceServiceGrantAccessRequest
	ResourceGrantAccessResponse  = providerv1.ResourceServiceGrantAccessResponse
	ResourceRevokeAccessRequest  = providerv1.ResourceServiceRevokeAccessRequest
	ResourceRevokeAccessResponse = providerv1.ResourceServiceRevokeAccessResponse
	ResourceGetAccessRequest     = providerv1.ResourceServiceGetAccessRequest
	ResourceGetAccessResponse    = providerv1.ResourceServiceGetAccessResponse

	ApplicationCreateRequest   = providerv1.ApplicationServiceCreateRequest
	ApplicationCreateResponse  = providerv1.ApplicationServiceCreateResponse
	ApplicationDestroyRequest  = providerv1.ApplicationServiceDestroyRequest
	ApplicationDestroyResponse = providerv1.ApplicationServiceDestroyResponse
	ApplicationDeployRequest   = providerv1.ApplicationServiceDeployRequest
	ApplicationDeployResponse  = providerv1.ApplicationServiceDeployResponse

	CredentialResolveRequest  = providerv1.CredentialServiceResolveRequest
	CredentialResolveResponse = providerv1.CredentialServiceResolveResponse
)

// The types an InputDefinition or OutputDefinition can have.
const (
	TypeString  = "string"
	TypeNumber  = "number"
	TypeBoolean = "boolean"
	TypeSelect  = "select"
	TypeList    = "list"
	TypeMap     = "map"
)
