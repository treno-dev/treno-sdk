# Steward SDK for Go

The Steward SDK for Go. Today it holds the part for writing Steward providers, in the package
`github.com/treno-dev/steward-sdk/go/provider`. A provider declares what it needs to connect to the tools it
works with and what it manages there, and implements the calls Steward makes. The SDK handles
everything else: the gRPC services, the handshake with the runner, the health check, and routing each
call to your handler by kind.

```sh
go get github.com/treno-dev/steward-sdk/go
```

```go
import "github.com/treno-dev/steward-sdk/go/provider"
```

Requires Go 1.25 or newer.

## Start a provider

```sh
steward provider init my-provider --language go
cd my-provider
go mod tidy
steward provider validate go run .
```

`steward provider init` takes its templates from the SDK release, so the project always matches the SDK
it depends on. To work on the SDK and its templates together, point it at a local checkout with
`--sdk-path /path/to/steward-sdk`.

The generated provider, with a service that can be provisioned and given access to, is
[`../templates/go/main.go.tmpl`](../templates/go/main.go.tmpl). A provider needs one
`provider.New(...)`, any number of `Service(...)` and `Application(...)` declarations, and a final
`Serve()`. If it needs more than one tool, for example both Cloudflare and AWS, take a secret for
each tool.

```go
p := provider.New(provider.Options{Name: "github", Version: "0.1.0", Inputs: inputs, Validate: validate})

p.Service("repository", provider.ServiceOptions{
	Provision:   provision,
	Deprovision: deprovision,
})

p.Serve()
```

## What you declare

**The provider**, in `provider.Options`:

| Field | |
|---|---|
| `Name`, `Version` | Required. |
| `Title`, `Description` | Shown in Steward. |
| `Inputs` | The settings the provider is configured with. |
| `Secrets` | The secrets it is configured with, such as an API token. |
| `Validate` | Optional. Checks the secrets, which Steward cannot. Returns `ValidationError`s the user can fix, in the list of what they are about: `InputErrors`, `SecretErrors`, or `Errors` for the credential as a whole. Without it, every credential is accepted. |
| `Roles`, `Permissions`, `GrantAccess`, `RevokeAccess`, `GetAccess` | Access to the provider as a whole, such as membership of an organization. |

**A service**, with `p.Service("kind", provider.ServiceOptions{...})`, and **an application**,
with `p.Application("kind", provider.ApplicationOptions{...})`, are each declared under a kind that
is unique within the provider, such as `"repository"`, and take a `Title`, a `Description`, `Inputs`,
`Secrets` and `Outputs`. Services also take `Roles` and `Permissions`. Applications take `Sources`, the types they can
be deployed from: `"github"`, `"registry"`, `"s3"` or `"raw"`. The `Source` of an application arrives as
`Type`, `Config` and `Ref`, where `Config` holds the settings of that type: `github` `{owner, name,
branch?}`, `registry` `{image, tag?}`, `s3` `{bucket, key}` and `raw` `{path}`. `Ref` pins a commit, an
image digest or an object version, and is empty for the newest.

**Inputs and outputs** are declared like variables:

```go
&provider.InputDefinition{Name: "visibility", Label: "Visibility", Type: provider.TypeSelect, Options: []string{"private", "public"}}
```

`Type` is one of `TypeString`, `TypeNumber`, `TypeBoolean`, `TypeSelect`, `TypeList` (of strings) or
`TypeMap`. Mark an output `Sensitive: true` when it is a secret, such as a generated password, and it
stays with the runner.

**Secrets** are kept, sent and shown apart from inputs. Each has a `Name`, used as
it is, a `Description` and whether it is `Required`. Its value is never shown back once set.

```go
&provider.SecretDefinition{Name: "GITHUB_TOKEN", Description: "A token that can create repositories.", Required: true}
```

## Access

Access exists at two levels, each with the same three handlers, `GrantAccess`, `RevokeAccess` and
`GetAccess`:

- **The provider**, in `provider.Options`: membership of the tool as a whole, such as an organization or a
  site. Steward grants this first, then access to the services.
- **A service kind**, in `provider.ServiceOptions`: a role on one service, such as a repository.

Declaring `Roles` or `Permissions` is what says that the provider, or the kind, supports access. They are
two separate lists:

- **`Roles`** are the ones the tool defines itself, each with a `Name`, `Title` and `Description`.
- **`Permissions`** are the pieces Steward builds its own roles from. A permission is a `Name` with an
  optional `Level`.

```go
p := provider.New(provider.Options{
	Name:         "github",
	Version:      "0.1.0",
	Inputs:       inputs,
	Roles:        []*provider.RoleDefinition{{Name: "member", Title: "Member"}, {Name: "owner", Title: "Owner"}},
	GrantAccess:  inviteMember,
	RevokeAccess: removeMember,
	GetAccess:    findMember,
})

p.Service("repository", provider.ServiceOptions{
	Permissions:  []*provider.PermissionDefinition{{Name: "pull"}, {Name: "push"}},
	Roles:        []*provider.RoleDefinition{{Name: "viewer", Title: "Viewer"}, {Name: "editor", Title: "Editor"}},
	GrantAccess:  addCollaborator,
	RevokeAccess: removeCollaborator,
	GetAccess:    findCollaborator,
})
```

The provider's handlers take the `Provider…` messages and a service's take the `Service…` ones:

```go
func inviteMember(ctx context.Context, request *provider.ProviderGrantAccessRequest) (*provider.ProviderGrantAccessResponse, error)
func addCollaborator(ctx context.Context, request *provider.ServiceGrantAccessRequest) (*provider.ServiceGrantAccessResponse, error)
```

Handlers receive the `Identity` (`ExternalId`, `Name`, `Attrs`) and the `Role` to apply, and a service
call also gets the `Service` (`Kind`, `Name`). The `Role` is always set. For a role Steward composed
from your permissions, `Permissions` lists its pieces too, for you to apply as far as the tool allows.
They return:

- `GrantAccess`: the `Role` and `Permissions` as the tool applied them, which may differ from what was
  requested, an `Id` for the grant if the tool has one (Steward sends it back on revoke), and
  `Pending: true` while the person has to act first, such as accepting an invitation.
- `RevokeAccess`: nothing. It succeeds if the identity did not have the role.
- `GetAccess`: the `Roles` and `Permissions` the identity holds now, and whether a grant is still
  `Pending`.

Steward works out overlap between roles before it asks you to revoke one, so you can remove the role
you are given in full.

## What Steward offers

Steward offers what you implement. A call for a handler you did not set answers `UNIMPLEMENTED`, and
Steward treats that as not offered. Access also needs its `Roles` or `Permissions`, which tell Steward
what can be granted.

| You set | The kind can |
|---|---|
| `Provision`, `Deprovision` | be created and removed |
| `List` | be discovered (optional: Steward keeps track of what it creates, so `List` is only for finding services that already exist) |
| `Roles` or `Permissions`, with `GrantAccess`, `RevokeAccess`, `GetAccess` | have access granted (services and the provider) |
| `Create`, `Destroy`, `Deploy` | be run as an application |

## Handlers

A handler is `func(context.Context, *Request) (*Response, error)`, with the request and response named
for the entity and the call: `ServiceProvisionRequest`, `ServiceProvisionResponse`. They are the
generated contract messages, re-exported by the package so a provider imports one thing.

- **Getters are nil-safe.** Use `request.GetCredential().GetSecrets()["API_TOKEN"]`: anything the runner
  did not send reads as empty.
- **Inputs and secrets.** `Credential.Inputs` and `Inputs` hold the values of the inputs; call `.AsMap()`
  on one for a plain `map[string]any` (numbers arrive as `float64`). The values of secrets arrive apart
  from them, keyed by secret name, in `Credential.Secrets` (and a service's or application's `Secrets`)
  as a `map[string]string`. Never store them.
- **Open-ended values in a response**, such as `Outputs`, are built with `provider.Struct(map[string]any{...})`,
  and an input's `Default` with `provider.Value(...)`. They take what JSON can hold; anything else
  panics, which the SDK reports as an `INTERNAL` error with the stack on stderr, as it does for any
  panic in a handler.
- **Idempotent, and no report of changes.** Make calls that change something idempotent: creating what
  exists, or removing what is gone, simply succeeds. Return the result of the change (the `Outputs`, the
  applied `Role`), not a description of what changed: Steward keeps the state and works out the
  difference itself.
- **Services and identities.** A service is identified by its `Kind` and `Name`, which Steward stores
  and sends on every later call. `Provision` receives them with the `Inputs` and returns only the
  `Outputs`, such as a URL or an id the tool assigned.
- **Variables.** An application's variables each carry `Sensitive`, `Sealed` and `Locked`. A `Deploy`
  carries the application's full set of variables, so changing them is a deploy. Fail it with
  `FAILED_PRECONDITION` if it would change or remove a locked variable.

## Errors

Return a gRPC status to report a specific failure:

```go
return nil, status.Error(codes.FailedPrecondition, "APP_KEY is locked and cannot be removed")
```

Common codes: `InvalidArgument`, `NotFound`, `FailedPrecondition`, `Unimplemented`. Any other error is
reported as `INTERNAL`. A call for an unknown kind is answered with `NotFound` by the SDK.

## Logging

Write logs to standard error, which the runner collects. A provider must be started by a runner, which sets
what `provider.Handshake` describes: started by hand it says so and exits, so use `steward provider validate`.

## Developing the SDK

The messages and gRPC services in `gen/` are generated from `../proto` with `buf` (`buf.gen.yaml`,
using buf's remote providers, so it needs network access). Unlike the other SDKs the generated code **is
committed**, because Go consumers install the module as it is, with no build step. Regenerate it after
the contract changes:

```sh
buf generate
go vet ./... && go build ./...
```
