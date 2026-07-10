package m3u8

import "testing"

// The duration and live/vod cases mirror the spec examples in
// openspec/specs/plugin-m3u8-utils: media playlist duration sums #EXTINF, a
// master playlist has no direct duration, and live is "has segments, no
// ENDLIST, not a master".
func TestParseMediaPlaylistDurationAndLive(t *testing.T) {
	vod := "#EXTM3U\n#EXTINF:10.0,\na.ts\n#EXTINF:10.0,\nb.ts\n#EXTINF:10.0,\nc.ts\n#EXT-X-ENDLIST\n"
	live := "#EXTM3U\n#EXTINF:6.0,\na.ts\n#EXTINF:6.0,\nb.ts\n"
	master := "#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=2000000,RESOLUTION=1920x1080\nhi.m3u8\n"

	cases := []struct {
		name     string
		content  string
		wantDur  float64
		wantLive bool
	}{
		{"vod media playlist with endlist", vod, 30.0, false},
		{"live media playlist without endlist", live, 12.0, true},
		{"master playlist", master, 0, false},
		{"empty string", "", 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ParseMediaPlaylistDuration(c.content); got != c.wantDur {
				t.Errorf("ParseMediaPlaylistDuration = %v, want %v", got, c.wantDur)
			}
			if got := IsLiveStream(c.content); got != c.wantLive {
				t.Errorf("IsLiveStream = %v, want %v", got, c.wantLive)
			}
		})
	}
}

// ParseMasterVariants mirrors the spec's two-variant example: each variant
// carries its RESOLUTION (width/height), BANDWIDTH, CODECS (when present) and
// the stream URI on the following line. CODECS is quoted and itself contains a
// comma, so the attribute parser must respect quotes.
func TestParseMasterVariants(t *testing.T) {
	master := "#EXTM3U\n" +
		`#EXT-X-STREAM-INF:BANDWIDTH=2000000,RESOLUTION=1920x1080,CODECS="avc1.640028,mp4a.40.2"` + "\n" +
		"hi.m3u8\n" +
		"#EXT-X-STREAM-INF:BANDWIDTH=800000,RESOLUTION=1280x720\n" +
		"lo.m3u8\n"

	got := ParseMasterVariants(master)
	if len(got) != 2 {
		t.Fatalf("ParseMasterVariants returned %d variants, want 2", len(got))
	}

	v1 := got[0]
	if v1.Width != 1920 || v1.Height != 1080 {
		t.Errorf("variant 1 resolution = %dx%d, want 1920x1080", v1.Width, v1.Height)
	}
	if v1.Resolution != "1920x1080" {
		t.Errorf("variant 1 Resolution = %q, want %q", v1.Resolution, "1920x1080")
	}
	if v1.Bandwidth != 2000000 {
		t.Errorf("variant 1 Bandwidth = %d, want 2000000", v1.Bandwidth)
	}
	if v1.Codecs != "avc1.640028,mp4a.40.2" {
		t.Errorf("variant 1 Codecs = %q, want %q", v1.Codecs, "avc1.640028,mp4a.40.2")
	}
	if v1.URI != "hi.m3u8" {
		t.Errorf("variant 1 URI = %q, want hi.m3u8", v1.URI)
	}

	v2 := got[1]
	if v2.Width != 1280 || v2.Height != 720 {
		t.Errorf("variant 2 resolution = %dx%d, want 1280x720", v2.Width, v2.Height)
	}
	if v2.Bandwidth != 800000 {
		t.Errorf("variant 2 Bandwidth = %d, want 800000", v2.Bandwidth)
	}
	if v2.URI != "lo.m3u8" {
		t.Errorf("variant 2 URI = %q, want lo.m3u8", v2.URI)
	}
}

// A non-master playlist yields no variants.
func TestParseMasterVariantsOnMediaPlaylist(t *testing.T) {
	media := "#EXTM3U\n#EXTINF:10.0,\na.ts\n#EXT-X-ENDLIST\n"
	if got := ParseMasterVariants(media); len(got) != 0 {
		t.Errorf("ParseMasterVariants on media playlist = %v, want empty", got)
	}
}
