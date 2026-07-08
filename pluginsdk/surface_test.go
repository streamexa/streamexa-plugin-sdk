package pluginsdk

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The SDK's surface must not drag in any raw browser (CDP/Playwright) type.
// We assert no SDK source file imports a browser-driver package; the public
// types (types.go) already use only plain data types.
func TestSDKImportsNoBrowserDriver(t *testing.T) {
	forbidden := []string{
		"github.com/go-rod/rod",
		"github.com/chromedp/chromedp",
		"playwright",
		"internal/analyzer/capture", // the host's CDP capture layer
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(".", e.Name()), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", e.Name(), err)
		}
		for _, imp := range f.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			for _, bad := range forbidden {
				if strings.Contains(path, bad) {
					t.Errorf("%s imports forbidden browser package %q via %q", e.Name(), bad, path)
				}
			}
		}
	}
}

// The exported API interface must expose exactly the whitelisted capabilities,
// none of which name a browser type.
func TestAPIInterfaceIsCapabilityOnly(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "types.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var apiMethods []string
	ast.Inspect(f, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok || ts.Name.Name != "API" {
			return true
		}
		it, ok := ts.Type.(*ast.InterfaceType)
		if !ok {
			return true
		}
		for _, m := range it.Methods.List {
			for _, name := range m.Names {
				apiMethods = append(apiMethods, name.Name)
			}
		}
		return false
	})
	want := map[string]bool{
		"GetResponseBody": true, "Click": true, "WaitForSelector": true,
		"WaitForTimeout": true, "PlayVideos": true, "Snapshot": true,
		"Fetch": true, "Log": true,
	}
	if len(apiMethods) != len(want) {
		t.Errorf("API methods = %v, want the %d whitelisted capabilities", apiMethods, len(want))
	}
	for _, m := range apiMethods {
		if !want[m] {
			t.Errorf("API exposes unexpected method %q", m)
		}
	}
}
