// Copyright 2025 Oppie Thunder Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// build.rs — produce libhelios.a via `go build -buildmode=c-archive` if the
// artefact is not already present, then emit cargo link directives.
//
// The Go toolchain is invoked from the repo root (three levels up from this
// crate). The output path is bindings/c/libhelios.a relative to the repo root.
//
// Environment overrides:
//   HELIOS_FFI_LIB_DIR  — search directory containing libhelios.a (skips build)
//   HELIOS_GO_BIN       — path to `go` binary (defaults to PATH lookup)
//   HELIOS_SKIP_GO_BUILD=1 — assume libhelios.a already exists

use std::env;
use std::path::{Path, PathBuf};
use std::process::Command;

fn repo_root() -> PathBuf {
    // CARGO_MANIFEST_DIR = .../bindings/rust/helios-sys
    let manifest_dir = PathBuf::from(env::var("CARGO_MANIFEST_DIR").unwrap());
    manifest_dir
        .ancestors()
        .nth(3)
        .expect("expected repo root three dirs above helios-sys")
        .to_path_buf()
}

fn ensure_archive(lib_dir: &Path) {
    let archive = lib_dir.join("libhelios.a");
    if archive.exists() && env::var("HELIOS_SKIP_GO_BUILD").as_deref() == Ok("1") {
        return;
    }

    let root = repo_root();
    let go = env::var("HELIOS_GO_BIN").unwrap_or_else(|_| "go".to_string());

    // Tell cargo to rerun this build script when the Go source changes.
    println!("cargo:rerun-if-changed={}", root.join("cmd/heliosffi/main.go").display());
    println!("cargo:rerun-if-changed={}", root.join("pkg/helios/vst").display());
    println!("cargo:rerun-if-env-changed=HELIOS_FFI_LIB_DIR");
    println!("cargo:rerun-if-env-changed=HELIOS_GO_BIN");
    println!("cargo:rerun-if-env-changed=HELIOS_SKIP_GO_BUILD");

    let status = Command::new(&go)
        .args([
            "build",
            "-buildmode=c-archive",
            "-o",
        ])
        .arg(&archive)
        .arg("./cmd/heliosffi")
        .current_dir(&root)
        .env("CGO_ENABLED", "1")
        .status()
        .unwrap_or_else(|e| {
            panic!("failed to invoke `{} build -buildmode=c-archive`: {e}", go)
        });

    if !status.success() {
        panic!("`go build -buildmode=c-archive` failed (status={status})");
    }
}

fn main() {
    let lib_dir = match env::var("HELIOS_FFI_LIB_DIR") {
        Ok(dir) => PathBuf::from(dir),
        Err(_) => repo_root().join("bindings/c"),
    };

    ensure_archive(&lib_dir);

    // Cargo link directives.
    println!("cargo:rustc-link-search=native={}", lib_dir.display());
    println!("cargo:rustc-link-lib=static=helios");
    // c-archive needs pthread + dl on Linux; on macOS dl is implicit.
    if cfg!(target_os = "linux") {
        println!("cargo:rustc-link-lib=pthread");
        println!("cargo:rustc-link-lib=dl");
    } else if cfg!(target_os = "macos") {
        println!("cargo:rustc-link-lib=framework=CoreFoundation");
        println!("cargo:rustc-link-lib=framework=Security");
    }
}
