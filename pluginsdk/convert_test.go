package pluginsdk

import "github.com/streamexa/streamexa-plugin-sdk/pluginpb"

import "testing"

// snapshotFromProto must carry the response headers and the inlined manifest
// body added for capability plugin-data-snapshot, so a plugin reads them
// directly off the snapshot without a host round-trip.
func TestSnapshotFromProtoCarriesResponseData(t *testing.T) {
	in := &pluginpb.PageSnapshot{
		Requests: []*pluginpb.CapturedRequest{
			{
				Id:              "r0",
				Url:             "https://cdn.example.com/master.m3u8",
				ResponseHeaders: map[string]string{"Content-Type": "application/vnd.apple.mpegurl"},
				Body:            "#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=1\nlo.m3u8\n",
			},
			{Id: "r1", Url: "https://cdn.example.com/seg.ts"},
		},
	}
	got := snapshotFromProto(in)
	if len(got.Requests) != 2 {
		t.Fatalf("got %d requests, want 2", len(got.Requests))
	}
	manifest := got.Requests[0]
	if manifest.ResponseHeaders["Content-Type"] != "application/vnd.apple.mpegurl" {
		t.Errorf("manifest ResponseHeaders = %v, want Content-Type set", manifest.ResponseHeaders)
	}
	if manifest.Body == "" {
		t.Errorf("manifest Body is empty, want inlined manifest content")
	}
	if seg := got.Requests[1]; seg.Body != "" {
		t.Errorf("segment Body = %q, want empty", seg.Body)
	}
}
