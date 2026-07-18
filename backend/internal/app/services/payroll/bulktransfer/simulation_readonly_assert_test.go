package bulktransfer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSimulationSourceHasNoWriteCalls is the canonical read-only enforcement
// for the settlement simulation. It greps the simulation source files for any
// forbidden write-call substring and fails if one is found.
//
// This is stronger than relying on sql.TxOptions{ReadOnly:true} (which the
// configured MySQL driver silently ignores — see red-team Finding 9). The
// primary read-only guarantee is structural: SimulationService only calls
// ExportPlanner.Plan() (pure by Phase 1 invariant) and the read-only
// GetAccountTotalInRange. This test keeps that invariant honest as the
// codebase evolves.
//
// If you legitimately need to add a write call to the simulation, you're
// doing something wrong — the simulation MUST NOT mutate. Reconsider the
// design before editing this test.
func TestSimulationSourceHasNoWriteCalls(t *testing.T) {
	files := []string{
		"simulation_service.go",
		"simulation_validators.go", // may not exist yet — skipped if absent
	}
	// Substrings whose presence in the simulation source would indicate a
	// write call. Each is the receiver/method of a write op the simulation
	// must never invoke.
	forbidden := []string{
		".Create(",
		".CreateBatch(",
		".CreateInBatches(",
		".Update(",
		".UpdateFileIDByCodes(",
		".BulkUpdate",
		".Delete",
		".Save(",
		".Publish(",
		"eventBus",
		"fileRepo",
		"transactionCodeRepo",
		"MarkReconciled",
		"MarkSettled",
	}

	for _, name := range files {
		path := filepath.Join(".", name)
		bytes, err := os.ReadFile(path)
		if err != nil {
			// Optional file (e.g. simulation_validators.go not yet created).
			continue
		}
		src := string(bytes)
		// Strip line comments and block comments so commented-out examples
		// (like the audit publish reference in Export()) don't trigger false
		// positives. Crude but sufficient.
		src = stripComments(src)

		for _, sub := range forbidden {
			if idx := strings.Index(src, sub); idx >= 0 {
				// Compute line number for the error message.
				line := 1 + strings.Count(src[:idx], "\n")
				t.Errorf("FORBIDDEN write-call %q found in %s at line %d — simulation MUST be read-only", sub, name, line)
			}
		}
	}
}

// stripComments removes // line comments and /* */ block comments from src.
// Crude — does not handle strings containing comment-like sequences, which is
// fine here because the simulation source doesn't contain such strings.
func stripComments(src string) string {
	var out strings.Builder
	i := 0
	for i < len(src) {
		// Line comment
		if i+1 < len(src) && src[i] == '/' && src[i+1] == '/' {
			// Skip to end of line.
			for i < len(src) && src[i] != '\n' {
				i++
			}
			continue
		}
		// Block comment
		if i+1 < len(src) && src[i] == '/' && src[i+1] == '*' {
			i += 2
			for i+1 < len(src) && !(src[i] == '*' && src[i+1] == '/') {
				i++
			}
			i += 2
			continue
		}
		out.WriteByte(src[i])
		i++
	}
	return out.String()
}
