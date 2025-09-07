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