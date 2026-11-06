"""The provider part of the Steward SDK for Python: the plumbing between the provider contract (gRPC,
see ../proto) and the functions a provider writes.

    from steward_sdk.provider import create_provider


    provider = create_provider(name="github", version="0.1.0", inputs=[...], validate=validate)
    provider.resource(kind="repository", inputs=[...], provision=provision, deprovision=deprovision)
    provider.serve()

A provider declares itself and each of its resource or application kinds once, with the handlers
attached. The SDK routes every call to the right handler by kind, builds the description Steward asks
for, serves the gRPC services, prints the handshake line, answers the health check, and answers any
call without a handler with UNIMPLEMENTED.

Everything a provider logs must go to stderr: stdout is reserved for the handshake line.
"""

from steward_sdk._gen.steward.provider.v1.provider_pb2 import *  # noqa: F401,F403
from steward_sdk.provider._provider import (
    ApplicationCreateRequest,
    ApplicationCreateResponse,
    ApplicationDeployRequest,
    ApplicationDeployResponse,
    ApplicationDestroyRequest,
    ApplicationDestroyResponse,
    Provider,
    ProviderError,
    ProviderGetAccessRequest,
    ProviderGetAccessResponse,
    ProviderGrantAccessRequest,
    ProviderGrantAccessResponse,
    ProviderRevokeAccessRequest,
    ProviderRevokeAccessResponse,
    ProviderValidateRequest,
    ProviderValidateResponse,
    ResourceDeprovisionRequest,
    ResourceDeprovisionResponse,
    ResourceGetAccessRequest,
    ResourceGetAccessResponse,
    ResourceGrantAccessRequest,
    ResourceGrantAccessResponse,
    ResourceListRequest,
    ResourceListResponse,
    ResourceProvisionRequest,
    ResourceProvisionResponse,
    ResourceRevokeAccessRequest,
    ResourceRevokeAccessResponse,
    StatusCode,
    create_provider,
    plain,
)

__all__ = [
    "ApplicationCreateRequest",
    "ApplicationCreateResponse",
    "ApplicationDeployRequest",
    "ApplicationDeployResponse",
    "ApplicationDestroyRequest",
    "ApplicationDestroyResponse",
    "Provider",
    "ProviderError",
    "ProviderGetAccessRequest",
    "ProviderGetAccessResponse",
    "ProviderGrantAccessRequest",
    "ProviderGrantAccessResponse",
    "ProviderRevokeAccessRequest",
    "ProviderRevokeAccessResponse",
    "ProviderValidateRequest",
    "ProviderValidateResponse",
    "ResourceDeprovisionRequest",
    "ResourceDeprovisionResponse",
    "ResourceGetAccessRequest",
    "ResourceGetAccessResponse",
    "ResourceGrantAccessRequest",
    "ResourceGrantAccessResponse",
    "ResourceListRequest",
    "ResourceListResponse",
    "ResourceProvisionRequest",
    "ResourceProvisionResponse",
    "ResourceRevokeAccessRequest",
    "ResourceRevokeAccessResponse",
    "StatusCode",
    "create_provider",
    "plain",
]
