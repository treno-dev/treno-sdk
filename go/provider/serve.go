package provider

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"

	goplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	providerv1 "github.com/treno-dev/steward-sdk/go/gen/steward/provider/v1"
)

// Serve starts serving the provider and blocks until the runner stops it. It is hashicorp/go-plugin that
// does the work the runner depends on: it listens on a unix socket (TCP on Windows), prints the
// handshake line, answers the health check for the service "provider", and shuts down when asked.
//
// A provider must be started by a runner, which sets what Handshake describes in its environment. Started
// by hand, it says so and exits, so run it with `steward provider validate` instead.
func (p *Provider) Serve() {
	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins:         goplugin.PluginSet{Name: &grpcProvider{provider: p}},
		GRPCServer: func(options []grpc.ServerOption) *grpc.Server {
			return goplugin.DefaultGRPCServer(append(options, grpc.ChainUnaryInterceptor(recoverPanics)))
		},
	})
}

// grpcProvider registers the services of the provider with the gRPC server that go-plugin runs.
type grpcProvider struct {
	goplugin.NetRPCUnsupportedPlugin
	provider *Provider
}

func (g *grpcProvider) GRPCServer(_ *goplugin.GRPCBroker, server *grpc.Server) error {
	p := g.provider

	providerv1.RegisterProviderServiceServer(server, providerService{provider: p})

	if len(p.resources) > 0 {
		providerv1.RegisterResourceServiceServer(server, resourceService{provider: p})
	}

	if len(p.applications) > 0 {
		providerv1.RegisterApplicationServiceServer(server, applicationService{provider: p})
	}

	if len(p.connectors) > 0 {
		providerv1.RegisterConnectorServiceServer(server, connectorService{provider: p})
	}

	return nil
}

// GRPCClient is the runner's side. A runner uses the connection as it is, with the generated clients.
func (g *grpcProvider) GRPCClient(_ context.Context, _ *goplugin.GRPCBroker, conn *grpc.ClientConn) (any, error) {
	return conn, nil
}

// ClientProvider is what a runner passes to go-plugin, under Name, so that it gets the gRPC connection
// back from Dispense.
func ClientProvider() goplugin.Plugin {
	return &grpcProvider{}
}

// recoverPanics reports a panic in a handler as an INTERNAL error, with its stack on stderr, instead
// of letting it take the whole provider down.
func recoverPanics(ctx context.Context, request any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (response any, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			fmt.Fprintf(os.Stderr, "panic in handler: %v\n%s", recovered, debug.Stack())
			err = status.Error(codes.Internal, fmt.Sprint(recovered))
		}
	}()

	return handler(ctx, request)
}

// The services route each call to the handler of the same name on the provider or on the kind the
// request is about, and answer a call without a handler with UNIMPLEMENTED. The generated Unimplemented
// servers keep the provider compatible when the contract gains calls.

type providerService struct {
	providerv1.UnimplementedProviderServiceServer
	provider *Provider
}

func (s providerService) Describe(ctx context.Context, request *providerv1.DescribeRequest) (*providerv1.DescribeResponse, error) {
	return s.provider.describe(ctx, request.GetConnector()), nil
}

func (s providerService) GrantAccess(ctx context.Context, request *ProviderGrantAccessRequest) (*ProviderGrantAccessResponse, error) {
	if s.provider.options.GrantAccess == nil {
		return nil, unimplemented("grant_access", "provider")
	}

	return s.provider.options.GrantAccess(ctx, request)
}

func (s providerService) RevokeAccess(ctx context.Context, request *ProviderRevokeAccessRequest) (*ProviderRevokeAccessResponse, error) {
	if s.provider.options.RevokeAccess == nil {
		return nil, unimplemented("revoke_access", "provider")
	}

	return s.provider.options.RevokeAccess(ctx, request)
}

func (s providerService) GetAccess(ctx context.Context, request *ProviderGetAccessRequest) (*ProviderGetAccessResponse, error) {
	if s.provider.options.GetAccess == nil {
		return nil, unimplemented("get_access", "provider")
	}

	return s.provider.options.GetAccess(ctx, request)
}

type resourceService struct {
	providerv1.UnimplementedResourceServiceServer
	provider *Provider
}

func (s resourceService) Provision(ctx context.Context, request *ResourceProvisionRequest) (*ResourceProvisionResponse, error) {
	kind, err := s.provider.resource(request.GetKind())
	if err != nil {
		return nil, err
	}

	if kind.Provision == nil {
		return nil, unimplemented("provision", "resource")
	}

	return kind.Provision(ctx, request)
}

func (s resourceService) Deprovision(ctx context.Context, request *ResourceDeprovisionRequest) (*ResourceDeprovisionResponse, error) {
	kind, err := s.provider.resource(request.GetResource().GetKind())
	if err != nil {
		return nil, err
	}

	if kind.Deprovision == nil {
		return nil, unimplemented("deprovision", "resource")
	}

	return kind.Deprovision(ctx, request)
}

func (s resourceService) List(ctx context.Context, request *ResourceListRequest) (*ResourceListResponse, error) {
	kind, err := s.provider.resource(request.GetKind())
	if err != nil {
		return nil, err
	}

	if kind.List == nil {
		return nil, unimplemented("list", "resource")
	}

	return kind.List(ctx, request)
}

func (s resourceService) GrantAccess(ctx context.Context, request *ResourceGrantAccessRequest) (*ResourceGrantAccessResponse, error) {
	kind, err := s.provider.resource(request.GetResource().GetKind())
	if err != nil {
		return nil, err
	}

	if kind.GrantAccess == nil {
		return nil, unimplemented("grant_access", "resource")
	}

	return kind.GrantAccess(ctx, request)
}

func (s resourceService) RevokeAccess(ctx context.Context, request *ResourceRevokeAccessRequest) (*ResourceRevokeAccessResponse, error) {
	kind, err := s.provider.resource(request.GetResource().GetKind())
	if err != nil {
		return nil, err
	}

	if kind.RevokeAccess == nil {
		return nil, unimplemented("revoke_access", "resource")
	}

	return kind.RevokeAccess(ctx, request)
}

func (s resourceService) GetAccess(ctx context.Context, request *ResourceGetAccessRequest) (*ResourceGetAccessResponse, error) {
	kind, err := s.provider.resource(request.GetResource().GetKind())
	if err != nil {
		return nil, err
	}

	if kind.GetAccess == nil {
		return nil, unimplemented("get_access", "resource")
	}

	return kind.GetAccess(ctx, request)
}

type applicationService struct {
	providerv1.UnimplementedApplicationServiceServer
	provider *Provider
}

func (s applicationService) Create(ctx context.Context, request *ApplicationCreateRequest) (*ApplicationCreateResponse, error) {
	kind, err := s.provider.application(request.GetKind())
	if err != nil {
		return nil, err
	}

	if kind.Create == nil {
		return nil, unimplemented("create", "application")
	}

	return kind.Create(ctx, request)
}

func (s applicationService) Destroy(ctx context.Context, request *ApplicationDestroyRequest) (*ApplicationDestroyResponse, error) {
	kind, err := s.provider.application(request.GetApplication().GetKind())
	if err != nil {
		return nil, err
	}

	if kind.Destroy == nil {
		return nil, unimplemented("destroy", "application")
	}

	return kind.Destroy(ctx, request)
}

func (s applicationService) Deploy(ctx context.Context, request *ApplicationDeployRequest) (*ApplicationDeployResponse, error) {
	kind, err := s.provider.application(request.GetApplication().GetKind())
	if err != nil {
		return nil, err
	}

	if kind.Deploy == nil {
		return nil, unimplemented("deploy", "application")
	}

	return kind.Deploy(ctx, request)
}

type connectorService struct {
	providerv1.UnimplementedConnectorServiceServer
	provider *Provider
}

func (s connectorService) Resolve(ctx context.Context, request *ConnectorResolveRequest) (*ConnectorResolveResponse, error) {
	credential, err := s.provider.connector(request.GetKind())
	if err != nil {
		return nil, err
	}

	if credential.Resolve == nil {
		return nil, unimplemented("resolve", "credential")
	}

	return credential.Resolve(ctx, request)
}
