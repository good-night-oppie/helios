package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/good-night-oppie/helios-engine/internal/metrics"
	"github.com/good-night-oppie/helios-engine/pkg/helios/l1cache"
	"github.com/good-night-oppie/helios-engine/pkg/helios/objstore"
	"github.com/good-night-oppie/helios-engine/pkg/helios/types"
	"github.com/good-night-oppie/helios-engine/pkg/helios/vst"
	"github.com/good-night-oppie/helios-engine/pkg/cli"
)

// Engine interface for testability
type Engine interface {
	AttachStores(l1cache.Cache, objstore.Store)
	WriteFile(path string, content []byte) error
	Commit(msg string) (types.SnapshotID, types.CommitMetrics, error)
	Restore(id types.SnapshotID) error
	Diff(from, to types.SnapshotID) (types.DiffStats, error)
	Materialize(id types.SnapshotID, outDir string, opts types.MatOpts) (types.CommitMetrics, error)
	L1Stats() l1cache.CacheStats
	EngineMetricsSnapshot() metrics.Snapshot
}

// Config holds dependencies for CLI handlers
type Config struct {
	EngineFactory func() (Engine, error)
}

// MatOpts for materialize command
type MatOpts struct {
	Include []string
	Exclude []string
}

// HandleCommit processes commit command
func HandleCommit(w io.Writer, cfg Config, workDir string) error {
	if workDir != "" {
		if err := os.Chdir(workDir); err != nil {
			return fmt.Errorf("work dir: %w", err)
		}
	}

	eng, err := cfg.EngineFactory()
	if err != nil {
		return err
	}

	// Ingest current working directory into the engine before committing.
	// This populates v.cur so that Commit() has real blobs to persist into L2.
	if err := ingestCurrentDir(eng); err != nil {
		return err
	}

	id, _, err := eng.Commit("")
	if err != nil {
		return err
	}

	out := map[string]any{
		"snapshot_id": id,
	}
	return json.NewEncoder(w).Encode(out)
}

// ingestCurrentDir walks the current working dir and writes regular files
// into the engine using relative, slash-normalized paths.
// Skips internal folders like .git and .helios.
func ingestCurrentDir(eng interface{ WriteFile(string, []byte) error }) error {
    root, err := os.Getwd()
    if err != nil { return err }
    skip := map[string]struct{}{".git": {}, ".helios": {}}

    return filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
        if walkErr != nil { return walkErr }
        name := d.Name()
        if d.IsDir() {
            if _, found := skip[name]; found {
                return fs.SkipDir
            }
            return nil
        }
        // Only ingest regular files; skip symlinks, sockets, etc.
        if !d.Type().IsRegular() { return nil }

        rel, err := filepath.Rel(root, path)
        if err != nil { return err }
        rel = filepath.ToSlash(rel)
        // Double safety: skip anything under .git/ or .helios/
        if strings.HasPrefix(rel, ".git/") || strings.HasPrefix(rel, ".helios/") {
            return nil
        }

        b, err := os.ReadFile(path)
        if err != nil { return err }
        if err := eng.WriteFile(rel, b); err != nil { return err }
        if os.Getenv("HELIOS_DEBUG") == "1" {
            fmt.Fprintf(os.Stderr, "helios-debug: ingest %s (%d bytes)\n", rel, len(b))
        }
        return nil
    })
}

// HandleRestore processes restore command
func HandleRestore(w io.Writer, cfg Config, id string) error {
	if id == "" {
		return fmt.Errorf("--id is required")
	}

	eng, err := cfg.EngineFactory()
	if err != nil {
		return err
	}

	if err := eng.Restore(types.SnapshotID(id)); err != nil {
		return err
	}

	out := map[string]any{"restored": id}
	return json.NewEncoder(w).Encode(out)
}

// HandleDiff processes diff command
func HandleDiff(w io.Writer, cfg Config, from, to string) error {
	if from == "" || to == "" {
		return fmt.Errorf("--from and --to are required")
	}

	eng, err := cfg.EngineFactory()
	if err != nil {
		return err
	}

	dr, err := eng.Diff(types.SnapshotID(from), types.SnapshotID(to))
	if err != nil {
		return err
	}

	return json.NewEncoder(w).Encode(dr)
}

// HandleMaterialize processes materialize command
func HandleMaterialize(w io.Writer, cfg Config, id, outDir string, opts MatOpts) error {
	if id == "" || outDir == "" {
		return fmt.Errorf("--id and --out are required")
	}

	eng, err := cfg.EngineFactory()
	if err != nil {
		return err
	}

	matOpts := types.MatOpts{
		Include: opts.Include,
		Exclude: opts.Exclude,
	}

	_, err = eng.Materialize(types.SnapshotID(id), outDir, matOpts)
	if err != nil {
		return err
	}

	out := map[string]any{
		"materialized": id,
		"out":          outDir,
	}
	return json.NewEncoder(w).Encode(out)
}

// HandleStats processes stats command
func HandleStats(w io.Writer, cfg Config) error {
	eng, err := cfg.EngineFactory()
	if err != nil {
		return err
	}

	st := eng.L1Stats()
	em := eng.EngineMetricsSnapshot()

	out := map[string]any{
		"l1": map[string]any{
			"hits":      st.Hits,
			"misses":    st.Misses,
			"evictions": st.Evictions,
			"size":      st.SizeBytes,
			"items":     st.Items,
		},
		"engine": map[string]any{
			"commit_latency_us_p50": em.P50,
			"commit_latency_us_p95": em.P95,
			"commit_latency_us_p99": em.P99,
			"new_objects":           em.NewObjects,
			"new_bytes":             em.NewBytes,
		},
	}
	return json.NewEncoder(w).Encode(out)
}

// DefaultEngineFactory creates a real engine with L1/L2 stores
func DefaultEngineFactory() (Engine, error) {
	eng := vst.New()

	// Attach a small L1 cache for observable stats
	l1, _ := l1cache.New(l1cache.Config{CapacityBytes: 8 << 20, CompressionThreshold: 256})

	// Get store directory using the unified resolver
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}
	objDir, err := cli.ResolveStore(cwd)
	if err != nil {
		return nil, fmt.Errorf("resolve store directory: %w", err)
	}
	if os.Getenv("HELIOS_DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, "helios-debug: cwd=%s store=%s\n", cwd, objDir)
	}

	l2, err := objstore.Open(objDir, nil)
	if err != nil {
		return nil, err
	}
	eng.AttachStores(l1, l2)
	return eng, nil
}
