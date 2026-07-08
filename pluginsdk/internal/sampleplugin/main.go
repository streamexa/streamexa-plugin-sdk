// Command sampleplugin is a minimal plugin built ONLY against the public SDK
// surface (no go-plugin, gRPC, or browser imports). The host's SDK test launches
// it to confirm a plugin authored against the SDK is dispatched and its Run is
// invoked with the snapshot.
package main

import (
	"context"
	"strconv"

	"github.com/lanyitin/streamexa-plugin-sdk/pluginsdk"
)

type plugin struct{}

func (plugin) Match(string) bool { return true }

func (plugin) Run(_ context.Context, _ pluginsdk.API, snap pluginsdk.Snapshot) (pluginsdk.Result, error) {
	// Echo how many requests the snapshot carried so the host can confirm the
	// snapshot reached Run, and return one source derived from the page DOM.
	return pluginsdk.Result{
		Sources:  []pluginsdk.Source{{URL: "https://example.com/from-sample.m3u8", MimeType: "application/vnd.apple.mpegurl"}},
		Metadata: map[string]string{"seen_requests": strconv.Itoa(len(snap.Requests))},
	}, nil
}

func main() {
	pluginsdk.Serve(plugin{})
}
