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

// Package main produces a CGO c-archive (libhelios.a + libhelios.h) exporting
// a minimal, opaque-handle C ABI for the Helios VST + VFSFork primitives.
// Intended for in-process consumers (Rust via bindgen, Python via ctypes,
// any C-FFI capable runtime) that cannot afford a subprocess-per-op model.
//
// Build:
//   go build -buildmode=c-archive -o bindings/c/libhelios.a ./cmd/heliosffi
//
// All returned heap buffers (data, snapshot ids, error messages) are allocated
// with C.malloc and MUST be released by the caller with helios_buffer_free
// (for byte buffers) or helios_string_free (for NUL-terminated strings).
//
// Handles (helios_vst_t, helios_fork_t) are opaque uint64 ids backed by a
// process-global sync.Map. They are NOT pointers; passing a freed handle is
// safe and returns HELIOS_E_INVALID_HANDLE.
package main

/*
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
*/
import "C"

import (
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/good-night-oppie/helios/pkg/helios/types"
	"github.com/good-night-oppie/helios/pkg/helios/vst"
)

// Error codes mirrored on the C side via helios.h preamble.
// Stable wire values; do not reorder.
const (
	cOK              C.int = 0
	cErrInvalidArg   C.int = -1
	cErrInvalidHdl   C.int = -2
	cErrNotFound     C.int = -3
	cErrInternal     C.int = -4
	cErrBranchStale  C.int = -5
	cErrForkMerged   C.int = -6
	cErrForkDiscard  C.int = -7
	cErrBranchExists C.int = -8
)

// Handle registry: process-wide opaque uint64 → *VST or *Fork.
// Separate maps keep type-confusion impossible.
var (
	vstReg     sync.Map // uint64 -> *vst.VST
	forkReg    sync.Map // uint64 -> *vst.Fork
	vstNextID  uint64
	forkNextID uint64
)

func registerVST(v *vst.VST) uint64 {
	id := atomic.AddUint64(&vstNextID, 1)
	vstReg.Store(id, v)
	return id
}

func lookupVST(id uint64) *vst.VST {
	val, ok := vstReg.Load(id)
	if !ok {
		return nil
	}
	return val.(*vst.VST)
}

func registerFork(f *vst.Fork) uint64 {
	id := atomic.AddUint64(&forkNextID, 1)
	forkReg.Store(id, f)
	return id
}

func lookupFork(id uint64) *vst.Fork {
	val, ok := forkReg.Load(id)
	if !ok {
		return nil
	}
	return val.(*vst.Fork)
}

// cBufFromBytes copies a Go slice into a C.malloc'd buffer and writes its
// pointer + length through the supplied out params. Returns cOK on success.
// The caller must helios_buffer_free the returned pointer (safe to call on
// NULL — Go side helios_buffer_free no-ops on nil).
//
// Three-way encoding so callers can distinguish "missing" from "present
// but empty":
//
//	data == nil               → (*outBuf = NULL,        *outLen = 0)
//	                            interpreted as "path does not exist"
//	data != nil, len(data)==0 → (*outBuf = malloc(1),   *outLen = 0)
//	                            interpreted as "path exists, zero bytes"
//	len(data) > 0             → (*outBuf = malloc(len), *outLen = len)
//
// The empty-but-present case allocates a 1-byte placeholder so that a
// caller checking `out_buf != NULL` correctly sees an existing empty file
// rather than mistaking it for absence. The reported length is still 0 so
// no reader will dereference the placeholder; helios_buffer_free reclaims
// the byte. See PR #39 review for the round-trip failure this fixes.
func cBufFromBytes(data []byte, outBuf **C.uchar, outLen *C.size_t) C.int {
	if outBuf == nil || outLen == nil {
		return cErrInvalidArg
	}
	if data == nil {
		// Absent path: NULL + 0 = "not found".
		*outBuf = nil
		*outLen = 0
		return cOK
	}
	if len(data) == 0 {
		// Present empty path: distinguish from nil by allocating a single
		// byte placeholder. Length stays 0; consumers must not read it.
		p := C.malloc(1)
		if p == nil {
			return cErrInternal
		}
		*outBuf = (*C.uchar)(p)
		*outLen = 0
		return cOK
	}
	p := C.malloc(C.size_t(len(data)))
	if p == nil {
		return cErrInternal
	}
	C.memcpy(p, unsafe.Pointer(&data[0]), C.size_t(len(data)))
	*outBuf = (*C.uchar)(p)
	*outLen = C.size_t(len(data))
	return cOK
}

// cStrFromString copies a Go string into a NUL-terminated C.malloc'd buffer.
// On success writes pointer to *outStr and length (without NUL) to *outLen.
func cStrFromString(s string, outStr **C.char, outLen *C.size_t) C.int {
	if outStr == nil {
		return cErrInvalidArg
	}
	p := C.malloc(C.size_t(len(s) + 1))
	if p == nil {
		return cErrInternal
	}
	if len(s) > 0 {
		C.memcpy(p, unsafe.Pointer(unsafe.StringData(s)), C.size_t(len(s)))
	}
	// trailing NUL
	*(*byte)(unsafe.Pointer(uintptr(p) + uintptr(len(s)))) = 0
	*outStr = (*C.char)(p)
	if outLen != nil {
		*outLen = C.size_t(len(s))
	}
	return cOK
}

// goBytesFromC creates a Go slice view (no copy) over a C-owned buffer.
func goBytesFromC(buf *C.uchar, n C.size_t) []byte {
	if n == 0 || buf == nil {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(buf)), int(n))
}

// goStringFromC reads a NUL-terminated C string into a Go string.
func goStringFromC(s *C.char) string {
	if s == nil {
		return ""
	}
	return C.GoString(s)
}

// goStringFromCN reads a (ptr, len) C string into a Go string (no NUL needed).
func goStringFromCN(s *C.char, n C.size_t) string {
	if s == nil || n == 0 {
		return ""
	}
	return C.GoStringN(s, C.int(n))
}

// ---------------------------------------------------------------------------
// VST lifecycle
// ---------------------------------------------------------------------------

//export helios_vst_new
func helios_vst_new() C.uint64_t {
	v := vst.New()
	return C.uint64_t(registerVST(v))
}

//export helios_vst_free
func helios_vst_free(h C.uint64_t) C.int {
	id := uint64(h)
	val, ok := vstReg.LoadAndDelete(id)
	if !ok {
		return cErrInvalidHdl
	}
	v := val.(*vst.VST)
	_ = v.Close()
	return cOK
}

// ---------------------------------------------------------------------------
// VST: working-set IO
// ---------------------------------------------------------------------------

//export helios_vst_write_file
func helios_vst_write_file(h C.uint64_t, path *C.char, data *C.uchar, length C.size_t) C.int {
	return writeFileImpl(h, "", path, data, length)
}

//export helios_vst_write_file_for_agent
func helios_vst_write_file_for_agent(h C.uint64_t, agentPtr *C.char, agentLen C.size_t, path *C.char, data *C.uchar, length C.size_t) C.int {
	return writeFileImpl(h, goStringFromCN(agentPtr, agentLen), path, data, length)
}

func writeFileImpl(h C.uint64_t, agent string, path *C.char, data *C.uchar, length C.size_t) C.int {
	v := lookupVST(uint64(h))
	if v == nil {
		return cErrInvalidHdl
	}
	if path == nil {
		return cErrInvalidArg
	}
	// Copy the C bytes into a Go-owned slice (VST.WriteFileForAgent already
	// copies but this keeps the contract explicit and avoids retaining the
	// C pointer.)
	src := goBytesFromC(data, length)
	cp := make([]byte, len(src))
	copy(cp, src)
	if err := v.WriteFileForAgent(vst.AgentId(agent), goStringFromC(path), cp); err != nil {
		return cErrInternal
	}
	return cOK
}

//export helios_vst_read_file
func helios_vst_read_file(h C.uint64_t, path *C.char, outBuf **C.uchar, outLen *C.size_t) C.int {
	return readFileImpl(h, "", path, outBuf, outLen)
}

//export helios_vst_read_file_for_agent
func helios_vst_read_file_for_agent(h C.uint64_t, agentPtr *C.char, agentLen C.size_t, path *C.char, outBuf **C.uchar, outLen *C.size_t) C.int {
	return readFileImpl(h, goStringFromCN(agentPtr, agentLen), path, outBuf, outLen)
}

func readFileImpl(h C.uint64_t, agent string, path *C.char, outBuf **C.uchar, outLen *C.size_t) C.int {
	v := lookupVST(uint64(h))
	if v == nil {
		return cErrInvalidHdl
	}
	if path == nil {
		return cErrInvalidArg
	}
	b, err := v.ReadFileForAgent(vst.AgentId(agent), goStringFromC(path))
	if err != nil {
		return cErrInternal
	}
	return cBufFromBytes(b, outBuf, outLen)
}

//export helios_vst_delete_file
func helios_vst_delete_file(h C.uint64_t, path *C.char) C.int {
	return deleteFileImpl(h, "", path)
}

//export helios_vst_delete_file_for_agent
func helios_vst_delete_file_for_agent(h C.uint64_t, agentPtr *C.char, agentLen C.size_t, path *C.char) C.int {
	return deleteFileImpl(h, goStringFromCN(agentPtr, agentLen), path)
}

func deleteFileImpl(h C.uint64_t, agent string, path *C.char) C.int {
	v := lookupVST(uint64(h))
	if v == nil {
		return cErrInvalidHdl
	}
	if path == nil {
		return cErrInvalidArg
	}
	v.DeleteFileForAgent(vst.AgentId(agent), goStringFromC(path))
	return cOK
}

// ---------------------------------------------------------------------------
// VST: commit / restore
// ---------------------------------------------------------------------------

//export helios_vst_commit
func helios_vst_commit(h C.uint64_t, msg *C.char, outID **C.char, outLen *C.size_t) C.int {
	return commitImpl(h, "", msg, outID, outLen)
}

//export helios_vst_commit_for_agent
func helios_vst_commit_for_agent(h C.uint64_t, agentPtr *C.char, agentLen C.size_t, msg *C.char, outID **C.char, outLen *C.size_t) C.int {
	return commitImpl(h, goStringFromCN(agentPtr, agentLen), msg, outID, outLen)
}

func commitImpl(h C.uint64_t, agent string, msg *C.char, outID **C.char, outLen *C.size_t) C.int {
	v := lookupVST(uint64(h))
	if v == nil {
		return cErrInvalidHdl
	}
	id, _, err := v.CommitForAgent(vst.AgentId(agent), goStringFromC(msg))
	if err != nil {
		return cErrInternal
	}
	return cStrFromString(string(id), outID, outLen)
}

//export helios_vst_restore_memory
func helios_vst_restore_memory(h C.uint64_t, id *C.char, idLen C.size_t) C.int {
	return restoreMemoryImpl(h, "", id, idLen)
}

//export helios_vst_restore_memory_for_agent
func helios_vst_restore_memory_for_agent(h C.uint64_t, agentPtr *C.char, agentLen C.size_t, id *C.char, idLen C.size_t) C.int {
	return restoreMemoryImpl(h, goStringFromCN(agentPtr, agentLen), id, idLen)
}

func restoreMemoryImpl(h C.uint64_t, agent string, id *C.char, idLen C.size_t) C.int {
	v := lookupVST(uint64(h))
	if v == nil {
		return cErrInvalidHdl
	}
	sid := types.SnapshotID(goStringFromCN(id, idLen))
	// In-memory restore only: bypass filesystem materialisation so the FFI
	// path is suitable for hot agent loops (no PWD coupling, no atomic-rename).
	if err := v.RestoreForAgent(vst.AgentId(agent), sid, types.RestoreOpts{DryRun: false, WriteToFilesystem: false}); err != nil {
		return cErrNotFound
	}
	return cOK
}

// ---------------------------------------------------------------------------
// VST: branches
// ---------------------------------------------------------------------------

//export helios_vst_create_branch
func helios_vst_create_branch(h C.uint64_t, name *C.char, head *C.char, headLen C.size_t) C.int {
	return createBranchImpl(h, "", name, head, headLen)
}

//export helios_vst_create_branch_for_agent
func helios_vst_create_branch_for_agent(h C.uint64_t, agentPtr *C.char, agentLen C.size_t, name *C.char, head *C.char, headLen C.size_t) C.int {
	return createBranchImpl(h, goStringFromCN(agentPtr, agentLen), name, head, headLen)
}

func createBranchImpl(h C.uint64_t, agent string, name *C.char, head *C.char, headLen C.size_t) C.int {
	v := lookupVST(uint64(h))
	if v == nil {
		return cErrInvalidHdl
	}
	if name == nil {
		return cErrInvalidArg
	}
	err := v.CreateBranchForAgent(vst.AgentId(agent), vst.BranchID(goStringFromC(name)), types.SnapshotID(goStringFromCN(head, headLen)))
	switch err {
	case nil:
		return cOK
	case vst.ErrBranchExists:
		return cErrBranchExists
	default:
		return cErrInternal
	}
}

//export helios_vst_branch_head
func helios_vst_branch_head(h C.uint64_t, name *C.char, outID **C.char, outLen *C.size_t) C.int {
	return branchHeadImpl(h, "", name, outID, outLen)
}

//export helios_vst_branch_head_for_agent
func helios_vst_branch_head_for_agent(h C.uint64_t, agentPtr *C.char, agentLen C.size_t, name *C.char, outID **C.char, outLen *C.size_t) C.int {
	return branchHeadImpl(h, goStringFromCN(agentPtr, agentLen), name, outID, outLen)
}

func branchHeadImpl(h C.uint64_t, agent string, name *C.char, outID **C.char, outLen *C.size_t) C.int {
	v := lookupVST(uint64(h))
	if v == nil {
		return cErrInvalidHdl
	}
	id, ok := v.BranchHeadForAgent(vst.AgentId(agent), vst.BranchID(goStringFromC(name)))
	if !ok {
		return cErrNotFound
	}
	return cStrFromString(string(id), outID, outLen)
}

// ---------------------------------------------------------------------------
// Fork lifecycle
// ---------------------------------------------------------------------------

//export helios_fork_new
func helios_fork_new(h C.uint64_t, base *C.char, baseLen C.size_t, outForkHdl *C.uint64_t) C.int {
	return forkNewImpl(h, "", base, baseLen, outForkHdl)
}

//export helios_fork_new_for_agent
func helios_fork_new_for_agent(h C.uint64_t, agentPtr *C.char, agentLen C.size_t, base *C.char, baseLen C.size_t, outForkHdl *C.uint64_t) C.int {
	return forkNewImpl(h, goStringFromCN(agentPtr, agentLen), base, baseLen, outForkHdl)
}

func forkNewImpl(h C.uint64_t, agent string, base *C.char, baseLen C.size_t, outForkHdl *C.uint64_t) C.int {
	v := lookupVST(uint64(h))
	if v == nil {
		return cErrInvalidHdl
	}
	if outForkHdl == nil {
		return cErrInvalidArg
	}
	f, err := v.ForkForAgent(vst.AgentId(agent), types.SnapshotID(goStringFromCN(base, baseLen)))
	if err != nil {
		return cErrNotFound
	}
	*outForkHdl = C.uint64_t(registerFork(f))
	return cOK
}

//export helios_fork_discard
func helios_fork_discard(fh C.uint64_t) C.int {
	id := uint64(fh)
	val, ok := forkReg.LoadAndDelete(id)
	if !ok {
		return cErrInvalidHdl
	}
	f := val.(*vst.Fork)
	f.Discard()
	return cOK
}

//export helios_fork_write
func helios_fork_write(fh C.uint64_t, path *C.char, data *C.uchar, length C.size_t) C.int {
	f := lookupFork(uint64(fh))
	if f == nil {
		return cErrInvalidHdl
	}
	if path == nil {
		return cErrInvalidArg
	}
	src := goBytesFromC(data, length)
	if err := f.Write(goStringFromC(path), src); err != nil {
		return forkErrToCode(err)
	}
	return cOK
}

//export helios_fork_read
func helios_fork_read(fh C.uint64_t, path *C.char, outBuf **C.uchar, outLen *C.size_t) C.int {
	f := lookupFork(uint64(fh))
	if f == nil {
		return cErrInvalidHdl
	}
	if path == nil {
		return cErrInvalidArg
	}
	b, err := f.Read(goStringFromC(path))
	if err != nil {
		return forkErrToCode(err)
	}
	return cBufFromBytes(b, outBuf, outLen)
}

//export helios_fork_delete
func helios_fork_delete(fh C.uint64_t, path *C.char) C.int {
	f := lookupFork(uint64(fh))
	if f == nil {
		return cErrInvalidHdl
	}
	if path == nil {
		return cErrInvalidArg
	}
	if err := f.Delete(goStringFromC(path)); err != nil {
		return forkErrToCode(err)
	}
	return cOK
}

//export helios_fork_merge_into
func helios_fork_merge_into(fh C.uint64_t, branch *C.char, outID **C.char, outLen *C.size_t) C.int {
	f := lookupFork(uint64(fh))
	if f == nil {
		return cErrInvalidHdl
	}
	if branch == nil {
		return cErrInvalidArg
	}
	id, err := f.MergeInto(vst.BranchID(goStringFromC(branch)))
	if err != nil {
		return forkErrToCode(err)
	}
	// On successful merge the fork is consumed: drop its handle so callers can't
	// reuse it (Discard is a no-op on merged forks, but freeing the slot keeps
	// the registry from leaking).
	forkReg.Delete(uint64(fh))
	return cStrFromString(string(id), outID, outLen)
}

func forkErrToCode(err error) C.int {
	switch err {
	case nil:
		return cOK
	case vst.ErrBranchStale:
		return cErrBranchStale
	case vst.ErrForkMerged:
		return cErrForkMerged
	case vst.ErrForkDiscarded:
		return cErrForkDiscard
	case vst.ErrUnknownBranch, vst.ErrUnknownSnapshot:
		return cErrNotFound
	default:
		return cErrInternal
	}
}

// ---------------------------------------------------------------------------
// Buffer / string deallocators
// ---------------------------------------------------------------------------

//export helios_buffer_free
func helios_buffer_free(buf *C.uchar) {
	if buf != nil {
		C.free(unsafe.Pointer(buf))
	}
}

//export helios_string_free
func helios_string_free(s *C.char) {
	if s != nil {
		C.free(unsafe.Pointer(s))
	}
}

// main is required by Go but never executed for buildmode=c-archive.
func main() {}
