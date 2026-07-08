package pluginsdk

import (
	"context"
	"fmt"
	"sync"

	goplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"github.com/lanyitin/streamexa-plugin-sdk/pluginpb"
	"github.com/lanyitin/streamexa-plugin-sdk/transport"
)

// Serve is the plugin entry point: a plugin author's main() calls this with
// their Plugin implementation. It performs the go-plugin handshake and gRPC
// registration; the author writes no go-plugin or gRPC code.
func Serve(p Plugin) {
	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: transport.Handshake,
		Plugins:         transport.ServerPluginSet(&grpcServer{impl: p}),
		GRPCServer:      goplugin.DefaultGRPCServer,
	})
}

// grpcServer adapts the author's Plugin to the wire PluginService and bridges
// the reverse HostService channel into the SDK's API.
type grpcServer struct {
	pluginpb.UnimplementedPluginServiceServer
	impl   Plugin
	broker *goplugin.GRPCBroker
}

// SetBroker satisfies transport.BrokerAware.
func (s *grpcServer) SetBroker(b *goplugin.GRPCBroker) { s.broker = b }

func (s *grpcServer) Run(ctx context.Context, req *pluginpb.RunRequest) (*pluginpb.RunResponse, error) {
	api := &apiClient{ctx: ctx, broker: s.broker, brokerID: req.GetHostBrokerId()}
	defer api.close()

	res, err := s.impl.Run(ctx, api, snapshotFromProto(req.GetSnapshot()))
	if err != nil {
		return nil, err
	}
	return &pluginpb.RunResponse{Result: resultToProto(res)}, nil
}

// apiClient implements API by calling the host's reverse HostService. It dials
// the broker lazily on first use, so a plugin that performs no actions needs no
// reverse channel.
type apiClient struct {
	ctx      context.Context
	broker   *goplugin.GRPCBroker
	brokerID uint32

	once sync.Once
	host pluginpb.HostServiceClient
	conn *grpc.ClientConn
	err  error
}

func (a *apiClient) client() (pluginpb.HostServiceClient, error) {
	a.once.Do(func() {
		if a.broker == nil || a.brokerID == 0 {
			a.err = fmt.Errorf("host action channel is not available")
			return
		}
		conn, err := a.broker.Dial(a.brokerID)
		if err != nil {
			a.err = fmt.Errorf("dial host action channel: %w", err)
			return
		}
		a.conn = conn
		a.host = pluginpb.NewHostServiceClient(conn)
	})
	return a.host, a.err
}

func (a *apiClient) close() {
	if a.conn != nil {
		_ = a.conn.Close()
	}
}

func (a *apiClient) GetResponseBody(requestID string) (string, error) {
	h, err := a.client()
	if err != nil {
		return "", err
	}
	resp, err := h.GetResponseBody(a.ctx, &pluginpb.GetResponseBodyRequest{RequestId: requestID})
	if err != nil {
		return "", err
	}
	if !resp.GetFound() {
		return "", fmt.Errorf("response body not found for request id %q", requestID)
	}
	return resp.GetBody(), nil
}

func (a *apiClient) Click(selector string) error {
	h, err := a.client()
	if err != nil {
		return err
	}
	resp, err := h.Click(a.ctx, &pluginpb.ClickRequest{Selector: selector})
	if err != nil {
		return err
	}
	if !resp.GetOk() {
		return fmt.Errorf("click failed: %s", resp.GetError())
	}
	return nil
}

func (a *apiClient) WaitForSelector(selector string) error {
	h, err := a.client()
	if err != nil {
		return err
	}
	resp, err := h.WaitFor(a.ctx, &pluginpb.WaitForRequest{Selector: selector})
	if err != nil {
		return err
	}
	if !resp.GetOk() {
		return fmt.Errorf("wait failed: %s", resp.GetError())
	}
	return nil
}

func (a *apiClient) WaitForTimeout(ms int) error {
	h, err := a.client()
	if err != nil {
		return err
	}
	resp, err := h.WaitFor(a.ctx, &pluginpb.WaitForRequest{TimeoutMs: int64(ms)})
	if err != nil {
		return err
	}
	if !resp.GetOk() {
		return fmt.Errorf("wait failed: %s", resp.GetError())
	}
	return nil
}

func (a *apiClient) PlayVideos() (int, error) {
	h, err := a.client()
	if err != nil {
		return 0, err
	}
	resp, err := h.PlayVideos(a.ctx, &pluginpb.PlayVideosRequest{})
	if err != nil {
		return 0, err
	}
	if !resp.GetOk() {
		return int(resp.GetClicked()), fmt.Errorf("play videos failed: %s", resp.GetError())
	}
	return int(resp.GetClicked()), nil
}

func (a *apiClient) Snapshot() (Snapshot, error) {
	h, err := a.client()
	if err != nil {
		return Snapshot{}, err
	}
	resp, err := h.Snapshot(a.ctx, &pluginpb.SnapshotRequest{})
	if err != nil {
		return Snapshot{}, err
	}
	return snapshotFromProto(resp.GetSnapshot()), nil
}

func (a *apiClient) Fetch(req FetchRequest) (FetchResponse, error) {
	h, err := a.client()
	if err != nil {
		return FetchResponse{}, err
	}
	resp, err := h.Fetch(a.ctx, &pluginpb.FetchRequest{
		Url:     req.URL,
		Method:  req.Method,
		Headers: req.Headers,
		Body:    req.Body,
	})
	if err != nil {
		return FetchResponse{}, err
	}
	return FetchResponse{Status: int(resp.GetStatus()), Headers: resp.GetHeaders(), Body: resp.GetBody()}, nil
}

func (a *apiClient) Log(level, message string) {
	h, err := a.client()
	if err != nil {
		return
	}
	_, _ = h.Log(a.ctx, &pluginpb.LogRequest{Level: level, Message: message})
}
