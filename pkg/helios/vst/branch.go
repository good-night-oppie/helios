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

// Branch registry — minimal named-head primitive required by Fork.MergeInto's
// compare-and-swap. Intentionally narrow: this is not a full reference-tracking
// store, only the mutable pointer that VFSFork advances atomically.
//
// Branches are per-agent: each AgentId owns an independent (name -> SnapshotID)
// map. The legacy single-tenant API (CreateBranch / BranchHead / Branches /
// DeleteBranch) delegates to the AgentDefault agent. Snapshots themselves are
// content-addressed and shared across agents via VST.snaps.

package vst

import (
	"fmt"

	"github.com/good-night-oppie/helios/pkg/helios/types"
)

// CreateBranch registers a new named branch under the default agent.
// Equivalent to CreateBranchForAgent(AgentDefault, name, head).
func (v *VST) CreateBranch(name BranchID, head types.SnapshotID) error {
	return v.CreateBranchForAgent(AgentDefault, name, head)
}

// CreateBranchForAgent registers a new named branch under the given agent.
// Use the empty SnapshotID to create an orphan branch (Fork from "" allowed
// only if an empty-tree snapshot has been committed; otherwise callers must
// commit first and then CreateBranch with the resulting id).
func (v *VST) CreateBranchForAgent(agent AgentId, name BranchID, head types.SnapshotID) error {
	if err := validateAgent(agent); err != nil {
		return err
	}
	if name == "" {
		return fmt.Errorf("vst.CreateBranch: empty branch name")
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	s := v.agentRW(agent)
	if _, ok := s.branches[name]; ok {
		return ErrBranchExists
	}
	if head != "" {
		if _, ok := v.snaps[head]; !ok {
			return fmt.Errorf("vst.CreateBranch: %w (head=%s)", ErrUnknownSnapshot, head)
		}
	}
	s.branches[name] = head
	return nil
}

// BranchHead returns the default agent's head for the named branch.
// Equivalent to BranchHeadForAgent(AgentDefault, name).
func (v *VST) BranchHead(name BranchID) (types.SnapshotID, bool) {
	return v.BranchHeadForAgent(AgentDefault, name)
}

// BranchHeadForAgent returns the named agent's head for the named branch.
// Returns ok=false if either the agent has no state yet or the branch is
// not registered under that agent.
func (v *VST) BranchHeadForAgent(agent AgentId, name BranchID) (types.SnapshotID, bool) {
	if validateAgent(agent) != nil {
		return "", false
	}
	v.mu.RLock()
	defer v.mu.RUnlock()
	s, ok := v.agentRO(agent)
	if !ok {
		return "", false
	}
	id, ok := s.branches[name]
	return id, ok
}

// Branches returns a snapshot of the default agent's branch map.
// Equivalent to BranchesForAgent(AgentDefault).
func (v *VST) Branches() map[BranchID]types.SnapshotID {
	return v.BranchesForAgent(AgentDefault)
}

// BranchesForAgent returns a snapshot of the named agent's branch map.
// The returned map is a defensive copy; mutating it does not affect VST state.
//
// Returns an empty (non-nil) map in two indistinguishable cases:
//   - the agent has never been seen by the VST (no agentState entry), or
//   - the agent has been written to but has not registered any branches.
//
// This is asymmetric with BranchHeadForAgent which returns ok=false on
// both miss cases. The contract is intentional: the public API never
// exposes whether an agent has internal state, only whether a specific
// branch / path is registered. Callers should not conflate "agent
// unknown" with "agent has zero branches" — the runtime treats both as
// "nothing to return" by design.
func (v *VST) BranchesForAgent(agent AgentId) map[BranchID]types.SnapshotID {
	if validateAgent(agent) != nil {
		return map[BranchID]types.SnapshotID{}
	}
	v.mu.RLock()
	defer v.mu.RUnlock()
	s, ok := v.agentRO(agent)
	if !ok {
		return map[BranchID]types.SnapshotID{}
	}
	out := make(map[BranchID]types.SnapshotID, len(s.branches))
	for k, val := range s.branches {
		out[k] = val
	}
	return out
}

// DeleteBranch removes a default-agent branch.
// Equivalent to DeleteBranchForAgent(AgentDefault, name).
func (v *VST) DeleteBranch(name BranchID) error {
	return v.DeleteBranchForAgent(AgentDefault, name)
}

// DeleteBranchForAgent removes the named branch from the named agent.
// Returns ErrUnknownBranch if the branch is not registered under that agent.
func (v *VST) DeleteBranchForAgent(agent AgentId, name BranchID) error {
	if err := validateAgent(agent); err != nil {
		return err
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	s, ok := v.agentRO(agent)
	if !ok {
		return ErrUnknownBranch
	}
	if _, ok := s.branches[name]; !ok {
		return ErrUnknownBranch
	}
	delete(s.branches, name)
	return nil
}
