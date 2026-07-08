package pluginsdk

import (
	"encoding/json"
	"fmt"
	"os"
)

// ReattachHandshake is the on-disk debug reattach handshake a plugin publishes
// in --debug mode (see Serve) so the host can reattach to the already-running,
// debugger-launched process instead of launching its own subprocess. It is plain
// data — it carries no go-plugin type. This format is the SDK↔host contract for
// reattach debugging; the SDK owns it as the single source of truth. The SDK
// writes it from a go-plugin ReattachConfig; the host reads it and rebuilds the
// ReattachConfig on its side.
type ReattachHandshake struct {
	Protocol        string `json:"protocol"`         // e.g. "grpc"
	ProtocolVersion int    `json:"protocol_version"` // negotiated protocol version
	Network         string `json:"network"`          // net.Addr.Network(), e.g. "unix"/"tcp"
	Address         string `json:"address"`          // net.Addr.String()
	Pid             int    `json:"pid"`              // the serving process's pid
}

// WriteReattachHandshake serializes h to path as JSON.
func WriteReattachHandshake(path string, h ReattachHandshake) error {
	b, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// ReadReattachHandshake parses the JSON reattach handshake at path. A missing
// file returns the os error (check with os.IsNotExist); malformed content
// returns a parse error naming the path.
func ReadReattachHandshake(path string) (ReattachHandshake, error) {
	var h ReattachHandshake
	b, err := os.ReadFile(path)
	if err != nil {
		return h, err
	}
	if err := json.Unmarshal(b, &h); err != nil {
		return h, fmt.Errorf("parse reattach handshake %s: %w", path, err)
	}
	return h, nil
}
