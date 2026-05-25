# Helios ↔ ionq integration (Vectors A/B)

This document is the reference for embedding Helios in multi-agent runtimes
that cannot afford a subprocess per checkpoint/checkout cycle. It covers the
two P0 primitives ionq consumes:

- **Vector A — checkpoint backend.** In-process VST `Commit`/`Restore`
  exposed via CGO `c-archive` (`cmd/heliosffi`) and a safe Rust wrapper
  (`bindings/rust/helios-rs`).
- **Vector B — in-memory validation harness.** `VFSFork` overlay primitive
  (`pkg/helios/vst/fork.go`) — fork-without-commit, validate-against-fork,
  atomic compare-and-swap merge or discard. Memory cost is O(changed chunks),
  not O(working set).

The K-way speculative scheduler (Vector C) is deferred to P1.

---

## Architecture at a glance

```
+----------------------------------------------------------------+
| ionq Rust runtime  (or any C-FFI capable language)             |
+----------------------------------------------------------------+
              | helios_rs (safe wrapper, lifetimes + thiserror)
              v
+----------------------------------------------------------------+
| helios-sys: extern "C" decls (handwritten, bindgen-free)       |
+----------------------------------------------------------------+
              | C ABI (uint64 opaque handles)
              v
+----------------------------------------------------------------+
| libhelios.a  ←  go build -buildmode=c-archive ./cmd/heliosffi  |
|   cmd/heliosffi   →  pkg/helios/vst   (VST + Fork + branches)  |
|                       BLAKE3 CAS + Merkle tree                 |
+----------------------------------------------------------------+
```

The FFI surface is intentionally narrow. All cross-language allocations are
owned by the side that allocated them: Go-allocated buffers ship back via
`helios_buffer_free`/`helios_string_free`; Rust-allocated slices stay on
the Rust stack.

---

## Build & smoke test

```bash
# 1. Build libhelios.a + libhelios.h via cgo c-archive
make ffi-archive

# 2. Compile and run the C smoke test
make ffi-smoke

# 3. Build the Rust crate workspace (helios-sys + helios-rs)
make rust-build

# 4. Run the round-trip example (init→commit→fork→merge→restore)
make rust-smoke
```

Reference timings (Intel Xeon 8259CL, release build, single-threaded, warm):

| Op                              | Wall  | Notes                          |
|---------------------------------|-------|--------------------------------|
| `Vst::new` (warm)               |  ~2 µs | cold first call ~6 ms (Go runtime init, amortised) |
| `write_file`                    | ~15 µs |                                |
| `commit` (1 file)               | ~80 µs |                                |
| `create_branch`                 |  ~2 µs |                                |
| `fork`                          |  ~3 µs |                                |
| `fork.write` (overlay)          |  ~2 µs |                                |
| `fork.read` (base passthrough)  |  ~2 µs |                                |
| `fork.merge_into` (success CAS) | ~17 µs | excludes RocksDB flush         |
| `fork.merge_into` (stale CAS)   |  ~7 µs | branch head moved; fork alive  |
| `branch_head`                   |  ~2 µs |                                |
| `restore` (in-memory)           |  ~3 µs |                                |

All values comfortably under the 1 ms / 50 µs / 100 µs gates from the brief.

---

## C ABI: error codes

All entry points return `int`; `0` = success, negative = error. Defined in
`bindings/c/helios.h` and `bindings/rust/helios-sys/src/lib.rs`:

| Constant                       | Value | Meaning                                     |
|--------------------------------|------:|---------------------------------------------|
| `HELIOS_OK`                    |   0   | success                                     |
| `HELIOS_E_INVALID_ARG`         |  -1   | NULL pointer or empty required arg          |
| `HELIOS_E_INVALID_HANDLE`      |  -2   | handle is unknown or already freed          |
| `HELIOS_E_NOT_FOUND`           |  -3   | snapshot / branch / path not found          |
| `HELIOS_E_INTERNAL`            |  -4   | unexpected Go-side error (check stderr)     |
| `HELIOS_E_BRANCH_STALE`        |  -5   | merge CAS failed; branch head moved         |
| `HELIOS_E_FORK_MERGED`         |  -6   | op attempted on already-merged fork         |
| `HELIOS_E_FORK_DISCARDED`      |  -7   | op attempted on already-discarded fork      |
| `HELIOS_E_BRANCH_EXISTS`       |  -8   | `CreateBranch` collision                    |

Wire values are **stable**; do not reorder.

---

## Ownership & threading contract

- **Handles** (`helios_vst_t`, `helios_fork_t`) are opaque `uint64` values
  backed by a process-global `sync.Map`. They are NOT pointers. `0` is
  reserved as "no handle" only in `helios_fork_new`'s out-param.
- **Output byte buffers** (`unsigned char **out_buf`) are allocated via
  `C.malloc` on the Go side and MUST be released with `helios_buffer_free`.
- **Output strings** (`char **out_id`) are NUL-terminated and released
  with `helios_string_free`. The length out-param does NOT include the NUL.
- **Concurrency:** the library is goroutine-safe AND thread-safe. Distinct
  forks may be driven from distinct OS threads in parallel. The Rust wrapper
  marks `Vst` / `Fork` as `Send + Sync` accordingly.

---

## VFSFork semantics (Vector B core)

The Fork primitive sits between "ephemeral working set" and "committed
snapshot": it is a copy-on-write overlay over an immutable base snapshot.

```
                              fork.MergeInto(branch)
   base SnapshotID --+--> Fork --+--> new SnapshotID  (branch head advances)
                     |           |
                     |           +--> ErrBranchStale  (branch head moved)
                     |                (fork stays alive; caller may rebase)
                     |
                     +--> Fork --+--> Discard          (RAM released, no I/O)
```

Constraints honoured by the implementation:

1. **No RocksDB writes until MergeInto succeeds.** Fork overlays live in L0
   (per-fork RAM map). Blobs and snapshot metadata are only persisted to L2
   inside the successful `MergeInto` phase 4.
2. **Atomic compare-and-swap on the branch head.** If `f.base != branches[branch]`
   at the moment the VST write lock is acquired, `ErrBranchStale` is returned
   and no state is mutated.
3. **`Discard` is mandatory.** A `runtime.SetFinalizer` is wired defensively
   so dropped Forks do not pin memory forever, but the explicit `Discard`
   path is the supported lifecycle. The Rust `Fork::drop` calls
   `helios_fork_discard` automatically.
4. **Parallel fork safety.** K forks from the same base share an immutable
   reference to the base snapshot map. Tests cover both disjoint-branch
   parallelism (32 forks × 100 ops) and racing-CAS parallelism (16 forks
   competing for the same branch with rebase-retry; exactly K winners).

---

## Where things live

| Path                                     | Purpose                                   |
|------------------------------------------|-------------------------------------------|
| `pkg/helios/vst/fork.go`                 | Fork type + MergeInto/Discard/Diff/Read/Write |
| `pkg/helios/vst/branch.go`               | Branch registry (CreateBranch/BranchHead/...) |
| `pkg/helios/vst/fork_test.go`            | Functional + concurrent + finalizer tests |
| `pkg/helios/vst/fork_bench_test.go`      | Microbenchmarks for the acceptance gates  |
| `cmd/heliosffi/main.go`                  | CGO `//export` shim — opaque uint64 ABI   |
| `bindings/c/helios.h`                    | Curated C header (error codes + contract) |
| `bindings/c/smoke.c`                     | C-level round-trip smoke test             |
| `bindings/rust/helios-sys/`              | Raw `extern "C"` declarations + build.rs  |
| `bindings/rust/src/lib.rs`               | Safe wrapper: `Vst`, `Fork`, `SnapshotId` |
| `bindings/rust/examples/helios_smoke.rs` | `cargo run --example helios_smoke`        |

---

## Out of scope for P0 (deferred)

- **FastCDC chunking** (Vector A enhancement; file-level CAS is fine for now).
- **`Validator` trait + `propose_and_select`** (Vector C, P1+).
- **Two-tier hash** (BLAKE3-only stays).
- **WAL fsync chaos-test** (P2 reliability).
- **Replay-deterministic event journal link** (P2).
- **Fork TTL via generation marker** (P2 GC).
