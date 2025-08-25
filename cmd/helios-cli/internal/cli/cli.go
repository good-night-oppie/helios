package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/good-night-oppie/helios-engine/pkg/helios/l1cache"
	"github.com/good-night-oppie/helios-engine/pkg/helios/objstore"
	"github.com/good-night-oppie/helios-engine/pkg/helios/types"
	"github.com/good-night-oppie/helios-engine/pkg/helios/vst"
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

	id, _, err := eng.Commit("")
	if err != nil {
		return err
	}

	out := map[string]any{
		"snapshot_id": id,
	}
	return json.NewEncoder(w).Encode(out)
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
	out := map[string]any{
		"l1": map[string]any{
			"hits":      st.Hits,
			"misses":    st.Misses,
			"evictions": st.Evictions,
			"size":      st.SizeBytes,
			"items":     st.Items,
		},
	}
	return json.NewEncoder(w).Encode(out)
}

// DefaultEngineFactory creates a real engine with L1/L2 stores
func DefaultEngineFactory() (Engine, error) {
	eng := vst.New()

	// Attach a small L1 cache for observable stats
	l1, _ := l1cache.New(l1cache.Config{CapacityBytes: 8 << 20, CompressionThreshold: 256})

	// Persist objects in a hidden folder inside CWD (safe user-space path)
	cwd, _ := os.Getwd()
	objDir := filepath.Join(cwd, ".helios", "objects")
	if err := os.MkdirAll(objDir, 0o755); err != nil {
		return nil, err
	}
	l2, err := objstore.Open(objDir, nil)
	if err != nil {
		return nil, err
	}
	eng.AttachStores(l1, l2)
	return eng, nil
}
