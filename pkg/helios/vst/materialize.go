package vst

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/good-night-oppie/helios-engine/pkg/helios/types"
)

// Materialize writes the files from a snapshot to a real directory on disk.
func (v *VST) Materialize(id types.SnapshotID, outDir string, opts types.MatOpts) (types.CommitMetrics, error) {
	start := time.Now()
	snap, ok := v.snaps[id]
	if !ok {
		return types.CommitMetrics{}, fmt.Errorf("unknown snapshot: %s", id)
	}

	var bytesTotal int64
	var filesWritten int64

	for path, content := range snap {
		// Check if file should be included/excluded based on options
		if !shouldMaterialize(path, opts) {
			continue
		}

		dst := filepath.Join(outDir, path)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return types.CommitMetrics{}, err
		}
		if err := os.WriteFile(dst, content, 0o644); err != nil {
			return types.CommitMetrics{}, err
		}
		bytesTotal += int64(len(content))
		filesWritten++
	}

	return types.CommitMetrics{
		CommitLatency: time.Since(start),
		NewObjects:    filesWritten,
		NewBytes:      bytesTotal,
	}, nil
}

// shouldMaterialize checks if a file path should be materialized based on include/exclude patterns
func shouldMaterialize(path string, opts types.MatOpts) bool {
	// If Include patterns are specified, the path must match at least one
	if len(opts.Include) > 0 {
		matched := false
		for _, pattern := range opts.Include {
			if matchGlob(path, pattern) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// If Exclude patterns are specified, the path must not match any
	if len(opts.Exclude) > 0 {
		for _, pattern := range opts.Exclude {
			if matchGlob(path, pattern) {
				return false
			}
		}
	}

	return true
}

// matchGlob checks if a path matches a glob pattern
// Supports simple patterns like "src/**" or "*.go"
func matchGlob(path, pattern string) bool {
	// Handle ** for recursive matching
	if len(pattern) > 2 && pattern[len(pattern)-2:] == "**" {
		prefix := pattern[:len(pattern)-2]
		return len(path) >= len(prefix) && path[:len(prefix)] == prefix
	}

	// Use filepath.Match for other patterns
	matched, _ := filepath.Match(pattern, path)
	return matched
}
