"""The provider part of the Treno SDK for Python: the plumbing between the provider contract (gRPC,
see ../proto) and the functions a provider writes.

    from treno_sdk.provider import create_provider


    provider = create_provider(name="github", version="0.1.0", inputs=[...], validate=validate)
    provider.service(kind="repository", inputs=[...], provision=provision, deprovision=deprovision)
    provider.serve()

A provider declares itself and each of its service or application kinds once, with the handlers
attached. The SDK routes every call to the right handler by kind, builds the description Treno asks
for, serves the gRPC services, prints the handshake line, answers the health check, and answers any
call without a handler with UNIMPLEMENTED.

Everything a provider logs must go to stderr: stdout is reserved for the handshake line.
"""

from treno_sdk._gen.treno.provider.v1.provider_pb2 import *  # noqa: F401,F403
from treno_sdk.provider._provider import (
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
    ServiceDeprovisionRequest,
    ServiceDeprovisionResponse,
    ServiceGetAccessRequest,
    ServiceGetAccessResponse,
    ServiceGrantAccessRequest,
    ServiceGrantAccessResponse,
    ServiceListRequest,
    ServiceListResponse,
    ServiceProvisionRequest,
    ServiceProvisionResponse,
    ServiceRevokeAccessRequest,
    ServiceRevokeAccessResponse,
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
    "ServiceDeprovisionRequest",
    "ServiceDeprovisionResponse",
    "ServiceGetAccessRequest",
    "ServiceGetAccessResponse",
    "ServiceGrantAccessRequest",
    "ServiceGrantAccessResponse",
    "ServiceListRequest",
    "ServiceListResponse",
    "ServiceProvisionRequest",
    "ServiceProvisionResponse",
    "ServiceRevokeAccessRequest",
    "ServiceRevokeAccessResponse",
    "StatusCode",
    "create_provider",
    "plain",
]
