// The provider part of the Treno SDK for JavaScript: the plumbing between the provider contract (gRPC,
// see ../../../proto) and the plain objects a provider works with.
//
//   import { createProvider } from '@treno-dev/treno-sdk/provider';
//
//   const provider = createProvider({ name: 'github', version: '0.1.0', inputs: [...], validate });
//   provider.service({ kind: 'repository', inputs: [...], provision, deprovision, list });
//   provider.serve();
//
// A provider declares itself and each of its service or application kinds once, with the handlers
// attached. The SDK routes every call to the right handler by kind, builds the description Treno
// asks for, serves the gRPC services, prints the handshake line, answers the health check, fills in
// whatever a handler leaves out of its response, and answers any call without a handler with
// UNIMPLEMENTED.
//
// Everything a provider logs must go to stderr: stdout is reserved for the handshake line.
//
// Messages are generated from the contract, which also turns google.protobuf.Struct into plain
// objects, so handlers never deal with protobuf values.

import { randomBytes } from 'node:crypto';
import { join } from 'node:path';

import * as grpc from '@grpc/grpc-js';

import healthCheck from 'grpc-health-check';

import type {
  ApplicationDefinition,
  DeepPartial,
  InputDefinition,
  OutputDefinition,
  PermissionDefinition,
  ProviderDefinition,
  ServiceDefinition,
  RoleDefinition,
  SecretDefinition,
} from '../gen/treno/provider/v1/provider.js';
import * as contract from '../gen/treno/provider/v1/provider.js';

export * from '../gen/treno/provider/v1/provider.js';

const { HealthImplementation } = healthCheck;

// The version of the Treno provider contract this provider speaks.
const APP_PROTOCOL_VERSION = 1;

// A runner sets this cookie in the environment of every provider it starts, as hashicorp/go-plugin does.
// A provider that does not find it was started by hand. Kept in step with go/provider/handshake.go.
const MAGIC_COOKIE_KEY = 'TRENO_PROVIDER';
const MAGIC_COOKIE_VALUE = 'b6d7a1f2-treno-provider';

type MaybePromise<T> = T | Promise<T>;

// Protobuf lets a message be absent, so the generated types mark every message field as possibly
// undefined. The runner always sends what a handler needs, and `fill` below makes sure of it, so a
// handler's request has its message fields present. Optional scalars, such as an identity's
// externalId, stay optional because they really can be missing.
type Present<Value> = NonNullable<Value> extends object ? Complete<NonNullable<Value>> : Value;
type Complete<Message> = Message extends (infer Item)[]
  ? Complete<Item>[]
  : Message extends object
    ? { [Field in keyof Message]: Present<Message[Field]> }
    : Message;

// The request and response of each handler, named for what it handles (the entity, then the call),
// for annotating your own handlers. A request has its message fields present, and a response may
// leave fields out.
export type ProviderValidateRequest = Complete<contract.ValidateRequest>;
export type ProviderValidateResponse = DeepPartial<contract.ValidateResponse>;
export type ProviderGrantAccessRequest = Complete<contract.ProviderServiceGrantAccessRequest>;
export type ProviderGrantAccessResponse = DeepPartial<contract.ProviderServiceGrantAccessResponse>;
export type ProviderRevokeAccessRequest = Complete<contract.ProviderServiceRevokeAccessRequest>;
export type ProviderRevokeAccessResponse = DeepPartial<contract.ProviderServiceRevokeAccessResponse>;
export type ProviderGetAccessRequest = Complete<contract.ProviderServiceGetAccessRequest>;
export type ProviderGetAccessResponse = DeepPartial<contract.ProviderServiceGetAccessResponse>;
export type ServiceProvisionRequest = Complete<contract.ServiceServiceProvisionRequest>;
export type ServiceProvisionResponse = DeepPartial<contract.ServiceServiceProvisionResponse>;
export type ServiceDeprovisionRequest = Complete<contract.ServiceServiceDeprovisionRequest>;
export type ServiceDeprovisionResponse = DeepPartial<contract.ServiceServiceDeprovisionResponse>;
export type ServiceListRequest = Complete<contract.ServiceServiceListRequest>;
export type ServiceListResponse = DeepPartial<contract.ServiceServiceListResponse>;
export type ServiceGrantAccessRequest = Complete<contract.ServiceServiceGrantAccessRequest>;
export type ServiceGrantAccessResponse = DeepPartial<contract.ServiceServiceGrantAccessResponse>;
export type ServiceRevokeAccessRequest = Complete<contract.ServiceServiceRevokeAccessRequest>;
export type ServiceRevokeAccessResponse = DeepPartial<contract.ServiceServiceRevokeAccessResponse>;
export type ServiceGetAccessRequest = Complete<contract.ServiceServiceGetAccessRequest>;
export type ServiceGetAccessResponse = DeepPartial<contract.ServiceServiceGetAccessResponse>;
export type ApplicationCreateRequest = Complete<contract.ApplicationServiceCreateRequest>;
export type ApplicationCreateResponse = DeepPartial<contract.ApplicationServiceCreateResponse>;
export type ApplicationDestroyRequest = Complete<contract.ApplicationServiceDestroyRequest>;
export type ApplicationDestroyResponse = DeepPartial<contract.ApplicationServiceDestroyResponse>;
export type ApplicationDeployRequest = Complete<contract.ApplicationServiceDeployRequest>;
export type ApplicationDeployResponse = DeepPartial<contract.ApplicationServiceDeployResponse>;

// For a generated service definition, the handler of each call: it takes the request and returns
// the response, which may leave fields out.
export type Handlers<Service> = {
  [Call in keyof Service]?: Service[Call] extends {
    requestDeserialize: (value: Buffer) => infer Request;
    responseDeserialize: (value: Buffer) => infer Response;
  }
    ? (request: Complete<Request>) => MaybePromise<DeepPartial<Response>>
    : never;
};

type Describe = {
  title?: string;
  description?: string;
  inputs?: DeepPartial<InputDefinition>[];
  secrets?: DeepPartial<SecretDefinition>[];
};

type Access = {
  permissions?: DeepPartial<PermissionDefinition>[];
  roles?: DeepPartial<RoleDefinition>[];
};

// The provider: what is needed to configure it, and, when it supports access as a whole, the handlers
// for that.
export type ProviderOptions = Describe &
  Access & {
    name: string;
    version: string;
  } & Omit<Handlers<typeof contract.ProviderServiceService>, 'describe'>;

export type ServiceOptions = Describe &
  Access & {
    kind: string;
    outputs?: DeepPartial<OutputDefinition>[];
  } & Handlers<typeof contract.ServiceServiceService>;

export type ApplicationOptions = Describe & {
  kind: string;
  sources?: string[];
  outputs?: DeepPartial<OutputDefinition>[];
} & Handlers<typeof contract.ApplicationServiceService>;

// A request carries the kind or the thing it is about.
type AnyRequest = {
  kind?: string;
  service?: { kind: string };
  application?: { kind: string };
};

type Target = 'provider' | 'service' | 'application';

const ACCESS_CALLS = ['grantAccess', 'revokeAccess', 'getAccess'] as const;
const PROVISIONING_CALLS = ['provision', 'deprovision'] as const;

function fail(code: grpc.status, message: string): never {
  throw { code, message };
}

function serviceDefinition(options: ServiceOptions): DeepPartial<ServiceDefinition> {
  const { kind, title, description, inputs, secrets, outputs, permissions, roles } = options;

  return { kind, title, description, inputs, secrets, outputs, permissions, roles };
}

function applicationDefinition(options: ApplicationOptions): DeepPartial<ApplicationDefinition> {
  const { kind, title, description, inputs, secrets, sources, outputs } = options;

  return { kind, title, description, inputs, secrets, sources, outputs };
}

export class Provider {
  readonly services = new Map<string, ServiceOptions>();
  readonly applications = new Map<string, ApplicationOptions>();

  constructor(readonly options: ProviderOptions) {}

  /** Declares a kind of service the provider manages, with its handlers. */
  service(options: ServiceOptions): this {
    this.services.set(options.kind, options);

    return this;
  }

  /** Declares a kind of application the provider runs, with its handlers. */
  application(options: ApplicationOptions): this {
    this.applications.set(options.kind, options);

    return this;
  }

  /** Starts serving the provider. The runner stops it when it is done. */
  serve(): void {
    serve(this);
  }

  describe() {
    const { name, version } = this.options;

    return { name, version, definition: this.definition() };
  }

  // Routes a call to the handler of the same name on whatever the request is about: the
  // provider itself, or one of its kinds of service or application.
  route(call: string, target: Target, kindOf: (request: AnyRequest) => string = () => '') {
    return (request: AnyRequest) => {
      const options = target === 'provider' ? this.options : this.kind(target, kindOf(request));
      const handler = (options as Record<string, unknown>)[call] as ((request: AnyRequest) => unknown) | undefined;

      return handler
        ? handler(request)
        : fail(grpc.status.UNIMPLEMENTED, `${call} is not implemented for this ${target}`);
    };
  }

  private definition(): DeepPartial<ProviderDefinition> {
    const { title, description, inputs, secrets, permissions, roles } = this.options;

    return {
      title,
      description,
      inputs,
      secrets,
      permissions,
      roles,
      services: [...this.services.values()].map(serviceDefinition),
      applications: [...this.applications.values()].map(applicationDefinition),
    };
  }

  private kind(target: 'service' | 'application', kind: string): object {
    const kinds = target === 'service' ? this.services : this.applications;

    return kinds.get(kind) ?? fail(grpc.status.NOT_FOUND, `this provider has no ${target} "${kind}"`);
  }
}

/** Creates a provider. Declare its services, then call `serve()`. */
export function createProvider(options: ProviderOptions): Provider {
  return new Provider(options);
}

// Serving --------------------------------------------------------------------------------------

type Registered = Record<string, { path: string }>;
type Routed = Record<string, (request: AnyRequest) => unknown>;
type MessageType = { fromPartial(object: unknown): unknown };

// A call's response type is named after the call, optionally prefixed with its service, which is
// the naming the contract's lint enforces: DescribeResponse, ServiceServiceProvisionResponse.
function responseType(path: string): MessageType {
  const [, service, method] = /\.([A-Za-z]+)\/([A-Za-z]+)$/.exec(path) ?? [];
  const messages = contract as unknown as Record<string, MessageType | undefined>;
  const type = messages[`${method}Response`] ?? messages[`${service}${method}Response`];

  if (!type) {
    throw new Error(`no response type found for ${path}`);
  }

  return type;
}

function toStatus(error: unknown): grpc.ServiceError {
  if (typeof error === 'object' && error !== null && typeof (error as { code?: unknown }).code === 'number') {
    return error as grpc.ServiceError;
  }

  const message = error instanceof Error ? error.message : String(error);

  return { code: grpc.status.INTERNAL, message } as grpc.ServiceError;
}

// Makes the message fields a handler relies on present, so it can read `credential.inputs.x` without
// checking. An absent message becomes the generated type's default, and an absent Struct becomes an
// empty object.
const MESSAGES: Record<string, MessageType> = {
  credential: contract.Credential,
  service: contract.Service,
  application: contract.Application,
  identity: contract.Identity,
  role: contract.Role,
  source: contract.ApplicationSource,
};
const STRUCTS = ['inputs', 'outputs', 'attrs'];

function fill(value: unknown): unknown {
  if (Array.isArray(value)) {
    return value.map(fill);
  }

  if (typeof value !== 'object' || value === null) {
    return value;
  }

  const message = value as Record<string, unknown>;

  for (const [field, item] of Object.entries(message)) {
    const absent = item === undefined || item === null;

    if (absent && field in MESSAGES) {
      message[field] = fill(MESSAGES[field].fromPartial({}));
      continue;
    }

    message[field] = absent && STRUCTS.includes(field) ? {} : fill(item);
  }

  return message;
}

function unary(path: string, handler: (request: AnyRequest) => unknown): grpc.handleUnaryCall<unknown, unknown> {
  const response = responseType(path);

  return (call, callback) => {
    Promise.resolve()
      .then(() => handler(fill(call.request) as AnyRequest))
      .then((result) => callback(null, response.fromPartial(result ?? {})))
      .catch((error) => callback(toStatus(error)));
  };
}

function unimplemented(path: string): grpc.handleUnaryCall<unknown, unknown> {
  return (_call, callback) =>
    callback({
      code: grpc.status.UNIMPLEMENTED,
      message: `${path} is not implemented by this provider`,
    } as grpc.ServiceError);
}

// Maps each routed call onto the service call of the same name, and answers any call without a
// route with UNIMPLEMENTED.
function implement(service: Registered, routed: Routed): grpc.UntypedServiceImplementation {
  const implementation: grpc.UntypedServiceImplementation = {};

  for (const [name, definition] of Object.entries(service)) {
    const handler = routed[name];

    implementation[name] = handler ? unary(definition.path, handler) : unimplemented(definition.path);
  }

  return implementation;
}

function routes(provider: Provider, calls: readonly string[], target: Target, kindOf?: (request: AnyRequest) => string): Routed {
  return Object.fromEntries(calls.map((call) => [call, provider.route(call, target, kindOf)]));
}

// Where to listen: a unix socket in the directory the runner made for it, when it set one, which only the
// runner's user can open; otherwise a port on the loopback interface.
function listenAddress(): { bind: string; announce: (port: number) => string } {
  const directory = process.env.PROVIDER_UNIX_SOCKET_DIR;

  if (directory && process.platform !== 'win32') {
    const path = join(directory, `provider-${randomBytes(6).toString('hex')}`);

    return { bind: `unix:${path}`, announce: () => `unix|${path}` };
  }

  return { bind: '127.0.0.1:0', announce: (port) => `tcp|127.0.0.1:${port}` };
}

// Serves the call go-plugin makes to ask a provider to shut down, so that it exits at once and the
// runner does not wait for it. It has no generated code, since its messages are empty.
function addShutdown(server: grpc.Server, stop: () => void): void {
  const empty = { serialize: () => Buffer.alloc(0), deserialize: () => ({}) };

  server.addService(
    {
      Shutdown: {
        path: '/provider.GRPCController/Shutdown',
        requestStream: false,
        responseStream: false,
        requestSerialize: empty.serialize,
        requestDeserialize: empty.deserialize,
        responseSerialize: empty.serialize,
        responseDeserialize: empty.deserialize,
      },
    },
    {
      Shutdown: (_call: unknown, callback: grpc.sendUnaryData<unknown>) => {
        callback(null, {});
        setImmediate(stop);
      },
    },
  );
}

function serve(provider: Provider): void {
  if (process.env[MAGIC_COOKIE_KEY] !== MAGIC_COOKIE_VALUE) {
    console.error('This program is a Treno provider. It is started by a runner, not run directly.');
    console.error('To try it, use `treno provider validate` or `treno provider run`.');
    process.exit(1);
  }

  // A runner that has gone leaves these pipes broken, and Node crashes on the error that a direct write to
  // one raises. console.* already ignores it; this covers process.stdout.write and process.stderr.write, so
  // a provider outlives its runner and a new runner can reattach to it.
  for (const stream of [process.stdout, process.stderr]) {
    stream.on('error', () => {});
  }

  const server = new grpc.Server();
  const anyService = provider.services.size > 0;
  const anyApplication = provider.applications.size > 0;

  server.addService(contract.ProviderServiceService, implement(contract.ProviderServiceService, {
    describe: () => provider.describe(),
    // Every provider must answer Validate; without a handler of its own, every credential is accepted.
    validate: provider.options.validate ? provider.route('validate', 'provider') : () => ({}),
    ...routes(provider, ACCESS_CALLS, 'provider'),
  }));

  if (anyService) {
    const serviceKind = (request: AnyRequest) => request.service?.kind ?? request.kind ?? '';

    server.addService(
      contract.ServiceServiceService as unknown as grpc.ServiceDefinition,
      implement(
        contract.ServiceServiceService,
        routes(provider, [...ACCESS_CALLS, ...PROVISIONING_CALLS, 'list'], 'service', serviceKind),
      ),
    );
  }

  if (anyApplication) {
    const applicationKind = (request: AnyRequest) => request.application?.kind ?? request.kind ?? '';
    const calls = ['create', 'destroy', 'deploy'];

    server.addService(
      contract.ApplicationServiceService as unknown as grpc.ServiceDefinition,
      implement(contract.ApplicationServiceService, routes(provider, calls, 'application', applicationKind)),
    );
  }

  new HealthImplementation({ provider: 'SERVING' }).addToServer(server);

  const stop = () => server.tryShutdown(() => process.exit(0));

  addShutdown(server, stop);

  const { bind, announce } = listenAddress();

  server.bindAsync(bind, grpc.ServerCredentials.createInsecure(), (error, port) => {
    if (error) {
      console.error(error.message);
      process.exit(1);
    }

    // CORE-PROTOCOL-VERSION | APP-PROTOCOL-VERSION | NETWORK-TYPE | NETWORK-ADDR | PROTOCOL
    console.log(`1|${APP_PROTOCOL_VERSION}|${announce(port)}|grpc`);
  });

  for (const signal of ['SIGINT', 'SIGTERM'] as const) {
    process.on(signal, stop);
  }
}
