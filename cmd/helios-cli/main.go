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

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/good-night-oppie/helios/cmd/helios-cli/internal/cli"
)

// Version metadata. Overridden at build time via -ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}
	switch os.Args[1] {
	case "commit":
		handleCommit()
	case "restore":
		handleRestore()
	case "diff":
		handleDiff()
	case "materialize":
		handleMaterialize()
	case "stats":
		handleStats()
	case "version", "--version", "-v":
		handleVersion()
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Println(`helios
Commands:
  commit       --work <path>
  restore      --id <snapshotID>
  diff         --from <id> --to <id>
  materialize  --id <snapshotID> --out <dir> [--include <glob>] [--exclude <glob>]
  stats
  version      [-v|--version]`)
}

// --- CLI configuration ---

func newConfig() cli.Config {
	return cli.Config{
		EngineFactory: cli.DefaultEngineFactory,
	}
}

// --- commands ---

func handleCommit() {
	fs := flag.NewFlagSet("commit", flag.ExitOnError)
	work := fs.String("work", ".", "working directory")
	_ = fs.Parse(os.Args[2:])

	cfg := newConfig()
	if err := cli.HandleCommit(os.Stdout, cfg, *work); err != nil {
		die(err)
	}
}

func handleRestore() {
	fs := flag.NewFlagSet("restore", flag.ExitOnError)
	id := fs.String("id", "", "snapshot id")
	_ = fs.Parse(os.Args[2:])

	cfg := newConfig()
	if err := cli.HandleRestore(os.Stdout, cfg, *id); err != nil {
		die(err)
	}
}

func handleDiff() {
	fs := flag.NewFlagSet("diff", flag.ExitOnError)
	from := fs.String("from", "", "from snapshot id")
	to := fs.String("to", "", "to snapshot id")
	_ = fs.Parse(os.Args[2:])

	cfg := newConfig()
	if err := cli.HandleDiff(os.Stdout, cfg, *from, *to); err != nil {
		die(err)
	}
}

func handleMaterialize() {
	fs := flag.NewFlagSet("materialize", flag.ExitOnError)
	id := fs.String("id", "", "snapshot id")
	outDir := fs.String("out", "", "output directory")
	include := fs.String("include", "", "include glob (optional)")
	exclude := fs.String("exclude", "", "exclude glob (optional)")
	_ = fs.Parse(os.Args[2:])

	opts := cli.MatOpts{}
	if *include != "" {
		opts.Include = []string{*include}
	}
	if *exclude != "" {
		opts.Exclude = []string{*exclude}
	}

	cfg := newConfig()
	if err := cli.HandleMaterialize(os.Stdout, cfg, *id, *outDir, opts); err != nil {
		die(err)
	}
}

func handleStats() {
	cfg := newConfig()
	if err := cli.HandleStats(os.Stdout, cfg); err != nil {
		die(err)
	}
}

// handleVersion prints CLI version information.
func handleVersion() {
	fmt.Printf("helios %s (commit %s, built %s)\n", version, commit, date)
}

func die(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(1)
}
