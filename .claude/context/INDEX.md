# Oppie Thunder Context Index

## Available Context Documents

### Latest Context
- **[planner-engine-context-2025-08-30.md](./planner-engine-context-2025-08-30.md)**
  - Complete planner-engine MVP context
  - Architecture decisions (IR-only, MCTS-native)
  - Implementation status (all gates passing)
  - Future roadmap and technical debt

## Quick Access Points

### For New Agents Starting Work
1. Read the latest context document
2. Check current git status and branch
3. Review open issues/tasks in GitHub
4. Run `make test` to verify environment

### Key Architecture Decisions
- **IR-Only**: No DSL parsing, work directly with JSON
- **MCTS-Native**: Built for tree search from ground up
- **Hermetic MDP**: 100% deterministic execution
- **Skills System**: Focused action generation

### Performance Targets (All Achieved ✓)
- Parse: <10ms (achieved 3ms)
- Rollout: <2s (achieved 1.5s)
- Determinism: 100% (achieved)

### Critical Files
- `/ir/*.go` - Core implementation
- `/golden/*.go` - Test infrastructure
- `/cmd/planner/main.go` - CLI tool
- `Makefile` - CI/CD with 4 gates
- `MVP_SUMMARY.md` - User documentation

### Integration Points
- Helios engine (state management)
- MCP servers (Serena, Context7, etc.)
- GitHub Actions (CI/CD)

### Current Status
- ✅ MVP Complete
- ✅ All 4 CI/CD gates passing
- ✅ Performance targets met
- 🔄 Ready for backend integration
- 🔄 Ready for skill expansion

## Usage

For any agent working on planner-engine:
```bash
# Quick status check
cd /home/dev/workspace/oppie-thunder/planner-engine
git status
make test

# View context
cat /home/dev/workspace/oppie-thunder/.claude/context/planner-engine-context-2025-08-30.md
```

## Update Schedule
- After major features
- After architecture changes
- Weekly if active development
- Before handoffs between agents

---
*Last Updated: 2025-08-30*