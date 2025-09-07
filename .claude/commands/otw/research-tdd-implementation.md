# /otw/research-tdd-implementation - Research-Enhanced TDD Multi-Agent Orchestrator

## 🔴 SERENA IS THE PRIMARY CODING LEAD
**Serena MCP Server is the MASTER CONDUCTOR for all coding operations**. She knows the codebase to the bone through semantic analysis and leads all implementation phases.

## Triggers
- Starting any TaskMaster task with mandatory research phase
- Implementing features with research-first TDD workflow
- Coordinating research agents with implementation agents
- Managing the full Research → Red → Green → Refactor → Validate cycle

## Usage
```
/otw/research-tdd-implementation [task-id] [--parallel-research] [--depth deep|standard] [--validate-only]
```

## Core Workflow: Research → Red → Green → Refactor → Validate

### Phase 1: RESEARCH (Mandatory)
**Triggered automatically when task status changes to `in-progress`**

#### Serena-Led Codebase Analysis (FIRST)
1. **🔴 SERENA ANALYZES EXISTING CODE**: Understanding current architecture
   - `get_symbols_overview`: Map existing code structure
   - `find_symbol`: Locate relevant patterns and implementations
   - `find_referencing_symbols`: Understand dependencies
   - `list_memories`: Retrieve project knowledge

#### Parallel Research Execution (SECOND)
1. **Launch deep-researcher agent**: Industry best practices and case studies
2. **Launch Context7 queries**: Official documentation and API references  
3. **Launch DeepWiki searches**: Technical concepts and algorithms
4. **Launch Exa deep research**: Recent developments and benchmarks

#### Research Focus Areas
- **Architecture**: Design patterns, system architecture, component relationships
- **Performance**: Optimization techniques, benchmarks, bottlenecks
- **Security**: Vulnerabilities, authentication, authorization, data protection
- **Testing**: Test strategies, edge cases, property-based testing
- **Best Practices**: Industry standards, common patterns, anti-patterns

#### Research Output Recording
```bash
# Automatically record findings to task
task-master update-subtask --id=X.Y --prompt="
Research Findings:
- Architecture: [key patterns discovered]
- Performance: [optimization opportunities]
- Security: [considerations identified]
- Testing: [strategies to implement]
- Risks: [potential issues to avoid]
"
```

### Phase 2: RED (Test-First Development)
**Lead Agent**: SERENA coordinates with task-executor

#### Serena-Driven Test Creation
1. **🔴 SERENA LEADS TEST PLACEMENT**: 
   - `find_symbol`: Identify test file structure
   - `get_symbols_overview`: Understand test patterns
   - `insert_before_symbol`/`insert_after_symbol`: Place tests strategically
2. **Task-executor writes tests**: Based on Serena's codebase knowledge
3. **Include**: Edge cases from research + Serena's pattern analysis
4. **Add**: Security test cases identified by both research and Serena
5. **Implement**: Performance benchmarks from Serena's profiling
6. **Use**: Property-based testing for invariants

### Phase 3: GREEN (Implementation)
**Lead Agent**: SERENA as primary implementer

#### Serena-Led Implementation
1. **🔴 SERENA WRITES CODE**: Using semantic understanding
   - `replace_symbol_body`: Precise function implementations
   - `insert_after_symbol`: Add new methods/functions
   - `find_referencing_symbols`: Update all references correctly
2. **Apply**: Best practices from research + Serena's codebase patterns
3. **Avoid**: Anti-patterns identified by both research and Serena
4. **Include**: Security measures with Serena's validation
5. **Optimize**: Based on Serena's performance analysis

### Phase 4: REFACTOR (Quality Enhancement)
**Lead Agent**: SERENA orchestrates refactoring

#### Serena-Driven Refactoring
1. **🔴 SERENA REFACTORS SYSTEMATICALLY**:
   - `find_symbol`: Identify refactoring targets
   - `find_referencing_symbols`: Track all usages
   - `replace_symbol_body`: Apply refactoring precisely
   - `write_memory`: Document refactoring decisions
2. **Apply**: Design patterns validated by Serena
3. **Optimize**: Algorithms with Serena's performance insights
4. **Enhance**: Error handling using Serena's pattern library
5. **Improve**: Code organization based on Serena's analysis

### Phase 5: VALIDATE (Verification)
**Lead Agent**: SERENA validates with task-checker

#### Serena-Led Validation
1. **🔴 SERENA VALIDATES CODE QUALITY**:
   - `search_for_pattern`: Find potential issues
   - `find_symbol`: Verify implementation completeness
   - `think_about_collected_information`: Assess quality
   - `think_about_whether_you_are_done`: Final check
2. **Performance**: Serena verifies against benchmarks
3. **Security**: Serena scans for vulnerabilities
4. **Best Practices**: Serena checks pattern compliance
5. **Edge Cases**: Serena validates test coverage
6. **Coverage**: Serena ensures ≥85% test coverage

## Agent Orchestration with SERENA Leadership

| Phase | Lead Agent | Supporting Agents | Primary Tools | Deliverables |
|-------|------------|-------------------|---------------|--------------|
| Research | **🔴 SERENA** | deep-researcher, chief-scientist | Serena MCP, Context7, DeepWiki, Exa | Codebase analysis + Research doc |
| Red | **🔴 SERENA** | task-executor | Serena MCP (symbol ops), MultiEdit | Test suite |
| Green | **🔴 SERENA** | task-executor | Serena MCP (replace/insert), MultiEdit | Implementation |
| Refactor | **🔴 SERENA** | task-executor | Serena MCP (find/replace), Morphllm | Optimized code |
| Validate | **🔴 SERENA** | task-checker | Serena MCP (search/verify), Bash, Grep | Validation report |

## Specialized Agent Roles (All Under Serena's Leadership)

### 🔴 PRIMARY LEAD: SERENA MCP
- **Serena**: MASTER CONDUCTOR - Knows codebase to the bone
  - Leads ALL phases with semantic understanding
  - Coordinates other agents with codebase context
  - Provides symbol-level precision for all operations
  - Maintains project memory and patterns

### Research Phase Support Agents (Serena Coordinates)
- **deep-researcher**: Industry research (reports to Serena)
- **chief-scientist-deepmind**: Algorithm research (integrates with Serena)
- **alphazero-muzero-planner**: MCTS research (validated by Serena)
- **alphaevolve-scientist**: Evolutionary research (contextualized by Serena)
- **alphafold2-structural-scientist**: Pattern research (compared with Serena's analysis)

### Implementation Phase Support Agents (Serena Directs)
- **task-executor**: Executes under Serena's guidance
- **task-checker**: Validates using Serena's insights
- **eval-safety-infra-gatekeeper**: Security validation with Serena's context

### Support Agents (Serena Orchestrates)
- **task-orchestrator**: Parallel coordination under Serena
- **technical-writer**: Documents Serena's findings
- **quality-engineer**: Testing strategies from Serena's analysis

## MCP Integration (Serena-First Hierarchy)

### 🔴 PRIMARY MCP: SERENA (Used in ALL Phases)
- **get_symbols_overview**: Understand file structure FIRST
- **find_symbol**: Locate code entities with precision
- **find_referencing_symbols**: Track all usages and dependencies
- **replace_symbol_body**: Precise code modifications
- **insert_before_symbol/insert_after_symbol**: Strategic code placement
- **search_for_pattern**: Smart pattern matching
- **write_memory/read_memory**: Persistent knowledge management
- **think_about_* tools**: Meta-cognitive validation

### Research Phase Support MCPs
- **Context7**: Library docs (after Serena's codebase scan)
- **DeepWiki**: Concepts (contextualized by Serena)
- **Exa**: Web research (guided by Serena's findings)

### Implementation Phase Support MCPs
- **Morphllm**: Bulk transformations (directed by Serena)
- **Sequential**: Complex reasoning (informed by Serena)
- **Playwright**: E2E testing (targets from Serena)

## Examples

### Start Task with Full Research-TDD (Serena-Led)
```
/otw/research-tdd-implementation 12.6
# Automatically:
# 1. SERENA analyzes codebase structure FIRST
# 2. Launches parallel research agents with Serena's context
# 3. SERENA leads Red → Green → Refactor → Validate
# 4. SERENA validates completion and updates task status
```

### Deep Research for Complex Task
```
/otw/research-tdd-implementation 10.1 --depth deep
# Uses chief-scientist-deepmind for theoretical foundation
# Extends research phase with additional analysis
# Generates comprehensive research document
```

### Parallel Research Execution (Serena Coordinates)
```
/otw/research-tdd-implementation 14 --parallel-research
# SERENA coordinates parallel research:
# 1. SERENA scans codebase for context (FIRST)
# 2. Then launches simultaneously:
#    - Context7 for framework docs (with Serena's context)
#    - DeepWiki for algorithms (guided by Serena)
#    - Exa for industry practices (focused by Serena)
# 3. SERENA synthesizes all findings with codebase knowledge
```

### Validate Only (Skip Implementation)
```
/otw/research-tdd-implementation 12.5 --validate-only
# Runs task-checker on existing implementation
# Verifies against research criteria
# Updates task status based on validation
```

## Success Criteria

### Research Phase
- ≥3 credible sources identified
- Architecture patterns documented
- Performance benchmarks established
- Security considerations listed
- Test strategies defined

### Red Phase
- Test coverage plan ≥85%
- Edge cases from research included
- Performance benchmarks defined
- Security tests implemented

### Green Phase
- All tests passing
- Research insights applied
- No known anti-patterns used
- Security measures implemented

### Refactor Phase
- Code meets style guidelines
- Design patterns properly applied
- Performance optimized per research
- Maintainability improved

### Validate Phase
- Test coverage ≥85% achieved
- Performance targets met
- Security scan passed
- Best practices verified

## Memory Integration

```bash
# Workflow state persisted via Serena
write_memory("research_tdd_active", task_id)
write_memory("research_findings_[task_id]", findings)
write_memory("test_strategy_[task_id]", strategy)
write_memory("validation_results_[task_id]", results)
```

## Boundaries

**Will:**
- Enforce mandatory research phase for all tasks
- Coordinate multiple research sources in parallel
- Track complete TDD cycle through TaskMaster
- Validate against research-derived criteria

**Will Not:**
- Skip research phase (it's mandatory)
- Implement without tests (Red phase required)
- Deploy without validation
- Ignore security findings from research

## Integration with TaskMaster

```bash
# Automatic triggers
task-master set-status --id=X --status=in-progress
→ Triggers /otw/research-tdd-implementation X

# Progress tracking
task-master update-subtask --id=X.Y --prompt="[phase results]"

# Completion
task-master set-status --id=X --status=done
```

## Benefits of Research-First TDD

1. **Reduced Rework**: Issues discovered before implementation
2. **Better Architecture**: Informed design decisions
3. **Security by Design**: Vulnerabilities identified early
4. **Performance Optimization**: Bottlenecks known upfront
5. **Knowledge Transfer**: Research logged for team learning
6. **Quality Assurance**: Comprehensive test coverage
7. **Continuous Learning**: Patterns stored in project memory