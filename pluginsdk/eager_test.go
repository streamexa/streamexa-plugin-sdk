package pluginsdk

import (
	"context"
	"testing"
	"time"

	goplugin "github.com/hashicorp/go-plugin"

	"github.com/streamexa/streamexa-plugin-sdk/pluginpb"
	"github.com/streamexa/streamexa-plugin-sdk/transport"
)

// noActionPlugin's Run performs no host action, so it dials the host-action
// broker only if the SDK establishes the channel eagerly.
type noActionPlugin struct{}

func (noActionPlugin) Match(string) bool { return true }
func (noActionPlugin) Run(context.Context, API, Snapshot) (Result, error) {
	return Result{}, nil
}

// hostConnect reattaches to an in-process test-served plugin as the host would,
// returning the transport client (which exposes the plugin RPC client + broker).
func hostConnect(t *testing.T, rc *goplugin.ReattachConfig) *transport.Client {
	t.Helper()
	client := goplugin.NewClient(&goplugin.ClientConfig{
		HandshakeConfig:  transport.Handshake,
		Plugins:          transport.ClientPluginSet(),
		AllowedProtocols: []goplugin.Protocol{goplugin.ProtocolGRPC},
		Reattach:         rc,
	})
	t.Cleanup(client.Kill)
	rpcClient, err := client.Client()
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	raw, err := rpcClient.Dispense(transport.PluginName)
	if err != nil {
		t.Fatalf("dispense: %v", err)
	}
	return raw.(*transport.Client)
}

// In debug-serve mode (eager), the host-action broker channel is established
// before the plugin's Run body runs, so even a plugin that performs no action
// dials the broker; in production mode (lazy) it does not (capability
// go-plugin-runtime).
func TestEagerHostChannelDialsBeforeRunInDebugMode(t *testing.T) {
	cases := []struct {
		name     string
		eager    bool
		wantDial bool
	}{
		{"debug-serve eager dials without any action", true, true},
		{"production lazy does not dial without an action", false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			reCh := make(chan *goplugin.ReattachConfig, 1)
			closeCh := make(chan struct{})
			go goplugin.Serve(&goplugin.ServeConfig{
				HandshakeConfig: transport.Handshake,
				Plugins:         transport.ServerPluginSet(&grpcServer{impl: noActionPlugin{}, eager: c.eager}),
				GRPCServer:      goplugin.DefaultGRPCServer,
				Test:            &goplugin.ServeTestConfig{ReattachConfigCh: reCh, CloseCh: closeCh},
			})
			t.Cleanup(func() { close(closeCh) })

			var rc *goplugin.ReattachConfig
			select {
			case rc = <-reCh:
			case <-time.After(15 * time.Second):
				t.Fatal("plugin did not publish a reattach config")
			}

			conn := hostConnect(t, rc)

			// Register a broker listener the plugin will dial into if it establishes
			// the host-action channel, and detect that dial.
			broker := conn.Broker
			brokerID := broker.NextId()
			listener, err := broker.Accept(brokerID)
			if err != nil {
				t.Fatalf("broker accept: %v", err)
			}
			t.Cleanup(func() { _ = listener.Close() })
			dialed := make(chan struct{}, 1)
			go func() {
				if cn, err := listener.Accept(); err == nil {
					_ = cn.Close()
					dialed <- struct{}{}
				}
			}()

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if _, err := conn.Plugin.Run(ctx, &pluginpb.RunRequest{HostBrokerId: brokerID}); err != nil {
				t.Fatalf("Run: %v", err)
			}

			select {
			case <-dialed:
				if !c.wantDial {
					t.Error("broker was dialed for a no-action plugin, but eager=false (should stay lazy)")
				}
			case <-time.After(2 * time.Second):
				if c.wantDial {
					t.Error("broker was NOT dialed, but eager=true (should establish the channel before Run)")
				}
			}
		})
	}
}
