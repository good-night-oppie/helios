# OTW (Oppie Thunder Workflow) Commands Documentation

## Overview

The OTW commands are sophisticated workflow orchestrators that combine research, test-driven development, multi-agent coordination, and automated PR review with debate mechanisms. These commands ensure high-quality implementations through mandatory research phases and collaborative review cycles.

## Available Commands

### 1. `/otw/mcts-implementation`
**Purpose**: Orchestrate multi-phase MCTS/LATS implementation with specialized agents

**Usage**:
```bash
/otw/mcts-implementation [phase|all|status] [--parallel] [--validate] [--dry-run]
```

**Phases**:
- `research` - Research & Architecture Foundation (chief-scientist-deepmind leads)
- `engine` - Core Engine Implementation (alphazero-muzero-planner leads)
- `evolution` - Evolutionary Optimization (alphaevolve-scientist leads)
- `safety` - Safety & Infrastructure (eval-safety-infra-gatekeeper leads)
- `advanced` - Advanced Capabilities (alphafold2-structural-scientist leads)
- `all` - Execute complete workflow sequentially
- `status` - Check current workflow progress

**Example**:
```bash
# Start research phase
/otw/mcts-implementation research

# Launch core engine with parallel execution
/otw/mcts-implementation engine --parallel

# Check progress
/otw/mcts-implementation status
```

### 2. `/otw/research-tdd-implementation`
**Purpose**: Execute research-enhanced TDD workflow with mandatory research before implementation

**Usage**:
```bash
/otw/research-tdd-implementation [task-id] [--parallel-research] [--depth deep|standard] [--validate-only]
```

**Workflow Phases**:
1. **RESEARCH** - Parallel execution of Context7, DeepWiki, Exa research
2. **RED** - Write comprehensive tests based on research
3. **GREEN** - Implement with research insights
4. **REFACTOR** - Apply discovered patterns
5. **VALIDATE** - Verify against research criteria

**Options**:
- `--parallel-research` - Execute all research agents simultaneously
- `--depth deep` - Extended research with chief-scientist-deepmind
- `--validate-only` - Skip implementation, only validate existing code

**Example**:
```bash
# Standard research-TDD for a task
/otw/research-tdd-implementation 12.6

# Deep research for complex task
/otw/research-tdd-implementation 10.1 --depth deep

# Validate existing implementation
/otw/research-tdd-implementation 12.5 --validate-only
```

### 3. `/otw/research-tdd-pr-review`
**Purpose**: Complete research-TDD workflow with automated PR creation and sophisticated review debates

**Usage**:
```bash
/otw/research-tdd-pr-review [task-id] [--complexity N] [--force-debate] [--skip-research]
```

**Enhanced Workflow** (Phases 1-5 from research-tdd + new phases):
6. **COMMIT & REVIEW** - Automatic commit, push, and PR creation
7. **DEBATE & REFINEMENT** - Adaptive debate rounds based on complexity

**Options**:
- `--complexity N` - Set task complexity (1-10) to determine debate rounds
- `--force-debate` - Force maximum debate rounds regardless of complexity
- `--skip-research` - Skip research phase (not recommended)

**Debate Rounds by Complexity**:
| Complexity | Rounds | Review Focus |
|-----------|--------|--------------|
| 9-10 | 3-4 | Architecture, proofs, security |
| 7-8 | 2-3 | Trade-offs, evidence, alternatives |
| 5-6 | 1-2 | Best practices, quality |
| <5 | 1 | Quick review |

**Example**:
```bash
# Full workflow with PR review for complex task
/otw/research-tdd-pr-review 12.5 --complexity 8

# Force extended debate for critical component
/otw/research-tdd-pr-review 14 --complexity 10 --force-debate
```

## Command Selection Guide

### When to Use Each Command

**Use `/otw/mcts-implementation` when**:
- Implementing the core MCTS/LATS engine components
- Need multi-agent coordination for different phases
- Working on algorithmic or architectural foundations
- Require specialized expertise (planning, evolution, safety)

**Use `/otw/research-tdd-implementation` when**:
- Implementing any feature that needs research
- Want TDD with research-informed tests
- Need parallel research from multiple sources
- Don't need automated PR review

**Use `/otw/research-tdd-pr-review` when**:
- Working on complex tasks (complexity ≥ 7)
- Need automated PR creation and review
- Want collaborative debate with AI reviewers
- Require decision documentation and follow-up tasks

## Configuration Files

### Debate Protocol Configuration
**Location**: `.claude/config/debate-protocol.yaml`

Configures:
- Complexity thresholds for debate rounds
- Specialized agent assignments
- Escalation rules
- Review templates per complexity level

### Review Templates
**Location**: `.claude/templates/review/`

Available templates:
- `review-collaborative-critical.md` - Challenges assumptions, seeks deeper insights
- `review-storage.md` - Storage system specific review
- `review-algorithm.md` - Algorithm and complexity analysis
- `review-architecture.md` - System design review

### Scripts
**Location**: `.claude/scripts/`

- `request-pr-review.sh` - Generates sophisticated PR review requests
- `monitor_debate.sh` - Monitors and manages debate rounds

## Integration with TaskMaster

All OTW commands integrate with TaskMaster:

```bash
# Commands automatically update task status
task-master set-status --id=X --status=in-progress
# → Triggers /otw/research-tdd-implementation

# Research findings are recorded
task-master update-subtask --id=X.Y --prompt="Research findings..."

# Completion updates
task-master set-status --id=X --status=done
```

## Best Practices

### 1. Always Start with Research
Even for "simple" tasks, the research phase often reveals:
- Better approaches
- Hidden complexity
- Security considerations
- Performance optimizations

### 2. Set Accurate Complexity
Complexity determines review depth:
```bash
# Be honest about complexity
/otw/research-tdd-pr-review 12.5 --complexity 8  # Complex storage system
/otw/research-tdd-pr-review 10.2 --complexity 5  # Standard feature
```

### 3. Engage in Debates
Don't just accept or reject feedback:
- Provide evidence for your decisions
- Challenge reviewer assumptions
- Offer counter-proposals
- Document agreements and disagreements

### 4. Document Decisions
All architectural decisions from debates should be:
- Recorded in task details
- Saved to Serena memory
- Created as ADRs (Architecture Decision Records)

### 5. Create Follow-up Tasks
Action items from reviews become new tasks:
```bash
# Automatically created from debate outcomes
task-master add-task --prompt="Optimize L2 cache based on review feedback"
```

## Example Workflow Execution

### Complete Example: Task 12.5 (L2 Storage)

```bash
# 1. Start with research-TDD and PR review
/otw/research-tdd-pr-review 12.5 --complexity 8

# This will:
# - Launch parallel research (Context7, DeepWiki, Exa)
# - Record findings to task
# - Guide TDD implementation
# - Run tests and benchmarks
# - Commit with comprehensive message
# - Create PR with context
# - Request sophisticated review from @claude
# - Monitor 2-3 debate rounds
# - Document decisions
# - Create follow-up tasks

# 2. Monitor debate progress
gh pr view <PR_NUMBER> --comments

# 3. Respond to review with evidence
./scripts/prepare_debate_response.sh <PR_NUMBER> 2

# 4. After consensus, merge
gh pr merge <PR_NUMBER>

# 5. Mark task complete
task-master set-status --id=12.5 --status=done
```

## Troubleshooting

### Research Phase Hanging
If research agents don't respond:
```bash
# Check agent status
Task status

# Retry with standard depth
/otw/research-tdd-implementation <task-id> --depth standard
```

### PR Review Not Triggering
If Claude doesn't respond to review request:
```bash
# Manually trigger review
./scripts/request-pr-review.sh <PR_NUMBER> <TASK_ID> <COMPLEXITY>

# Check GitHub Actions
gh run list --workflow=claude-review
```

### Debate Not Progressing
If debate stalls:
```bash
# Force next round
/otw/research-tdd-pr-review debate --pr <PR_NUMBER> --round <N>

# Escalate to human review
gh pr edit <PR_NUMBER> --add-reviewer @team-lead
```

## Advanced Usage

### Custom Review Prompts
Create task-specific review templates:
```bash
# Create custom template
cat > .claude/templates/review/review-task-${TASK_ID}.md

# Use in review
/otw/research-tdd-pr-review ${TASK_ID} --template custom
```

### Multi-Agent Debates
For critical decisions, involve multiple agents:
```bash
# Request reviews from multiple specialists
/otw/research-tdd-pr-review ${TASK_ID} --agents "alphazero-muzero-planner,chief-scientist-deepmind"
```

### Cross-Task Dependencies
Handle dependent tasks:
```bash
# Complete prerequisite with validation
/otw/research-tdd-implementation 12.4 --validate-only

# Then proceed with dependent task
/otw/research-tdd-pr-review 12.5 --complexity 8
```

## Summary

The OTW commands transform task implementation from simple coding to comprehensive engineering:

1. **Research First**: Every implementation informed by current best practices
2. **Test Driven**: Tests based on research findings, not assumptions
3. **Automated Review**: Sophisticated PR reviews that challenge thinking
4. **Collaborative Debate**: Not approval/rejection, but exploration
5. **Continuous Learning**: Each debate improves future implementations

Use these commands to ensure every line of code is backed by research, tested thoroughly, and reviewed critically.