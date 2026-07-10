// Package m3u8 provides pure HLS manifest parsing shared by plugin authors and
// the analyzer host: it computes a media playlist's duration, detects a live
// stream, and parses a master playlist's variant streams. It is the single
// source of truth for generic HLS parsing (capability plugin-m3u8-utils).
//
// It carries no proprietary classification (ad-network patterns, ad/video
// heuristics) — those live only in the host's internal analysis.
//
// Every function is pure: it depends only on the manifest text passed in, with
// no host round-trip, network, or browser context.
package m3u8

import (
	"regexp"
	"strconv"
	"strings"
)

const (
	tagStreamInf = "#EXT-X-STREAM-INF"
	tagEndList   = "#EXT-X-ENDLIST"
	tagExtInf    = "#EXTINF:"
)

var (
	extInfRE     = regexp.MustCompile(`#EXTINF:\s*([0-9.]+)`)
	resolutionRE = regexp.MustCompile(`^(\d+)x(\d+)$`)
)

// Variant is one entry of a master playlist's #EXT-X-STREAM-INF: its declared
// resolution (both the raw "WxH" string and the parsed width/height), peak
// BANDWIDTH, CODECS, and the stream URI on the following line. A field is the
// zero value when its attribute is absent from the variant tag.
type Variant struct {
	Resolution string
	Width      int
	Height     int
	Bandwidth  int
	Codecs     string
	URI        string
}

// ParseMediaPlaylistDuration sums the #EXTINF segment durations of a media
// playlist. A master playlist (containing #EXT-X-STREAM-INF) has no direct
// duration and returns 0, as does empty input.
func ParseMediaPlaylistDuration(content string) float64 {
	if content == "" {
		return 0
	}
	if strings.Contains(content, tagStreamInf) {
		return 0
	}
	var total float64
	for _, m := range extInfRE.FindAllStringSubmatch(content, -1) {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil {
			total += v
		}
	}
	return total
}

// IsLiveStream reports whether a media playlist is a live stream: it has
// segments (#EXTINF) but is neither a master playlist (#EXT-X-STREAM-INF) nor a
// finished VOD (#EXT-X-ENDLIST). Empty input is not live.
func IsLiveStream(content string) bool {
	if content == "" {
		return false
	}
	if strings.Contains(content, tagStreamInf) {
		return false
	}
	if strings.Contains(content, tagEndList) {
		return false
	}
	return strings.Contains(content, tagExtInf)
}

// ParseMasterVariants parses the variant streams of a master playlist. Each
// #EXT-X-STREAM-INF tag pairs with the next non-comment, non-empty line as its
// URI. A playlist with no #EXT-X-STREAM-INF tags (a media playlist or empty
// input) yields no variants.
func ParseMasterVariants(content string) []Variant {
	if content == "" {
		return nil
	}
	lines := strings.Split(content, "\n")
	var variants []Variant
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(line, tagStreamInf+":") {
			continue
		}
		v := parseStreamInf(strings.TrimPrefix(line, tagStreamInf+":"))
		if uri, ok := nextURI(lines, i+1); ok {
			v.URI = uri
		}
		variants = append(variants, v)
	}
	return variants
}

// nextURI returns the first non-comment, non-empty line at or after start — the
// stream URI that follows an #EXT-X-STREAM-INF tag.
func nextURI(lines []string, start int) (string, bool) {
	for j := start; j < len(lines); j++ {
		candidate := strings.TrimSpace(lines[j])
		if candidate == "" || strings.HasPrefix(candidate, "#") {
			continue
		}
		return candidate, true
	}
	return "", false
}

// parseStreamInf parses the attribute list of an #EXT-X-STREAM-INF tag into a
// Variant, leaving URI unset.
func parseStreamInf(attrs string) Variant {
	var v Variant
	for _, attr := range splitAttributes(attrs) {
		key, value, ok := strings.Cut(attr, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"`)
		switch strings.ToUpper(key) {
		case "RESOLUTION":
			v.Resolution = value
			if m := resolutionRE.FindStringSubmatch(value); m != nil {
				v.Width, _ = strconv.Atoi(m[1])
				v.Height, _ = strconv.Atoi(m[2])
			}
		case "BANDWIDTH":
			v.Bandwidth, _ = strconv.Atoi(value)
		case "CODECS":
			v.Codecs = value
		}
	}
	return v
}

// splitAttributes splits a comma-separated HLS attribute list, treating commas
// inside double quotes (as in CODECS="avc1.4d401f,mp4a.40.2") as literal, not
// separators.
func splitAttributes(attrs string) []string {
	var out []string
	var start int
	inQuotes := false
	for i, r := range attrs {
		switch r {
		case '"':
			inQuotes = !inQuotes
		case ',':
			if !inQuotes {
				out = append(out, attrs[start:i])
				start = i + 1
			}
		}
	}
	out = append(out, attrs[start:])
	return out
}
