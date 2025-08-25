package policy

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var bannedFns = map[string]struct{}{
	"Mount":        {},
	"Unmount":      {},
	"PivotRoot":    {},
	"Unshare":      {},
	"Setns":        {},
	"MountSetattr": {},
}

var bannedConsts = map[string]struct{}{
	"CLONE_NEWNS":     {},
	"CLONE_NEWUSER":   {},
	"CLONE_NEWNET":    {},
	"CLONE_NEWUTS":    {},
	"CLONE_NEWCGROUP": {},
	"CLONE_NEWPID":    {},
	"CLONE_NEWIPC":    {},
	"CAP_SYS_ADMIN":   {},
}

func Test_NoPrivileged_KernelOps_InUserSpace(t *testing.T) {
	root := locateRepoRoot(t)
	targets := []string{"pkg", "cmd", "internal"}
	for _, dir := range targets {
		path := filepath.Join(root, dir)
		_ = filepath.WalkDir(path, func(p string, d os.DirEntry, _ error) error {
			if d.IsDir() || !strings.HasSuffix(p, ".go") || strings.Contains(p, "internal/policy/policy_guard_test.go") {
				return nil
			}
			checkFile(t, p)
			return nil
		})
	}
}

func checkFile(t *testing.T, filePath string) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", filePath, err)
	}
	ast.Inspect(f, func(n ast.Node) bool {
		if x, ok := n.(*ast.SelectorExpr); ok {
			if pkg, _ := x.X.(*ast.Ident); pkg != nil {
				if (pkg.Name == "syscall" || pkg.Name == "unix") && (has(bannedFns, x.Sel.Name) || has(bannedConsts, x.Sel.Name)) {
					t.Fatalf("forbidden: %s.%s in %s", pkg.Name, x.Sel.Name, filePath)
				}
			}
		}
		return true
	})
}

func has(m map[string]struct{}, k string) bool {
	_, ok := m[k]
	return ok
}

func locateRepoRoot(t *testing.T) string {
	wd, _ := os.Getwd()
	cur := wd
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(cur, "go.mod")); err == nil {
			return cur
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return wd
}
