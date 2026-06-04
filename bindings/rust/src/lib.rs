// Copyright 2025 Oppie Thunder Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// helios-rs — safe Rust wrapper over helios-sys (libhelios.a c-archive).
//
// Provides:
//   Vst       — owning handle for a Helios VST instance.
//   Fork      — owning handle for a copy-on-write overlay.
//   SnapshotId, BranchId — newtype wrappers around String.
//
// Drop impls call the corresponding helios_*_free / helios_fork_discard,
// so leaking a handle requires explicit `mem::forget`. All allocations
// returned from Go (data buffers, snapshot ids) are copied into Rust-owned
// Vec/String and then released via helios_buffer_free / helios_string_free.

#![deny(unsafe_op_in_unsafe_fn)]

use std::ffi::{CString, NulError};
use std::fmt;
use std::ptr;
use std::slice;

use helios_sys as sys;
use thiserror::Error;

/// Errors returned by helios-rs.
#[derive(Debug, Error)]
pub enum Error {
    #[error("invalid argument")]
    InvalidArg,
    #[error("invalid handle")]
    InvalidHandle,
    #[error("not found")]
    NotFound,
    #[error("internal helios error")]
    Internal,
    #[error("branch head moved since fork (stale merge)")]
    BranchStale,
    #[error("fork already merged")]
    ForkMerged,
    #[error("fork already discarded")]
    ForkDiscarded,
    #[error("branch already exists")]
    BranchExists,
    #[error("path contained interior NUL byte: {0}")]
    Nul(#[from] NulError),
    #[error("unknown FFI error code: {0}")]
    Unknown(i32),
}

fn rc_to_result(rc: i32) -> Result<(), Error> {
    match rc {
        sys::HELIOS_OK => Ok(()),
        sys::HELIOS_E_INVALID_ARG => Err(Error::InvalidArg),
        sys::HELIOS_E_INVALID_HANDLE => Err(Error::InvalidHandle),
        sys::HELIOS_E_NOT_FOUND => Err(Error::NotFound),
        sys::HELIOS_E_INTERNAL => Err(Error::Internal),
        sys::HELIOS_E_BRANCH_STALE => Err(Error::BranchStale),
        sys::HELIOS_E_FORK_MERGED => Err(Error::ForkMerged),
        sys::HELIOS_E_FORK_DISCARDED => Err(Error::ForkDiscarded),
        sys::HELIOS_E_BRANCH_EXISTS => Err(Error::BranchExists),
        other => Err(Error::Unknown(other)),
    }
}

/// SnapshotId is the content-addressed handle to a committed VST state.
#[derive(Clone, PartialEq, Eq, Hash)]
pub struct SnapshotId(pub String);

impl fmt::Debug for SnapshotId {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "SnapshotId({})", self.0)
    }
}

impl fmt::Display for SnapshotId {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(&self.0)
    }
}

impl From<String> for SnapshotId {
    fn from(s: String) -> Self {
        SnapshotId(s)
    }
}

/// AgentId names a tenant in the multi-tenant VST namespace.
///
/// Identical to the Go-side `vst.AgentId` (string newtype). String form
/// is sent across FFI as a (ptr, len) pair so embedded NUL bytes are
/// not required to be encoded. The reserved value `"default"` (also
/// available via `AgentId::default()` or `AGENT_DEFAULT`) is the agent
/// the legacy single-tenant API resolves to, so legacy callers and
/// new multi-tenant callers see the same data when they share that ID.
#[derive(Clone, PartialEq, Eq, Hash, Debug)]
pub struct AgentId(pub String);

/// Reserved agent string used by the legacy single-tenant API.
pub const AGENT_DEFAULT: &str = "default";

impl From<&str> for AgentId {
    fn from(s: &str) -> Self {
        AgentId(s.to_owned())
    }
}

impl From<String> for AgentId {
    fn from(s: String) -> Self {
        AgentId(s)
    }
}

impl Default for AgentId {
    fn default() -> Self {
        AgentId(AGENT_DEFAULT.to_owned())
    }
}

impl fmt::Display for AgentId {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(&self.0)
    }
}

/// BranchId names a mutable head pointer.
#[derive(Clone, PartialEq, Eq, Hash, Debug)]
pub struct BranchId(pub String);

impl From<&str> for BranchId {
    fn from(s: &str) -> Self {
        BranchId(s.to_owned())
    }
}

/// Owned VST handle. Drop calls helios_vst_free.
pub struct Vst {
    handle: sys::helios_vst_t,
}

impl Vst {
    /// Create a new in-process VST instance.
    pub fn new() -> Self {
        // SAFETY: helios_vst_new has no preconditions; it always returns a
        // valid handle id (Go side panics on OOM rather than returning 0).
        let h = unsafe { sys::helios_vst_new() };
        Vst { handle: h }
    }

    /// Write `data` at `path` in the working set.
    pub fn write_file(&self, path: &str, data: &[u8]) -> Result<(), Error> {
        let c_path = CString::new(path)?;
        // SAFETY: handle is valid; path/data lifetimes outlive the call.
        let rc = unsafe {
            sys::helios_vst_write_file(
                self.handle,
                c_path.as_ptr(),
                if data.is_empty() { ptr::null() } else { data.as_ptr() },
                data.len(),
            )
        };
        rc_to_result(rc)
    }

    /// Read the working-set content at `path`. Returns Ok(None) if absent.
    pub fn read_file(&self, path: &str) -> Result<Option<Vec<u8>>, Error> {
        let c_path = CString::new(path)?;
        let mut buf: *mut u8 = ptr::null_mut();
        let mut len: usize = 0;
        // SAFETY: handle valid; out params are owned locals.
        let rc = unsafe {
            sys::helios_vst_read_file(self.handle, c_path.as_ptr(), &mut buf, &mut len)
        };
        rc_to_result(rc)?;
        Ok(copy_owned_buffer(buf, len))
    }

    /// Delete a path from the working set (no-op if absent).
    pub fn delete_file(&self, path: &str) -> Result<(), Error> {
        let c_path = CString::new(path)?;
        // SAFETY: handle valid; path lifetime outlives the call.
        let rc = unsafe { sys::helios_vst_delete_file(self.handle, c_path.as_ptr()) };
        rc_to_result(rc)
    }

    /// Commit the working set and return the new SnapshotId.
    pub fn commit(&self, msg: &str) -> Result<SnapshotId, Error> {
        let c_msg = CString::new(msg)?;
        let mut out: *mut i8 = ptr::null_mut();
        let mut len: usize = 0;
        // SAFETY: handle valid; out params owned locals.
        let rc = unsafe {
            sys::helios_vst_commit(self.handle, c_msg.as_ptr(), &mut out, &mut len)
        };
        rc_to_result(rc)?;
        Ok(SnapshotId(copy_owned_string(out, len)))
    }

    /// Restore a committed snapshot into the working set (memory only; no
    /// filesystem materialisation). Equivalent to `checkout` for the hot
    /// agent loop.
    pub fn restore(&self, id: &SnapshotId) -> Result<(), Error> {
        let bytes = id.0.as_bytes();
        // SAFETY: handle valid; bytes lifetime outlives the call.
        let rc = unsafe {
            sys::helios_vst_restore_memory(
                self.handle,
                bytes.as_ptr() as *const i8,
                bytes.len(),
            )
        };
        rc_to_result(rc)
    }

    /// Register a named branch pointing at `head`.
    pub fn create_branch(&self, name: &BranchId, head: &SnapshotId) -> Result<(), Error> {
        let c_name = CString::new(name.0.as_str())?;
        let head_bytes = head.0.as_bytes();
        // SAFETY: handle valid; name/head outlive the call.
        let rc = unsafe {
            sys::helios_vst_create_branch(
                self.handle,
                c_name.as_ptr(),
                head_bytes.as_ptr() as *const i8,
                head_bytes.len(),
            )
        };
        rc_to_result(rc)
    }

    /// Return the current head SnapshotId for a branch, or None if unknown.
    pub fn branch_head(&self, name: &BranchId) -> Result<Option<SnapshotId>, Error> {
        let c_name = CString::new(name.0.as_str())?;
        let mut out: *mut i8 = ptr::null_mut();
        let mut len: usize = 0;
        // SAFETY: handle valid; out params owned locals.
        let rc = unsafe {
            sys::helios_vst_branch_head(self.handle, c_name.as_ptr(), &mut out, &mut len)
        };
        match rc_to_result(rc) {
            Ok(()) => Ok(Some(SnapshotId(copy_owned_string(out, len)))),
            Err(Error::NotFound) => Ok(None),
            Err(e) => Err(e),
        }
    }

    /// Open a copy-on-write Fork rooted at `base`.
    pub fn fork(&self, base: &SnapshotId) -> Result<Fork<'_>, Error> {
        let bytes = base.0.as_bytes();
        let mut fh: sys::helios_fork_t = 0;
        // SAFETY: handle valid; out param owned local.
        let rc = unsafe {
            sys::helios_fork_new(
                self.handle,
                bytes.as_ptr() as *const i8,
                bytes.len(),
                &mut fh,
            )
        };
        rc_to_result(rc)?;
        Ok(Fork {
            handle: fh,
            _vst: std::marker::PhantomData,
        })
    }

    // -- Per-agent (multi-tenant) variants -------------------------------
    //
    // Each method targets a specific AgentId tenant in the VST namespace,
    // mirroring the Go-side `*ForAgent` API (PR-VA-1a, #46). The AgentId
    // string crosses FFI as a (ptr, len) pair — NOT a NUL-terminated CString
    // — so interior NUL bytes need no escaping and the Go side decodes it with
    // `goStringFromCN`. An empty AgentId is sent as (NULL, 0); the Go side
    // normalises both "" and "default" to AGENT_DEFAULT, so the legacy
    // non-agent methods above and `AgentId::default()` resolve to one tenant.

    /// Write `data` at `path` in `agent`'s working set.
    pub fn write_file_for_agent(
        &self,
        agent: &AgentId,
        path: &str,
        data: &[u8],
    ) -> Result<(), Error> {
        let c_path = CString::new(path)?;
        let agent_bytes = agent.0.as_bytes();
        // SAFETY: handle valid; agent/path/data lifetimes outlive the call.
        let rc = unsafe {
            sys::helios_vst_write_file_for_agent(
                self.handle,
                if agent_bytes.is_empty() { ptr::null() } else { agent_bytes.as_ptr() as *const i8 },
                agent_bytes.len(),
                c_path.as_ptr(),
                if data.is_empty() { ptr::null() } else { data.as_ptr() },
                data.len(),
            )
        };
        rc_to_result(rc)
    }

    /// Read `agent`'s working-set content at `path`. Returns Ok(None) if absent.
    pub fn read_file_for_agent(
        &self,
        agent: &AgentId,
        path: &str,
    ) -> Result<Option<Vec<u8>>, Error> {
        let c_path = CString::new(path)?;
        let agent_bytes = agent.0.as_bytes();
        let mut buf: *mut u8 = ptr::null_mut();
        let mut len: usize = 0;
        // SAFETY: handle valid; out params are owned locals.
        let rc = unsafe {
            sys::helios_vst_read_file_for_agent(
                self.handle,
                if agent_bytes.is_empty() { ptr::null() } else { agent_bytes.as_ptr() as *const i8 },
                agent_bytes.len(),
                c_path.as_ptr(),
                &mut buf,
                &mut len,
            )
        };
        rc_to_result(rc)?;
        Ok(copy_owned_buffer(buf, len))
    }

    /// Delete `path` from `agent`'s working set (no-op if absent).
    pub fn delete_file_for_agent(&self, agent: &AgentId, path: &str) -> Result<(), Error> {
        let c_path = CString::new(path)?;
        let agent_bytes = agent.0.as_bytes();
        // SAFETY: handle valid; agent/path lifetimes outlive the call.
        let rc = unsafe {
            sys::helios_vst_delete_file_for_agent(
                self.handle,
                if agent_bytes.is_empty() { ptr::null() } else { agent_bytes.as_ptr() as *const i8 },
                agent_bytes.len(),
                c_path.as_ptr(),
            )
        };
        rc_to_result(rc)
    }

    /// Commit `agent`'s working set and return the new SnapshotId.
    pub fn commit_for_agent(&self, agent: &AgentId, msg: &str) -> Result<SnapshotId, Error> {
        let c_msg = CString::new(msg)?;
        let agent_bytes = agent.0.as_bytes();
        let mut out: *mut i8 = ptr::null_mut();
        let mut len: usize = 0;
        // SAFETY: handle valid; out params are owned locals.
        let rc = unsafe {
            sys::helios_vst_commit_for_agent(
                self.handle,
                if agent_bytes.is_empty() { ptr::null() } else { agent_bytes.as_ptr() as *const i8 },
                agent_bytes.len(),
                c_msg.as_ptr(),
                &mut out,
                &mut len,
            )
        };
        rc_to_result(rc)?;
        Ok(SnapshotId(copy_owned_string(out, len)))
    }

    /// Restore a committed snapshot into `agent`'s working set (memory only;
    /// no filesystem materialisation).
    pub fn restore_for_agent(&self, agent: &AgentId, id: &SnapshotId) -> Result<(), Error> {
        let agent_bytes = agent.0.as_bytes();
        let id_bytes = id.0.as_bytes();
        // SAFETY: handle valid; agent/id lifetimes outlive the call.
        let rc = unsafe {
            sys::helios_vst_restore_memory_for_agent(
                self.handle,
                if agent_bytes.is_empty() { ptr::null() } else { agent_bytes.as_ptr() as *const i8 },
                agent_bytes.len(),
                id_bytes.as_ptr() as *const i8,
                id_bytes.len(),
            )
        };
        rc_to_result(rc)
    }

    /// Register a named branch in `agent`'s namespace pointing at `head`.
    pub fn create_branch_for_agent(
        &self,
        agent: &AgentId,
        name: &BranchId,
        head: &SnapshotId,
    ) -> Result<(), Error> {
        let c_name = CString::new(name.0.as_str())?;
        let agent_bytes = agent.0.as_bytes();
        let head_bytes = head.0.as_bytes();
        // SAFETY: handle valid; agent/name/head lifetimes outlive the call.
        let rc = unsafe {
            sys::helios_vst_create_branch_for_agent(
                self.handle,
                if agent_bytes.is_empty() { ptr::null() } else { agent_bytes.as_ptr() as *const i8 },
                agent_bytes.len(),
                c_name.as_ptr(),
                head_bytes.as_ptr() as *const i8,
                head_bytes.len(),
            )
        };
        rc_to_result(rc)
    }

    /// Return the current head SnapshotId for a branch in `agent`'s namespace,
    /// or None if unknown.
    pub fn branch_head_for_agent(
        &self,
        agent: &AgentId,
        name: &BranchId,
    ) -> Result<Option<SnapshotId>, Error> {
        let c_name = CString::new(name.0.as_str())?;
        let agent_bytes = agent.0.as_bytes();
        let mut out: *mut i8 = ptr::null_mut();
        let mut len: usize = 0;
        // SAFETY: handle valid; out params are owned locals.
        let rc = unsafe {
            sys::helios_vst_branch_head_for_agent(
                self.handle,
                if agent_bytes.is_empty() { ptr::null() } else { agent_bytes.as_ptr() as *const i8 },
                agent_bytes.len(),
                c_name.as_ptr(),
                &mut out,
                &mut len,
            )
        };
        match rc_to_result(rc) {
            Ok(()) => Ok(Some(SnapshotId(copy_owned_string(out, len)))),
            Err(Error::NotFound) => Ok(None),
            Err(e) => Err(e),
        }
    }

    /// Open a copy-on-write Fork in `agent`'s namespace rooted at `base`.
    pub fn fork_for_agent(&self, agent: &AgentId, base: &SnapshotId) -> Result<Fork<'_>, Error> {
        let agent_bytes = agent.0.as_bytes();
        let base_bytes = base.0.as_bytes();
        let mut fh: sys::helios_fork_t = 0;
        // SAFETY: handle valid; out param is an owned local.
        let rc = unsafe {
            sys::helios_fork_new_for_agent(
                self.handle,
                if agent_bytes.is_empty() { ptr::null() } else { agent_bytes.as_ptr() as *const i8 },
                agent_bytes.len(),
                base_bytes.as_ptr() as *const i8,
                base_bytes.len(),
                &mut fh,
            )
        };
        rc_to_result(rc)?;
        Ok(Fork {
            handle: fh,
            _vst: std::marker::PhantomData,
        })
    }

    /// Raw FFI handle. Exposed for FFI interop with other languages; do not
    /// call helios_vst_free on this from external code while the Vst exists.
    pub fn raw_handle(&self) -> sys::helios_vst_t {
        self.handle
    }
}

impl Default for Vst {
    fn default() -> Self {
        Vst::new()
    }
}

impl Drop for Vst {
    fn drop(&mut self) {
        // SAFETY: handle is owned by this struct; double-free guarded by Go side
        // (LoadAndDelete on the registry).
        unsafe {
            let _ = sys::helios_vst_free(self.handle);
        }
    }
}

// SAFETY: All Vst ops go through the goroutine-safe Go runtime via uint64
// handle dispatch. There is no thread-local state on the Rust side.
unsafe impl Send for Vst {}
unsafe impl Sync for Vst {}

/// Owned Fork handle. Lifetime-bound to the parent Vst so the registry cannot
/// be torn down underneath an active fork. Drop calls helios_fork_discard.
pub struct Fork<'v> {
    handle: sys::helios_fork_t,
    _vst: std::marker::PhantomData<&'v Vst>,
}

impl<'v> Fork<'v> {
    /// Write `data` at `path` in the fork overlay (RAM only).
    pub fn write(&self, path: &str, data: &[u8]) -> Result<(), Error> {
        let c_path = CString::new(path)?;
        // SAFETY: handle valid; path/data outlive the call.
        let rc = unsafe {
            sys::helios_fork_write(
                self.handle,
                c_path.as_ptr(),
                if data.is_empty() { ptr::null() } else { data.as_ptr() },
                data.len(),
            )
        };
        rc_to_result(rc)
    }

    /// Read `path` through the overlay (overlay → tombstone → base).
    pub fn read(&self, path: &str) -> Result<Option<Vec<u8>>, Error> {
        let c_path = CString::new(path)?;
        let mut buf: *mut u8 = ptr::null_mut();
        let mut len: usize = 0;
        // SAFETY: handle valid; out params owned locals.
        let rc = unsafe {
            sys::helios_fork_read(self.handle, c_path.as_ptr(), &mut buf, &mut len)
        };
        rc_to_result(rc)?;
        Ok(copy_owned_buffer(buf, len))
    }

    /// Tombstone a path in the fork (no-op if absent from base).
    pub fn delete(&self, path: &str) -> Result<(), Error> {
        let c_path = CString::new(path)?;
        // SAFETY: handle valid; path outlives the call.
        let rc = unsafe { sys::helios_fork_delete(self.handle, c_path.as_ptr()) };
        rc_to_result(rc)
    }

    /// Atomically merge this fork into `branch`. Consumes the fork on success
    /// (the handle is invalidated). On `Error::BranchStale` the fork remains
    /// alive and the caller may rebase and retry.
    pub fn merge_into(mut self, branch: &BranchId) -> Result<SnapshotId, (Self, Error)> {
        let c_branch = match CString::new(branch.0.as_str()) {
            Ok(s) => s,
            Err(e) => return Err((self, Error::Nul(e))),
        };
        let mut out: *mut i8 = ptr::null_mut();
        let mut len: usize = 0;
        // SAFETY: handle valid; out params owned locals.
        let rc = unsafe {
            sys::helios_fork_merge_into(self.handle, c_branch.as_ptr(), &mut out, &mut len)
        };
        match rc_to_result(rc) {
            Ok(()) => {
                let id = SnapshotId(copy_owned_string(out, len));
                // Go side already invalidated the handle on success; suppress
                // Drop's discard to avoid the redundant InvalidHandle return.
                self.handle = 0;
                Ok(id)
            }
            Err(e) => Err((self, e)),
        }
    }

    /// Explicit discard. Drop calls this automatically; prefer Drop unless
    /// you want to release fork RAM before scope end.
    pub fn discard(mut self) {
        if self.handle != 0 {
            // SAFETY: handle valid; idempotent on Go side.
            unsafe { let _ = sys::helios_fork_discard(self.handle); }
            self.handle = 0;
        }
    }

    /// Raw FFI handle (do not pass to helios_fork_discard externally).
    pub fn raw_handle(&self) -> sys::helios_fork_t {
        self.handle
    }
}

impl<'v> Drop for Fork<'v> {
    fn drop(&mut self) {
        if self.handle != 0 {
            // SAFETY: handle valid; Go side ignores already-discarded forks.
            unsafe { let _ = sys::helios_fork_discard(self.handle); }
        }
    }
}

// SAFETY: same goroutine-safe rationale as Vst; the handle is uint64.
unsafe impl<'v> Send for Fork<'v> {}
unsafe impl<'v> Sync for Fork<'v> {}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

fn copy_owned_buffer(buf: *mut u8, len: usize) -> Option<Vec<u8>> {
    // Three-way encoding from cBufFromBytes in cmd/heliosffi/main.go:
    //   NULL                  -> None       (path absent)
    //   non-NULL,  len == 0   -> Some(vec![]) (path present, zero bytes)
    //   non-NULL,  len > 0    -> Some(bytes)
    if buf.is_null() {
        return None;
    }
    if len == 0 {
        // Empty-but-present sentinel: Go allocated a 1-byte placeholder so
        // we can distinguish absence from emptiness. We must NOT read it
        // (its contents are uninitialised) but we still own it and must
        // hand it back to helios_buffer_free.
        // SAFETY: buf was allocated by C.malloc on the Go side.
        unsafe { sys::helios_buffer_free(buf); }
        return Some(Vec::new());
    }
    // SAFETY: Go-side guarantees buf was malloc'd with `len` valid bytes.
    let bytes = unsafe { slice::from_raw_parts(buf, len) };
    let owned = bytes.to_vec();
    // SAFETY: buf was allocated by C.malloc on the Go side.
    unsafe { sys::helios_buffer_free(buf); }
    Some(owned)
}

fn copy_owned_string(ptr: *mut i8, len: usize) -> String {
    if ptr.is_null() {
        return String::new();
    }
    // SAFETY: Go-side guarantees NUL terminator at offset `len`.
    let bytes = unsafe { slice::from_raw_parts(ptr as *const u8, len) };
    let owned = String::from_utf8_lossy(bytes).into_owned();
    // SAFETY: ptr was allocated by C.malloc on the Go side.
    unsafe { sys::helios_string_free(ptr); }
    owned
}

#[cfg(test)]
mod tests {
    use super::*;

    // PR #39 review follow-up: write b"" then read back must yield
    // Some(vec![]) — NOT None. None means "absent"; an existing empty
    // file is a distinct state. Before the fix, cBufFromBytes returned
    // NULL for both nil and []byte{}, collapsing the two states on the
    // Rust side.
    #[test]
    fn test_empty_file_roundtrip_vst() {
        let v = Vst::new();
        v.write_file("empty.txt", b"").unwrap();
        let got = v.read_file("empty.txt").unwrap();
        assert_eq!(got, Some(Vec::new()), "VST: empty file must be Some(vec![]) not None");

        // Absent file must still read back as None.
        let missing = v.read_file("does_not_exist.txt").unwrap();
        assert_eq!(missing, None, "VST: absent file must read back as None");

        // Round-trip survives commit → fork → read.
        let base = v.commit("seed").unwrap();
        let branch: BranchId = "main".into();
        v.create_branch(&branch, &base).unwrap();

        let f = v.fork(&base).unwrap();
        let from_fork_base = f.read("empty.txt").unwrap();
        assert_eq!(
            from_fork_base,
            Some(Vec::new()),
            "Fork base passthrough: empty file must be Some(vec![]) not None"
        );

        // Fork-overlay empty write also round-trips.
        f.write("overlay_empty.txt", b"").unwrap();
        let from_overlay = f.read("overlay_empty.txt").unwrap();
        assert_eq!(
            from_overlay,
            Some(Vec::new()),
            "Fork overlay: empty file must be Some(vec![]) not None"
        );
        f.discard();
    }

    // PR-VA-1a: multi-tenant isolation across the FFI. A write under one
    // AgentId must not be visible to a different AgentId, and the legacy
    // non-agent API must alias the same tenant as AgentId::default()
    // ("default"), because the Go side normalises "" -> AGENT_DEFAULT.
    #[test]
    fn test_agent_tenant_roundtrip_isolation() {
        let v = Vst::new();
        let alice: AgentId = "alice".into();
        let bob: AgentId = "bob".into();

        // Round-trip: alice writes, alice reads her own bytes back.
        v.write_file_for_agent(&alice, "secret.txt", b"alice-data").unwrap();
        assert_eq!(
            v.read_file_for_agent(&alice, "secret.txt").unwrap(),
            Some(b"alice-data".to_vec()),
            "alice must read her own write"
        );

        // Isolation: bob cannot see alice's path (absent -> None, not bytes).
        assert_eq!(
            v.read_file_for_agent(&bob, "secret.txt").unwrap(),
            None,
            "bob must NOT see alice's file (tenant isolation)"
        );

        // Bob's independent write to the same path does not leak into alice.
        v.write_file_for_agent(&bob, "secret.txt", b"bob-data").unwrap();
        assert_eq!(
            v.read_file_for_agent(&bob, "secret.txt").unwrap(),
            Some(b"bob-data".to_vec()),
            "bob reads his own write"
        );
        assert_eq!(
            v.read_file_for_agent(&alice, "secret.txt").unwrap(),
            Some(b"alice-data".to_vec()),
            "alice's data is unchanged by bob's write to the same path"
        );

        // Legacy non-agent API aliases the "default" tenant, both directions.
        v.write_file("legacy.txt", b"legacy").unwrap();
        assert_eq!(
            v.read_file_for_agent(&AgentId::default(), "legacy.txt").unwrap(),
            Some(b"legacy".to_vec()),
            "legacy WriteFile must be visible under AgentId::default()"
        );
        v.write_file_for_agent(&AgentId::default(), "default_write.txt", b"d").unwrap();
        assert_eq!(
            v.read_file("default_write.txt").unwrap(),
            Some(b"d".to_vec()),
            "default-tenant write must be visible via the legacy ReadFile"
        );

        // The default tenant is itself isolated from the named tenants.
        assert_eq!(
            v.read_file_for_agent(&alice, "legacy.txt").unwrap(),
            None,
            "alice must be isolated from the default tenant"
        );

        // Delete under one agent does not affect another agent's same-named path.
        v.delete_file_for_agent(&bob, "secret.txt").unwrap();
        assert_eq!(
            v.read_file_for_agent(&bob, "secret.txt").unwrap(),
            None,
            "bob's delete removes bob's file"
        );
        assert_eq!(
            v.read_file_for_agent(&alice, "secret.txt").unwrap(),
            Some(b"alice-data".to_vec()),
            "bob's delete must NOT touch alice's same-named path"
        );
    }

    #[test]
    fn end_to_end_fork_merge() {
        let v = Vst::new();
        v.write_file("a.txt", b"alpha").unwrap();
        let base = v.commit("seed").unwrap();
        let branch: BranchId = "main".into();
        v.create_branch(&branch, &base).unwrap();

        let f = v.fork(&base).unwrap();
        f.write("a.txt", b"alpha-2").unwrap();
        let got = f.read("a.txt").unwrap().unwrap();
        assert_eq!(got, b"alpha-2");
        let new_id = f.merge_into(&branch).map_err(|(_, e)| e).unwrap();

        let head = v.branch_head(&branch).unwrap().unwrap();
        assert_eq!(head, new_id);
        assert_ne!(head, base);
    }

    #[test]
    fn stale_cas_returns_branch_stale() {
        let v = Vst::new();
        v.write_file("a.txt", b"alpha").unwrap();
        let base = v.commit("seed").unwrap();
        let branch: BranchId = "main".into();
        v.create_branch(&branch, &base).unwrap();

        let fa = v.fork(&base).unwrap();
        let fb = v.fork(&base).unwrap();

        fa.write("a.txt", b"A").unwrap();
        fb.write("a.txt", b"B").unwrap();

        let _ = fa.merge_into(&branch).map_err(|(_, e)| e).unwrap();

        // Project the error code only; drop the Fork before v goes out of scope.
        let stale_code = match fb.merge_into(&branch) {
            Ok(_) => None,
            Err((fb_alive, e)) => {
                drop(fb_alive);
                Some(e)
            }
        };
        match stale_code {
            Some(Error::BranchStale) => (),
            other => panic!("expected BranchStale, got {other:?}"),
        }
    }
}
