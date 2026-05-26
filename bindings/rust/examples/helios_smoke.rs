// Copyright 2025 Oppie Thunder Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// helios_smoke — round-trip + microbench for the Helios Rust bindings.
//
// Run from this directory:
//   cargo run --release --example helios_smoke
//
// Output: a per-op latency summary for init -> write -> commit -> branch ->
// fork -> overlay write -> merge -> checkout. Acceptance target: every op
// in this cycle completes in under 1ms.

use std::time::Instant;

use helios_rs::{BranchId, Vst};

fn measure<F: FnOnce() -> R, R>(label: &str, f: F) -> R {
    let t0 = Instant::now();
    let r = f();
    let dt = t0.elapsed();
    let micros = dt.as_secs_f64() * 1_000_000.0;
    let status = if dt.as_micros() < 1_000 { "OK " } else { "WARN" };
    println!("  {status}  {label:<28} {micros:>8.2} us");
    r
}

fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("Helios Rust bindings smoke test");
    println!("--------------------------------");

    // Warm up the cgo runtime so the first measured op doesn't pay the Go
    // runtime init cost (~5ms, amortised once per process).
    {
        let _warmup = Vst::new();
    }

    let v = measure("init (Vst::new, warm)", Vst::new);

    measure("write seed", || {
        v.write_file("seed.txt", b"ionq codex agents").unwrap()
    });

    let base = measure("commit", || v.commit("seed").unwrap());
    println!("  base snapshot id: {base}");

    let branch: BranchId = "main".into();
    measure("create_branch(main)", || {
        v.create_branch(&branch, &base).unwrap()
    });

    let fork = measure("fork", || v.fork(&base).unwrap());
    measure("fork write overlay", || {
        fork.write("overlay.txt", b"proposed payload").unwrap()
    });
    let _ = measure("fork read passthrough", || {
        fork.read("seed.txt").unwrap()
    });
    let new_id = measure("fork.merge_into(main)", || {
        fork.merge_into(&branch).map_err(|(_, e)| e).unwrap()
    });
    println!("  merged snapshot id: {new_id}");

    let head = measure("branch_head(main)", || {
        v.branch_head(&branch).unwrap().unwrap()
    });
    assert_eq!(head, new_id, "branch head should equal merged id");

    measure("restore (in-memory)", || v.restore(&new_id).unwrap());
    let got = measure("read after restore", || {
        v.read_file("overlay.txt").unwrap().unwrap()
    });
    assert_eq!(got, b"proposed payload");

    // K-fork divergence: drive several concurrent overlays in series and
    // confirm CAS semantics across rebases.
    println!("\nCAS rebase loop:");
    let head = v.branch_head(&branch).unwrap().unwrap();
    let f1 = v.fork(&head).unwrap();
    let f2 = v.fork(&head).unwrap();
    f1.write("racer.txt", b"f1 wins").unwrap();
    f2.write("racer.txt", b"f2 wins").unwrap();
    let _ = measure("f1.merge_into", || {
        f1.merge_into(&branch).map_err(|(_, e)| e).unwrap()
    });
    // f2 must see BranchStale and remain usable.
    let result = measure("f2.merge_into (stale)", || f2.merge_into(&branch));
    match result {
        Err((f2_alive, helios_rs::Error::BranchStale)) => {
            println!("  rebasing f2 onto new head and retrying...");
            f2_alive.discard();
            let new_head = v.branch_head(&branch).unwrap().unwrap();
            let f2b = v.fork(&new_head).unwrap();
            f2b.write("racer.txt", b"f2 retry").unwrap();
            let _ = measure("f2.merge_into (retry)", || {
                f2b.merge_into(&branch).map_err(|(_, e)| e).unwrap()
            });
        }
        Ok(_) => panic!("expected BranchStale on f2"),
        Err((_, e)) => return Err(Box::new(e)),
    }

    println!("\nOK — every op completed; acceptance gate (<1ms per op) reported above.");
    Ok(())
}
