// Package provider is the provider part of the Steward SDK for Go: the plumbing between the provider
// contract (gRPC, see ../../proto) and the functions a provider writes.
//
//	p := provider.New(provider.Options{Name: "github", Version: "0.1.0"})
//	p.Credential("app", provider.CredentialOptions{Inputs: inputs, Resolve: resolve})
//	p.Resource("repository", provider.ResourceOptions{Provision: provision, Deprovision: deprovision})
//	p.Serve()
//
// A provider declares itself and each of its resource or application kinds once, with the handlers
// attached. The SDK routes every call to the right
// handler by kind, builds the description Steward asks for, serves the gRPC services, prints the
// handshake line, answers the health check, and answers any call without a handler with UNIMPLEMENTED.
//
// Everything a provider logs must go to stderr: stdout is reserved for the handshake line.
//
// Handlers receive and return the generated contract messages. Fail a call with a gRPC status, for
// example status.Error(codes.FailedPrecondition, "..."); any other error is reported as INTERNAL.
package provider

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	providerv1 "github.com/treno-dev/steward-sdk/go/gen/steward/provider/v1"
)

// Options declare the provider and, when it supports access as a whole, the handlers for that. A provider has no settings or secrets of
// its own: the credentials, resources and applications it offers declare theirs.
type Options struct {
	Name        string
	Version     string
	Title       string
	Description string

	// SetupLinks says what the person does in another tool before the settings of a credential,
	// such as a page that creates a role, as links with the address already filled in. It is given
	// the credential when a runner installs it, and nothing when the description is read for the
	// docs, so it has to answer for a credential that is nil.
	SetupLinks func(context.Context, *Credential) []*SetupLink

	// What can be given on the provider as a whole, and the handlers that give it. A grant declares the inputs it wants.
	Grants []*GrantDefinition

	GrantAccess  func(context.Context, *ProviderGrantAccessRequest) (*ProviderGrantAccessResponse, error)
	RevokeAccess func(context.Context, *ProviderRevokeAccessRequest) (*ProviderRevokeAccessResponse, error)
	GetAccess    func(context.Context, *ProviderGetAccessRequest) (*ProviderGetAccessResponse, error)
}

// ResourceOptions describe a kind of resource the provider manages, with its handlers. A call
// for something without a handler fails with UNIMPLEMENTED.
type ResourceOptions struct {
	Title       string
	Description string

	// Inputs is what a resource of this kind takes besides its name, Secrets the secrets it takes, and
	// Outputs what it produces.
	Inputs  []*InputDefinition
	Secrets []*SecretDefinition
	Outputs []*OutputDefinition

	// What can be granted on this kind: the tool's roles, the permissions, or both.
	Permissions []*PermissionDefinition
	Roles       []*RoleDefinition

	Provision   func(context.Context, *ResourceProvisionRequest) (*ResourceProvisionResponse, error)
	Deprovision func(context.Context, *ResourceDeprovisionRequest) (*ResourceDeprovisionResponse, error)
	List        func(context.Context, *ResourceListRequest) (*ResourceListResponse, error)

	GrantAccess  func(context.Context, *ResourceGrantAccessRequest) (*ResourceGrantAccessResponse, error)
	RevokeAccess func(context.Context, *ResourceRevokeAccessRequest) (*ResourceRevokeAccessResponse, error)
	GetAccess    func(context.Context, *ResourceGetAccessRequest) (*ResourceGetAccessResponse, error)
}

// ApplicationOptions describe a kind of application the provider runs, with its handlers.
type ApplicationOptions struct {
	Title       string
	Description string

	Inputs  []*InputDefinition
	Secrets []*SecretDefinition
	Outputs []*OutputDefinition

	// Sources are the source types it can be deployed from: "github", "registry", "s3" or "raw".
	Sources []string

	Create  func(context.Context, *ApplicationCreateRequest) (*ApplicationCreateResponse, error)
	Destroy func(context.Context, *ApplicationDestroyRequest) (*ApplicationDestroyResponse, error)
	Deploy  func(context.Context, *ApplicationDeployRequest) (*ApplicationDeployResponse, error)
}

// CredentialOptions describe a kind of credential, a way of reaching the tool such as a role that is assumed with the identity token of
// Steward. A credential is an instance of a kind. The runner calls Resolve before a task invokes any provider, and hands what it returns
// to the provider as environment variables and files. A value is a secret of the call: do not persist it or log it.
type CredentialOptions struct {
	Title       string
	Description string

	// Inputs are the settings of a credential of this kind, and Secrets its secrets, which are references to a vault that the
	// runner resolves for the call.
	Inputs  []*InputDefinition
	Secrets []*SecretDefinition

	// IdentityAudience is the audience of the token that proves who the credential is, such as "sts.amazonaws.com". A kind
	// that declares it is sent the token in the request.
	IdentityAudience string

	Resolve func(context.Context, *CredentialResolveRequest) (*CredentialResolveResponse, error)
}

// Provider is what a provider author builds. Declare its resources, applications and credentials,
// then call Serve.
type Provider struct {
	options      Options
	resources    map[string]ResourceOptions
	applications map[string]ApplicationOptions
	credentials  map[string]CredentialOptions

	// Declaration order, so the description lists kinds the way the provider declared them.
	resourceOrder    []string
	applicationOrder []string
	credentialOrder  []string
}

// New creates a provider.
func New(options Options) *Provider {
	return &Provider{
		options:      options,
		resources:    map[string]ResourceOptions{},
		applications: map[string]ApplicationOptions{},
		credentials:  map[string]CredentialOptions{},
	}
}

// Credential declares a kind of credential, a way of reaching the tool. The name is unique within the provider, such as "oidc-role".
func (p *Provider) Credential(name string, options CredentialOptions) *Provider {
	if _, declared := p.credentials[name]; !declared {
		p.credentialOrder = append(p.credentialOrder, name)
	}

	p.credentials[name] = options

	return p
}

// Resource declares a kind of resource the provider manages. The kind is unique within the
// provider, such as "repository".
func (p *Provider) Resource(kind string, options ResourceOptions) *Provider {
	if _, declared := p.resources[kind]; !declared {
		p.resourceOrder = append(p.resourceOrder, kind)
	}

	p.resources[kind] = options

	return p
}

// Application declares a kind of application the provider runs. The kind is unique within the
// provider, such as "laravel".
func (p *Provider) Application(kind string, options ApplicationOptions) *Provider {
	if _, declared := p.applications[kind]; !declared {
		p.applicationOrder = append(p.applicationOrder, kind)
	}

	p.applications[kind] = options

	return p
}

func (p *Provider) describe(ctx context.Context, credential *Credential) *providerv1.DescribeResponse {
	o := p.options
	definition := &providerv1.ProviderDefinition{
		Title:       o.Title,
		Description: o.Description,
		Grants:      o.Grants,
	}

	for _, name := range p.resourceOrder {
		definition.Resources = append(definition.Resources, resourceDefinition(name, p.resources[name]))
	}

	for _, name := range p.applicationOrder {
		options := p.applications[name]
		definition.Applications = append(definition.Applications, &providerv1.ApplicationDefinition{
			Kind:        name,
			Title:       options.Title,
			Description: options.Description,
			Inputs:      options.Inputs,
			Secrets:     options.Secrets,
			Sources:     options.Sources,
			Outputs:     options.Outputs,
		})
	}

	for _, name := range p.credentialOrder {
		options := p.credentials[name]
		definition.Credentials = append(definition.Credentials, &providerv1.CredentialDefinition{
			Name:             name,
			Title:            options.Title,
			Description:      options.Description,
			Inputs:           options.Inputs,
			Secrets:          options.Secrets,
			IdentityAudience: options.IdentityAudience,
		})
	}

	response := &providerv1.DescribeResponse{Name: o.Name, Version: o.Version, Definition: definition}

	// The links are for a credential, so a description that was asked for without one has none.
	if credential != nil && o.SetupLinks != nil {
		response.SetupLinks = o.SetupLinks(ctx, credential)
	}

	return response
}

func resourceDefinition(kind string, options ResourceOptions) *providerv1.ResourceDefinition {
	return &providerv1.ResourceDefinition{
		Kind:        kind,
		Title:       options.Title,
		Description: options.Description,
		Inputs:      options.Inputs,
		Secrets:     options.Secrets,
		Permissions: options.Permissions,
		Roles:       options.Roles,
		Outputs:     options.Outputs,
	}
}

func (p *Provider) resource(kind string) (ResourceOptions, error) {
	found, ok := p.resources[kind]
	if !ok {
		return found, status.Error(codes.NotFound, fmt.Sprintf("this provider has no resource %q", kind))
	}

	return found, nil
}

func (p *Provider) application(kind string) (ApplicationOptions, error) {
	found, ok := p.applications[kind]
	if !ok {
		return found, status.Error(codes.NotFound, fmt.Sprintf("this provider has no application %q", kind))
	}

	return found, nil
}

func (p *Provider) credential(name string) (CredentialOptions, error) {
	found, ok := p.credentials[name]
	if !ok {
		return found, status.Error(codes.NotFound, fmt.Sprintf("this provider has no credential %q", name))
	}

	return found, nil
}

func unimplemented(call, target string) error {
	return status.Error(codes.Unimplemented, fmt.Sprintf("%s is not implemented for this %s", call, target))
}
