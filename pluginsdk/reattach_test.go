package pluginsdk

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestReattachHandshakeRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.reattach")
	want := ReattachHandshake{Protocol: "grpc", ProtocolVersion: 1, Network: "unix", Address: "/tmp/plugin123.sock", Pid: 4242}
	if err := WriteReattachHandshake(path, want); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := ReadReattachHandshake(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got != want {
		t.Errorf("round-trip = %+v, want %+v", got, want)
	}
}

func TestReadReattachHandshakeMalformed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.reattach")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadReattachHandshake(path); err == nil {
		t.Error("expected a parse error on malformed JSON")
	}
}

func TestReadReattachHandshakeMissing(t *testing.T) {
	_, err := ReadReattachHandshake(filepath.Join(t.TempDir(), "nope.reattach"))
	if !os.IsNotExist(err) {
		t.Errorf("expected os.IsNotExist error for a missing file, got %v", err)
	}
}

func TestParseDebugArgs(t *testing.T) {
	cases := []struct {
		args      []string
		wantDebug bool
		wantFile  string
	}{
		{nil, false, ""}, // host launch: no args
		{[]string{"--debug", "--reattach-file=/tmp/a.reattach"}, true, "/tmp/a.reattach"},
		{[]string{"-debug", "-reattach-file", "/tmp/b.reattach"}, true, "/tmp/b.reattach"},
		{[]string{"--reattach-file=/tmp/c.reattach"}, false, "/tmp/c.reattach"}, // file without --debug: not debug
		{[]string{"--unknown", "--debug"}, true, ""},                            // unknown args ignored
	}
	for _, c := range cases {
		gotDebug, gotFile := parseDebugArgs(c.args)
		if gotDebug != c.wantDebug || gotFile != c.wantFile {
			t.Errorf("parseDebugArgs(%v) = (%v, %q), want (%v, %q)", c.args, gotDebug, gotFile, c.wantDebug, c.wantFile)
		}
	}
}

type debugTestPlugin struct{}

func (debugTestPlugin) Match(string) bool { return true }
func (debugTestPlugin) Run(context.Context, API, Snapshot) (Result, error) {
	return Result{}, nil
}

// startDebugServe serves the plugin in-process and writes a complete, parseable
// reattach handshake — the SDK side of the debug-reattach contract.
func TestStartDebugServeWritesParseableHandshake(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.reattach")
	stop, err := startDebugServe(debugTestPlugin{}, path)
	if err != nil {
		t.Fatalf("startDebugServe: %v", err)
	}
	defer stop()

	h, err := ReadReattachHandshake(path)
	if err != nil {
		t.Fatalf("read handshake: %v", err)
	}
	if h.Protocol == "" || h.Network == "" || h.Address == "" || h.Pid == 0 {
		t.Errorf("incomplete reattach handshake: %+v", h)
	}
	if h.Pid != os.Getpid() {
		t.Errorf("handshake pid = %d, want this process %d", h.Pid, os.Getpid())
	}
}
