// Package transport holds the go-plugin wiring shared by the analyzer host
// (the go-plugin client) and the plugin SDK (the go-plugin server): the
// handshake config, the gRPC plugin adapter, and the client wrapper. Keeping it
// separate lets both sides depend on one contract without the SDK importing
// host internals. See capability go-plugin-runtime.
package transport

import (
	"context"

	goplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"github.com/lanyitin/streamexa-plugin-sdk/pluginpb"
)

// PluginName is the dispense key both sides use for the single plugin service.
const PluginName = "streamexa_plugin"

// Handshake gates plugin launches: a process without the matching magic cookie
// in its environment exits with a human-readable notice instead of hanging.
var Handshake = goplugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "STREAMEXA_PLUGIN",
	MagicCookieValue: "streamexa-go-plugin-v1",
}

// GoPlugin adapts our gRPC PluginService to go-plugin. On the server (plugin)
// side Impl is the plugin's PluginService implementation; on the client (host)
// side Impl is nil and GRPCClient returns a Client wrapper.
type GoPlugin struct {
	goplugin.NetRPCUnsupportedPlugin
	Impl pluginpb.PluginServiceServer
}

// BrokerAware is implemented by a plugin server impl that needs the go-plugin
// broker (to dial the host's reverse HostService channel). GRPCServer injects
// the broker before registering the service.
type BrokerAware interface {
	SetBroker(*goplugin.GRPCBroker)
}

// GRPCServer registers the plugin's PluginService on the served gRPC server,
// first handing the broker to a BrokerAware impl.
func (p *GoPlugin) GRPCServer(broker *goplugin.GRPCBroker, s *grpc.Server) error {
	if ba, ok := p.Impl.(BrokerAware); ok {
		ba.SetBroker(broker)
	}
	pluginpb.RegisterPluginServiceServer(s, p.Impl)
	return nil
}

// GRPCClient returns a Client carrying the PluginService client and the broker
// used to serve the reverse HostService.
func (p *GoPlugin) GRPCClient(_ context.Context, broker *goplugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &Client{Plugin: pluginpb.NewPluginServiceClient(c), Broker: broker}, nil
}

// Client is what the host dispenses: the plugin's Run client plus the go-plugin
// broker for serving the reverse HostService channel.
type Client struct {
	Plugin pluginpb.PluginServiceClient
	Broker *goplugin.GRPCBroker
}

// ClientPluginSet is the go-plugin plugin map for the host (client) side.
func ClientPluginSet() map[string]goplugin.Plugin {
	return map[string]goplugin.Plugin{PluginName: &GoPlugin{}}
}

// ServerPluginSet is the go-plugin plugin map for the plugin (server) side,
// binding the plugin's PluginService implementation.
func ServerPluginSet(impl pluginpb.PluginServiceServer) map[string]goplugin.Plugin {
	return map[string]goplugin.Plugin{PluginName: &GoPlugin{Impl: impl}}
}
