/*
 * Copyright 2025 Oppie Thunder Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/*
 * helios.h — stable C ABI for in-process Helios consumers.
 *
 * This file is the human-curated face of the FFI surface. It pulls in the
 * cgo-generated libhelios.h (which declares the exported helios_* functions)
 * and adds the stable error-code constants + ownership/threading contract.
 *
 * Build artefacts:
 *   libhelios.a   — c-archive produced by `go build -buildmode=c-archive`
 *   libhelios.h   — cgo-generated function declarations
 *   helios.h      — this curated header (include this from C / bindgen)
 *
 * Ownership rules (see cmd/heliosffi/main.go docstring for details):
 *   - Handles (helios_vst_t, helios_fork_t) are opaque uint64 values, NOT
 *     pointers. Use 0 to mean "no handle" (helios_vst_new never returns 0
 *     on success; check by calling a follow-up op and checking error code).
 *   - All output byte buffers (uchar*) are allocated by Helios via malloc
 *     and MUST be released with helios_buffer_free.
 *   - All output strings (char*) are NUL-terminated and MUST be released
 *     with helios_string_free.
 *   - The library is safe for concurrent use across goroutines AND across
 *     OS threads (the underlying VST is goroutine-safe; the FFI registry
 *     uses sync.Map). Distinct fork handles may be driven in parallel.
 */

#ifndef HELIOS_H
#define HELIOS_H

#include <stdint.h>
#include <stddef.h>

/* Pull in the cgo-generated function declarations. */
#include "libhelios.h"

#ifdef __cplusplus
extern "C" {
#endif

/* Opaque handle types. Currently uint64 backed by a process-global registry. */
typedef uint64_t helios_vst_t;
typedef uint64_t helios_fork_t;

/* Error codes. Stable wire values; do not reorder. */
#define HELIOS_OK                  ( 0)
#define HELIOS_E_INVALID_ARG       (-1)
#define HELIOS_E_INVALID_HANDLE    (-2)
#define HELIOS_E_NOT_FOUND         (-3)
#define HELIOS_E_INTERNAL          (-4)
#define HELIOS_E_BRANCH_STALE      (-5)
#define HELIOS_E_FORK_MERGED       (-6)
#define HELIOS_E_FORK_DISCARDED    (-7)
#define HELIOS_E_BRANCH_EXISTS     (-8)

#ifdef __cplusplus
}
#endif

#endif /* HELIOS_H */
