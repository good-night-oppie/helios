package vst

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/good-night-oppie/helios-engine/pkg/helios/types"
)

func FuzzPathRoundTrip(f *testing.F) {
	seed := []string{"a.txt", "dir/b.txt", "weird_字符/空 白.md", "./x", "../y"}
	for _, s := range seed {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, path string) {
		// Skip obviously insane inputs to keep fuzz time short
		if path == "" || !utf8.ValidString(path) || len(path) > 2048 {
			t.Skip()
		}
		// normalize your own way if needed
		eng := New()
		data := []byte("fuzz")
		_ = eng.WriteFile(path, data)
		id, _, err := eng.Commit("fuzz commit")
		if err != nil {
			t.Fatal(err)
		}

		// Use same engine instance to restore (snapshots are in memory)
		if err := eng.Restore(id); err != nil {
			t.Fatal(err)
		}

		got, err := eng.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "fuzz" {
			t.Fatalf("round-trip failed: %q", path)
		}
	})
}

func FuzzMaterializeSelectors(f *testing.F) {
	for _, s := range []string{"*.md", "**/*.go", "dir/**", "?.txt", "[ab]*"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, glob string) {
		if strings.Contains(glob, "\x00") || len(glob) > 256 {
			t.Skip()
		}
		eng := New()
		_ = eng.WriteFile("a/a.md", []byte("m"))
		_ = eng.WriteFile("a/b.go", []byte("g"))
		_ = eng.WriteFile("root.txt", []byte("t"))
		id, _, _ := eng.Commit("fuzz materialize")

		// Use same engine instance (snapshots are in memory)
		if err := eng.Restore(id); err != nil {
			t.Fatal(err)
		}
		tmp := t.TempDir()
		opts := types.MatOpts{Include: []string{glob}}
		// Materialize should never panic or corrupt output
		_, _ = eng.Materialize(id, tmp, opts)
	})
}
