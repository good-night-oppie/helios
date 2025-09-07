# /otw/mcts-implementation - Multi-Agent MCTS/LATS Implementation Orchestrator

## Triggers
- Initiating the Oppie Thunder MCTS implementation workflow
- Launching specific phases of the MCTS/LATS hybrid architecture
- Coordinating multi-agent tasks for AI engineering partner development
- Managing parallel execution of specialized AI agents

## Usage
```
/otw/mcts-implementation [phase|all|status] [--parallel] [--validate] [--dry-run]
```

## Phases
- `research` (Phase 1): Research & Architecture Foundation - chief-scientist-deepmind leads
- `engine` (Phase 2): Core Engine Implementation - alphazero-muzero-planner leads  
- `evolution` (Phase 3): Evolutionary Optimization - alphaevolve-scientist leads
- `safety` (Phase 4): Safety & Infrastructure - eval-safety-infra-gatekeeper leads
- `advanced` (Phase 5): Advanced Capabilities - alphafold2-structural-scientist leads
- `all`: Execute complete workflow sequentially
- `status`: Check current workflow progress and agent status

## Behavioral Flow

### Phase 1: Research Foundation
1. **Launch chief-scientist-deepmind**: Synthesize LATS/TS-LLM approaches
2. **Coordinate alphazero-muzero-planner**: Validate MCTS architecture
3. **Success Gate**: Theoretical proof of <5s iteration feasibility

### Phase 2: Core Engine
1. **Parallel Launch**:
   - alphazero-muzero-planner: LATS engine implementation
   - alphazero-muzero-planner: State management system
   - eval-safety-infra-gatekeeper: V8 isolate sandbox
2. **Success Gate**: 15-second iteration cycles achieved

### Phase 3: Evolution System
1. **Launch alphaevolve-scientist**: Trajectory replay buffer
2. **Implement**: DPO reward training
3. **Success Gate**: >1000 episodes collected

### Phase 4: Safety Infrastructure
1. **Launch eval-safety-infra-gatekeeper**: Production safety mechanisms
2. **Implement**: Deployment infrastructure
3. **Success Gate**: Security audit passed

### Phase 5: Advanced Features
1. **Launch alphafold2-structural-scientist**: Code structure analysis
2. **Implement**: Structure-aware planning
3. **Success Gate**: 30% code change reduction

## Agent Coordination

| Phase | Lead Agent | Supporting Agents | Key Deliverables |
|-------|------------|-------------------|------------------|
| 1 | chief-scientist-deepmind | alphazero-muzero-planner | Architecture spec |
| 2 | alphazero-muzero-planner | alphaevolve-scientist, eval-safety-infra-gatekeeper | LATS engine |
| 3 | alphaevolve-scientist | chief-scientist-deepmind | Learning system |
| 4 | eval-safety-infra-gatekeeper | All agents | Safety mechanisms |
| 5 | alphafold2-structural-scientist | alphazero-muzero-planner | Pattern library |

## MCP Integration
- **Sequential MCP**: Complex multi-step analysis for architecture design
- **Serena MCP**: Code structure analysis and symbol operations
- **Context7 MCP**: Framework patterns and best practices
- **Morphllm MCP**: Large-scale code transformations

## Tool Coordination
- **Task Tool**: Launch specialized agents with specific prompts
- **TodoWrite**: Track workflow progress and phase completion
- **Bash**: Execute TaskMaster commands and parallel operations
- **Read/Write**: Manage workflow documentation and results

## Key Patterns
- **Parallel Execution**: Phases 2-3 support parallel agent operations
- **Dependency Management**: TaskMaster tracks inter-task dependencies
- **Validation Gates**: Each phase has specific success criteria
- **Progressive Enhancement**: Build from LATS foundation to full MCTS

## Examples

### Start Research Phase
```
/otw/mcts-implementation research
# Launches chief-scientist-deepmind for theoretical foundation
# Creates architecture blueprint and validates feasibility
```

### Launch Core Engine (Parallel)
```
/otw/mcts-implementation engine --parallel
# Simultaneously launches:
# - LATS engine implementation
# - State management system
# - V8 sandbox infrastructure
```

### Execute Complete Workflow
```
/otw/mcts-implementation all --validate
# Runs all 5 phases sequentially
# Validates success gates between phases
# Generates comprehensive progress report
```

### Check Status
```
/otw/mcts-implementation status
# Shows current phase progress
# Lists active agent tasks
# Displays success metrics
```

## Boundaries

**Will:**
- Orchestrate multi-agent implementation of Oppie Thunder
- Manage parallel execution and dependencies
- Track progress through TaskMaster integration
- Validate phase completion criteria

**Will Not:**
- Directly implement code (agents handle implementation)
- Override safety validations
- Skip phases without explicit user confirmation
- Execute without proper agent definitions

## Success Metrics

```yaml
Performance_Targets:
  mcts_iteration_time: "<5s"
  state_latency:
    L0: "<100μs"
    L1: "<1ms"
    L2: "<5ms"
  
Quality_Metrics:
  test_coverage: "≥85%"
  security_score: "A+"
  code_review_time: "-50%"
  
Learning_Metrics:
  episodes_collected: ">1000"
  improvement_rate: ">5% per 100 episodes"
```

## Workflow Files

- **Workflow Definition**: `.claude/workflows/oppie-thunder-implementation.md`
- **Launch Script**: `.claude/workflows/launch-workflow.sh`
- **Agent Definitions**: `.claude/agents/*.md`
- **Progress Tracking**: TaskMaster database