// Copyright 2025 Oppie Thunder Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// helios-sys — raw extern declarations for libhelios.a (cmd/heliosffi).
//
// Handwritten rather than bindgen-generated to keep the crate dependency
// footprint tiny (bindgen pulls libclang). The C ABI surface is small and
// stable; see bindings/c/helios.h for the curated contract.

#![allow(non_camel_case_types)]
#![allow(non_upper_case_globals)]

use core::ffi::{c_char, c_int, c_uchar};

pub type helios_vst_t = u64;
pub type helios_fork_t = u64;

/// Error codes mirrored from bindings/c/helios.h. Stable wire values.
pub const HELIOS_OK: c_int = 0;
pub const HELIOS_E_INVALID_ARG: c_int = -1;
pub const HELIOS_E_INVALID_HANDLE: c_int = -2;
pub const HELIOS_E_NOT_FOUND: c_int = -3;
pub const HELIOS_E_INTERNAL: c_int = -4;
pub const HELIOS_E_BRANCH_STALE: c_int = -5;
pub const HELIOS_E_FORK_MERGED: c_int = -6;
pub const HELIOS_E_FORK_DISCARDED: c_int = -7;
pub const HELIOS_E_BRANCH_EXISTS: c_int = -8;

extern "C" {
    // -- VST lifecycle ----------------------------------------------------
    pub fn helios_vst_new() -> helios_vst_t;
    pub fn helios_vst_free(h: helios_vst_t) -> c_int;

    // -- VST working set --------------------------------------------------
    pub fn helios_vst_write_file(
        h: helios_vst_t,
        path: *const c_char,
        data: *const c_uchar,
        length: usize,
    ) -> c_int;
    pub fn helios_vst_write_file_for_agent(
        h: helios_vst_t,
        agent_ptr: *const c_char,
        agent_len: usize,
        path: *const c_char,
        data: *const c_uchar,
        length: usize,
    ) -> c_int;
    pub fn helios_vst_read_file(
        h: helios_vst_t,
        path: *const c_char,
        out_buf: *mut *mut c_uchar,
        out_len: *mut usize,
    ) -> c_int;
    pub fn helios_vst_read_file_for_agent(
        h: helios_vst_t,
        agent_ptr: *const c_char,
        agent_len: usize,
        path: *const c_char,
        out_buf: *mut *mut c_uchar,
        out_len: *mut usize,
    ) -> c_int;
    pub fn helios_vst_delete_file(h: helios_vst_t, path: *const c_char) -> c_int;
    pub fn helios_vst_delete_file_for_agent(
        h: helios_vst_t,
        agent_ptr: *const c_char,
        agent_len: usize,
        path: *const c_char,
    ) -> c_int;

    // -- VST commit / restore --------------------------------------------
    pub fn helios_vst_commit(
        h: helios_vst_t,
        msg: *const c_char,
        out_id: *mut *mut c_char,
        out_len: *mut usize,
    ) -> c_int;
    pub fn helios_vst_commit_for_agent(
        h: helios_vst_t,
        agent_ptr: *const c_char,
        agent_len: usize,
        msg: *const c_char,
        out_id: *mut *mut c_char,
        out_len: *mut usize,
    ) -> c_int;
    pub fn helios_vst_restore_memory(
        h: helios_vst_t,
        id: *const c_char,
        id_len: usize,
    ) -> c_int;
    pub fn helios_vst_restore_memory_for_agent(
        h: helios_vst_t,
        agent_ptr: *const c_char,
        agent_len: usize,
        id: *const c_char,
        id_len: usize,
    ) -> c_int;

    // -- VST branches -----------------------------------------------------
    pub fn helios_vst_create_branch(
        h: helios_vst_t,
        name: *const c_char,
        head: *const c_char,
        head_len: usize,
    ) -> c_int;
    pub fn helios_vst_create_branch_for_agent(
        h: helios_vst_t,
        agent_ptr: *const c_char,
        agent_len: usize,
        name: *const c_char,
        head: *const c_char,
        head_len: usize,
    ) -> c_int;
    pub fn helios_vst_branch_head(
        h: helios_vst_t,
        name: *const c_char,
        out_id: *mut *mut c_char,
        out_len: *mut usize,
    ) -> c_int;
    pub fn helios_vst_branch_head_for_agent(
        h: helios_vst_t,
        agent_ptr: *const c_char,
        agent_len: usize,
        name: *const c_char,
        out_id: *mut *mut c_char,
        out_len: *mut usize,
    ) -> c_int;

    // -- Fork lifecycle ---------------------------------------------------
    pub fn helios_fork_new(
        h: helios_vst_t,
        base: *const c_char,
        base_len: usize,
        out_fork: *mut helios_fork_t,
    ) -> c_int;
    pub fn helios_fork_new_for_agent(
        h: helios_vst_t,
        agent_ptr: *const c_char,
        agent_len: usize,
        base: *const c_char,
        base_len: usize,
        out_fork: *mut helios_fork_t,
    ) -> c_int;
    pub fn helios_fork_discard(fh: helios_fork_t) -> c_int;
    pub fn helios_fork_write(
        fh: helios_fork_t,
        path: *const c_char,
        data: *const c_uchar,
        length: usize,
    ) -> c_int;
    pub fn helios_fork_read(
        fh: helios_fork_t,
        path: *const c_char,
        out_buf: *mut *mut c_uchar,
        out_len: *mut usize,
    ) -> c_int;
    pub fn helios_fork_delete(fh: helios_fork_t, path: *const c_char) -> c_int;
    pub fn helios_fork_merge_into(
        fh: helios_fork_t,
        branch: *const c_char,
        out_id: *mut *mut c_char,
        out_len: *mut usize,
    ) -> c_int;

    // -- Deallocators -----------------------------------------------------
    pub fn helios_buffer_free(buf: *mut c_uchar);
    pub fn helios_string_free(s: *mut c_char);
}
