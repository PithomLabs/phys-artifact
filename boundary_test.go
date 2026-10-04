package artifact

import (
	"os/exec"
	"strings"
	"testing"
)

// Plan9.3 §4 / gate [3]: phys-artifact imports stdlib only.
// Mechanically enforced via `go list -deps`: every dependency must be
// standard library (no dot in the first path element) or this module itself.
func TestBoundaryStdlibOnly(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", ".").Output()
	if err != nil {
		t.Fatalf("go list -deps: %v", err)
	}
	for _, dep := range strings.Fields(string(out)) {
		if strings.HasPrefix(dep, "github.com/PithomLabs/phys-artifact") {
			continue
		}
		if strings.Contains(strings.Split(dep, "/")[0], ".") {
			t.Fatalf("non-stdlib dependency: %s", dep)
		}
	}
}
