package provider

import goplugin "github.com/hashicorp/go-plugin"

// Handshake is what a runner and a provider agree on before they talk, in the terms of hashicorp/go-plugin,
// which the runner uses to start providers. The protocol version is the version of the provider contract.
// The magic cookie is set in the environment of every provider a runner starts: a provider that does not find
// it was started by hand, not by a runner, and says so and exits.
//
// A provider in another language does the same with its own code. It reads STEWARD_PROVIDER from its
// environment, prints the handshake line, and, when PROVIDER_UNIX_SOCKET_DIR is set, listens on a unix
// socket created in that directory.
var Handshake = goplugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "STEWARD_PROVIDER",
	MagicCookieValue: "b6d7a1f2-steward-provider",
}

// Name is the name a provider is served under, and the name of the service its health check answers for. go-plugin registers that
// service under its own name, so this is its constant and not a word of ours.
const Name = goplugin.GRPCServiceName
