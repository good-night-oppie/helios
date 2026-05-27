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

package vst

import (
	"errors"
	"unicode/utf8"

	"github.com/good-night-oppie/helios/pkg/helios/types"
)

// AgentId names a tenant in the VST namespace. Multi-tenant callers thread
// AgentId through *ForAgent methods; single-tenant callers use the existing
// methods which delegate to AgentDefault. AgentId values must be valid UTF-8
// strings; the empty string is normalised to AgentDefault.
type AgentId string

// AgentDefault is the reserved agent used by the legacy single-tenant API.
// All non-agent-qualified methods (WriteFile, CreateBranch, Fork, ...) act
// on this agent so existing callers keep working unchanged.
const AgentDefault AgentId = "default"

// ErrInvalidAgent is returned by error-returning *ForAgent methods when the
// supplied AgentId is not valid UTF-8. Void / boolean methods treat invalid
// IDs as a silent miss (no-op delete, not-found read).
var ErrInvalidAgent = errors.New("vst: AgentId must be valid UTF-8")

// validateAgent enforces the AgentId UTF-8 contract documented on AgentId.
// The empty string is allowed at this layer because normaliseAgent maps it
// to AgentDefault.
func validateAgent(a AgentId) error {
	if !utf8.ValidString(string(a)) {
		return ErrInvalidAgent
	}
	return nil
}

// normaliseAgent maps the empty string to AgentDefault. Callers should
// always pass the result of this through to per-agent state lookups.
func normaliseAgent(a AgentId) AgentId {
	if a == "" {
		return AgentDefault
	}
	return a
}

// agentState carries the per-tenant slice of VST state. Snapshots remain
// content-addressed and shared across agents in VST.snaps so that the same
// content from any agent dedups to a single entry.
type agentState struct {
	cur        map[string][]byte             // current working set
	pathToHash map[string]types.Hash         // working-set path -> blob hash (for L1/L2 fetch)
	branches   map[BranchID]types.SnapshotID // named branch heads
}

func newAgentState() *agentState {
	return &agentState{
		cur:        make(map[string][]byte),
		pathToHash: make(map[string]types.Hash),
		branches:   make(map[BranchID]types.SnapshotID),
	}
}

// agentRO returns the agent's state for read-only access. The caller must
// hold at least v.mu.RLock. Returns (nil, false) if the agent has never
// been written to (single-tenant readers should fall through to the
// L1/L2-via-snap path; multi-tenant callers should treat absence as
// NotFound).
func (v *VST) agentRO(a AgentId) (*agentState, bool) {
	s, ok := v.agents[normaliseAgent(a)]
	return s, ok
}

// agentRW returns the agent's state for read-write access, lazily creating
// it if absent. The caller MUST hold v.mu.Lock (write lock); calling under
// only RLock will race with concurrent agentRW from another goroutine.
func (v *VST) agentRW(a AgentId) *agentState {
	a = normaliseAgent(a)
	if s, ok := v.agents[a]; ok {
		return s
	}
	s := newAgentState()
	v.agents[a] = s
	return s
}
