package cli

import (
  "fmt"
  "os"
  "path/filepath"
)

func ResolveStore(cwd string) (string, error) {
  if p := os.Getenv("HELIOS_STORE_DIR"); p != "" {
    if err := os.MkdirAll(p, 0o755); err != nil {
      return "", fmt.Errorf("create HELIOS_STORE_DIR: %w", err)
    }
    return p, nil
  }
  p := filepath.Join(cwd, ".helios", "objects")
  if err := os.MkdirAll(p, 0o755); err != nil {
    return "", fmt.Errorf("create default store: %w", err)
  }
  return p, nil
}