// Package provider is the provider part of the Steward SDK for Go: the plumbing between the provider
// contract (gRPC, see ../../proto) and the functions a provider writes.
//
//	p := provider.New(provider.Options{Name: "github", Version: "0.1.0"})
//	p.Service("repository", provider.ServiceOptions{Provision: provision, Deprovision: deprovision})
//	p.Serve()
//
// A provider declares itself and each of its service or application kinds once, with the handlers
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
// its own: the services and applications it offers declare theirs.
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

	// What can be given on the provider as a whole, and the handlers that give it. An entitlement declares the inputs it wants.
	Entitlements []*EntitlementDefinition

	GrantAccess  func(context.Context, *ProviderGrantAccessRequest) (*ProviderGrantAccessResponse, error)
	RevokeAccess func(context.Context, *ProviderRevokeAccessRequest) (*ProviderRevokeAccessResponse, error)
	GetAccess    func(context.Context, *ProviderGetAccessRequest) (*ProviderGetAccessResponse, error)
}

// ServiceOptions describe a kind of service the provider manages, with its handlers. A call
// for something without a handler fails with UNIMPLEMENTED.
type ServiceOptions struct {
	Title       string
	Description string

	// Inputs is what a service of this kind takes besides its name, Secrets the secrets it takes, and
	// Outputs what it produces.
	Inputs  []*InputDefinition
	Secrets []*SecretDefinition
	Outputs []*OutputDefinition

	// Set when the kind takes variables: name and value pairs, of any number, that whoever sets up a service adds to it, whose names
	// the provider does not declare. They arrive in the `Variables` of the calls.
	Variables *VariadicVariables

	// What can be granted on this kind: the tool's roles, the permissions, or both.
	Permissions []*PermissionDefinition
	Roles       []*RoleDefinition

	Provision   func(context.Context, *ServiceProvisionRequest) (*ServiceProvisionResponse, error)
	Deprovision func(context.Context, *ServiceDeprovisionRequest) (*ServiceDeprovisionResponse, error)
	List        func(context.Context, *ServiceListRequest) (*ServiceListResponse, error)

	GrantAccess  func(context.Context, *ServiceGrantAccessRequest) (*ServiceGrantAccessResponse, error)
	RevokeAccess func(context.Context, *ServiceRevokeAccessRequest) (*ServiceRevokeAccessResponse, error)
	GetAccess    func(context.Context, *ServiceGetAccessRequest) (*ServiceGetAccessResponse, error)
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

// Provider is what a provider author builds. Declare its services and applications,
// then call Serve.
type Provider struct {
	options      Options
	services     map[string]ServiceOptions
	applications map[string]ApplicationOptions

	// Declaration order, so the description lists kinds the way the provider declared them.
	serviceOrder     []string
	applicationOrder []string
}

// New creates a provider.
func New(options Options) *Provider {
	return &Provider{
		options:      options,
		services:     map[string]ServiceOptions{},
		applications: map[string]ApplicationOptions{},
	}
}

// Service declares a kind of service the provider manages. The kind is unique within the
// provider, such as "repository".
func (p *Provider) Service(kind string, options ServiceOptions) *Provider {
	if _, declared := p.services[kind]; !declared {
		p.serviceOrder = append(p.serviceOrder, kind)
	}

	p.services[kind] = options

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
		Title:        o.Title,
		Description:  o.Description,
		Entitlements: o.Entitlements,
	}

	for _, name := range p.serviceOrder {
		definition.Services = append(definition.Services, serviceDefinition(name, p.services[name]))
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

	response := &providerv1.DescribeResponse{Name: o.Name, Version: o.Version, Definition: definition}

	// The links are for a credential, so a description that was asked for without one has none.
	if credential != nil && o.SetupLinks != nil {
		response.SetupLinks = o.SetupLinks(ctx, credential)
	}

	return response
}

func serviceDefinition(kind string, options ServiceOptions) *providerv1.ServiceDefinition {
	return &providerv1.ServiceDefinition{
		Kind:        kind,
		Title:       options.Title,
		Description: options.Description,
		Inputs:      options.Inputs,
		Secrets:     options.Secrets,
		Permissions: options.Permissions,
		Roles:       options.Roles,
		Outputs:     options.Outputs,
		Variables:   options.Variables,
	}
}

func (p *Provider) service(kind string) (ServiceOptions, error) {
	found, ok := p.services[kind]
	if !ok {
		return found, status.Error(codes.NotFound, fmt.Sprintf("this provider has no service %q", kind))
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

func unimplemented(call, target string) error {
	return status.Error(codes.Unimplemented, fmt.Sprintf("%s is not implemented for this %s", call, target))
}
