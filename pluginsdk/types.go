// Package pluginsdk is the Go plugin SDK for the streamexa analyzer host. A
// third-party plugin author implements the Plugin interface and calls Serve
// from main(); the SDK performs the go-plugin handshake and gRPC wiring. The
// SDK surface exposes only plain data types and a mediated action API — never a
// go-plugin, gRPC, or raw browser (CDP/Playwright) type. See capability
// go-plugin-runtime.
//
// Isolation boundary: a plugin process is arbitrary third-party code. The
// isolation this system provides covers browser and page-data access only
// (mediated action whitelist, no raw CDP, redacted credentials). It is NOT
// OS-level sandboxing of the plugin process's own filesystem or network access.
package pluginsdk

import "context"

// Plugin is the single interface a plugin author implements.
type Plugin interface {
	// Match reports whether this plugin handles the target URL. The host also
	// pre-filters by the manifest's urlPatterns before launching; keep the two
	// consistent.
	Match(targetURL string) bool
	// Run receives the read-only snapshot and a mediated action API, and returns
	// the plugin's contribution. Returning an empty Result is a valid
	// zero-contribution.
	Run(ctx context.Context, api API, snapshot Snapshot) (Result, error)
}

// API is the host-mediated action whitelist available to a plugin during Run.
// It exposes no browser object — only capabilities.
type API interface {
	// GetResponseBody returns the response body for a request id present in the
	// snapshot, or an error for an unknown/expired id.
	GetResponseBody(requestID string) (string, error)
	// Click clicks the first element matching selector.
	Click(selector string) error
	// WaitForSelector waits until an element matching selector is present.
	WaitForSelector(selector string) error
	// WaitForTimeout waits for a fixed number of milliseconds.
	WaitForTimeout(ms int) error
	// PlayVideos clicks every <video> across all frames (including cross-origin
	// iframes) to trigger click-gated players, returning how many were clicked.
	// No video / all clicks failing is 0, nil — not an error.
	PlayVideos() (int, error)
	// Snapshot re-captures the DOM and any requests completed since the last
	// snapshot, returning a refreshed read-only snapshot.
	Snapshot() (Snapshot, error)
	// Fetch issues an HTTP request with the current browser session applied by
	// the host; the plugin never receives the session credentials.
	Fetch(req FetchRequest) (FetchResponse, error)
	// Log emits a diagnostic line through the host.
	Log(level, message string)
}

// Snapshot is a read-only view of the current CDP context. It carries no
// response bodies (use API.GetResponseBody).
type Snapshot struct {
	PageDOM       string
	IFrames       []IFrame
	Requests      []Request
	StreamResults []Source
}

// IFrame is one iframe's post-render DOM keyed by its frame URL.
type IFrame struct {
	FrameURL string
	DOM      string
}

// Request is one completed request with redacted headers plus native
// classification the host computed.
type Request struct {
	ID            string
	Method        string
	URL           string
	Status        int
	ResourceType  string
	Headers       map[string]string
	MediaType     string
	VideoDuration float64
	IsAd          bool
	IsLiveStream  bool
}

// Source mirrors the host's analyzed-source shape.
type Source struct {
	URL            string
	RequestHeaders map[string]string
	MimeType       string
	VideoDuration  float64
	EstimatedBytes int64
}

// Task is an optional composed download task.
type Task struct {
	Sources []Source
	Labels  map[string]string
}

// Result is exactly {Sources, Task?, Metadata?}.
type Result struct {
	Sources  []Source
	Task     *Task
	Metadata map[string]string
}

// FetchRequest is a host-mediated HTTP request.
type FetchRequest struct {
	URL     string
	Method  string
	Headers map[string]string
	Body    string
}

// FetchResponse is the result of a mediated fetch.
type FetchResponse struct {
	Status  int
	Headers map[string]string
	Body    string
}
