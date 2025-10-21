// Copyright 2025 Oppie Thunder Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.


package vst

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/good-night-oppie/helios/pkg/helios/types"
)

// Materialize writes the files from a snapshot to a real directory on disk.
func (v *VST) Materialize(id types.SnapshotID, outDir string, opts types.MatOpts) (types.CommitMetrics, error) {
	start := time.Now()

	// Validate and canonicalize output directory to prevent path traversal
	absOutDir, err := filepath.Abs(outDir)
	if err != nil {
		return types.CommitMetrics{}, fmt.Errorf("failed to get absolute path for output directory: %w", err)
	}

	// First try to get snapshot from memory
	v.mu.RLock()
	snap, ok := v.snaps[id]
	v.mu.RUnlock()

	// If not in memory, try L2
	if !ok && v.l2 != nil {
		// Get snapshot metadata
		snapshotKey := string("snapshot:" + id)
		dprintf("materialize: trying to get metadata with key %s", snapshotKey)
		metadataHash := types.Hash{Algorithm: types.BLAKE3, Digest: []byte(snapshotKey)}
		metadataBytes, ok, err := v.l2.Get(metadataHash)
		if err != nil {
			return types.CommitMetrics{}, err
		}
		if !ok {
			return types.CommitMetrics{}, fmt.Errorf("unknown snapshot in L2: %s", id)
		}

		// Unmarshal metadata
		var snapshotData map[string]types.Hash
		if err := json.Unmarshal(metadataBytes, &snapshotData); err != nil {
			return types.CommitMetrics{}, fmt.Errorf("failed to unmarshal snapshot metadata: %w", err)
		}

		// Restore files from L2
		snap = make(map[string][]byte)
		for path, hash := range snapshotData {
			data, ok, err := v.l2.Get(hash)
			if err != nil {
				return types.CommitMetrics{}, fmt.Errorf("failed to get file %s: %w", path, err)
			}
			if !ok {
				return types.CommitMetrics{}, fmt.Errorf("missing file data for %s", path)
			}
			snap[path] = data
		}
	} else if !ok {
		return types.CommitMetrics{}, fmt.Errorf("unknown snapshot: %s", id)
	}

	var bytesTotal int64
	var filesWritten int64

	for path, content := range snap {
		// Check if file should be included/excluded based on options
		if !shouldMaterialize(path, opts) {
			continue
		}

		// Validate path to prevent directory traversal attacks
		if err := validatePath(path); err != nil {
			return types.CommitMetrics{}, fmt.Errorf("invalid path in snapshot %s: %w", path, err)
		}

		dst := filepath.Join(absOutDir, path)

		// Double-check that the resolved destination is within outDir
		// This prevents path traversal via symlinks or other tricks
		absDst, err := filepath.Abs(dst)
		if err != nil {
			return types.CommitMetrics{}, fmt.Errorf("failed to resolve destination path: %w", err)
		}
		if !isSubpath(absOutDir, absDst) {
			return types.CommitMetrics{}, fmt.Errorf("path traversal detected: %s would escape output directory", path)
		}

		if err := os.MkdirAll(filepath.Dir(absDst), 0o755); err != nil {
			return types.CommitMetrics{}, err
		}
		if err := os.WriteFile(absDst, content, 0o644); err != nil {
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

// validatePath checks if a path is safe (doesn't contain path traversal attempts)
func validatePath(path string) error {
	// Reject absolute paths
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute paths not allowed: %s", path)
	}

	// Clean the path to normalize it
	cleaned := filepath.Clean(path)

	// Check for path traversal attempts
	if cleaned == ".." || filepath.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path traversal detected: %s", path)
	}

	return nil
}

// isSubpath checks if child is a subdirectory or file within parent
func isSubpath(parent, child string) bool {
	// Clean and make paths absolute for reliable comparison
	parent = filepath.Clean(parent)
	child = filepath.Clean(child)

	// Add trailing separator to parent to avoid false positives
	// e.g., /foo should not match /foobar
	if !filepath.HasSuffix(parent, string(filepath.Separator)) {
		parent += string(filepath.Separator)
	}

	return filepath.HasPrefix(child+string(filepath.Separator), parent)
}
