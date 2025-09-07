# /otw/research-tdd-pr-review - Research-TDD with Automated PR Review & Debate

## Triggers
- Completing any TaskMaster task with mandatory PR review
- Tasks with complexity score ≥ 7/10
- Changes exceeding 500 lines or architectural decisions
- Security-critical or performance-critical implementations

## Usage
```
/otw/research-tdd-pr-review [task-id] [--complexity N] [--force-debate] [--skip-research]
```

## Enhanced Workflow: Research → Red → Green → Refactor → Validate → Commit → Review → Debate

### Phases 1-5: Standard Research-TDD
(Inherited from /otw/research-tdd-implementation)

### Phase 6: COMMIT & REVIEW (New)
**Triggered when validation passes**

#### Automatic Commit & Push
```bash
# Collect metrics
COVERAGE=$(go test -cover ./... | grep -o '[0-9]*\.[0-9]*%')
BENCHMARKS=$(go test -bench=. | grep "ns/op")
COMPLEXITY=${TASK_COMPLEXITY:-7}

# Create comprehensive commit
git add -A
git commit -m "feat: Complete Task ${TASK_ID} - ${TASK_TITLE}

Implementation Summary:
${RESEARCH_FINDINGS}

Architecture Decisions:
${ARCHITECTURE_CHOICES}

Performance Metrics:
${BENCHMARKS}

Test Coverage: ${COVERAGE}
Complexity Score: ${COMPLEXITY}/10

Generated with [Claude Code](https://claude.ai/code)
via [Happy](https://happy.engineering)

Co-Authored-By: Claude <noreply@anthropic.com>
Co-Authored-By: Happy <yesreply@happy.engineering>"

# Push to feature branch
git push origin feature/task-${TASK_ID}
```

#### Create PR with Context
```bash
gh pr create --title "Task ${TASK_ID}: ${TASK_TITLE}" \
  --body "$(generate_pr_description)"
```

#### Request Specialized Review
Based on task complexity, generate custom review prompt with explicit reviewer persona:

```bash
# Post initial PR review request with system prompt
gh pr comment ${PR_NUMBER} --body "@claude Please review this implementation.

## System Prompt for Review

You are acting as a ${REVIEWER_ROLE} conducting a rigorous architectural review for Task ${TASK_ID}.

### Your Role: ${REVIEWER_PERSONA}
${REVIEWER_DESCRIPTION}

### Review Mandate
- **Complexity**: ${COMPLEXITY}/10 - This is a ${COMPLEXITY_LEVEL} task
- **Focus Areas**: ${FOCUS_AREAS}
- **Success Criteria**: ${SUCCESS_CRITERIA}

### Required Analysis

1. **Specification Compliance**
   - Verify implementation against: ${SPEC_DOCUMENT}
   - Check TDD compliance: Research → Red → Green → Refactor → Validate
   - Validate clean room constraints (no blue_team references)

2. **Critical Evaluation**
   - Architectural decisions: ${KEY_DECISIONS}
   - Performance targets: ${PERFORMANCE_METRICS}
   - Trade-offs: ${TRADE_OFF_ANALYSIS}

3. **Expert Assessment as ${DOMAIN} Specialist**
   - Correctness: Line-by-line code review for complexity ≥7
   - Innovation: Evaluate ${NOVEL_APPROACHES}
   - Risks: Identify failure modes and edge cases

### Review Deliverables

Based on complexity ${COMPLEXITY}/10, provide:
${REQUIRED_DELIVERABLES}

### Debate Protocol
This review will involve ${EXPECTED_ROUNDS} rounds of debate:
- Round 1: Initial critical review (be skeptical, demand evidence)
- Round 2: Evidence-based response evaluation
- Round 3+: Synthesis and action items

${ADDITIONAL_INSTRUCTIONS}
"
```

##### Reviewer Persona Selection
```bash
# Select reviewer based on task complexity and domain
select_reviewer_persona() {
  local complexity=$1
  local domain=$2
  
  case $complexity in
    9|10)
      REVIEWER_ROLE="Chief Scientist (DeepMind-style)"
      REVIEWER_PERSONA="chief-scientist-deepmind"
      REVIEWER_DESCRIPTION="You are a world-class researcher with expertise in ${domain}. You demand rigorous proof for all claims, challenge assumptions, and require empirical validation. Be highly skeptical of performance claims without data."
      EXPECTED_ROUNDS="3-4"
      REQUIRED_DELIVERABLES="
- [ ] Correctness proof or counterexample
- [ ] Performance analysis with benchmarks
- [ ] Security threat model
- [ ] Complete alternative implementation if flawed
- [ ] Formal verification of critical paths"
      ;;
    7|8)
      REVIEWER_ROLE="Principal Engineer"
      REVIEWER_PERSONA="principal-engineer"
      REVIEWER_DESCRIPTION="You are a seasoned engineer with deep ${domain} expertise. Focus on practical trade-offs, maintainability, and production readiness. Question design decisions that add complexity."
      EXPECTED_ROUNDS="2-3"
      REQUIRED_DELIVERABLES="
- [ ] Design trade-off analysis
- [ ] Performance benchmark validation
- [ ] Code quality assessment
- [ ] Production readiness checklist"
      ;;
    5|6)
      REVIEWER_ROLE="Senior Developer"
      REVIEWER_PERSONA="senior-developer"
      REVIEWER_DESCRIPTION="You are an experienced developer reviewing for correctness and best practices in ${domain}."
      EXPECTED_ROUNDS="1-2"
      REQUIRED_DELIVERABLES="
- [ ] Code correctness verification
- [ ] Test coverage analysis
- [ ] Best practices compliance"
      ;;
    *)
      REVIEWER_ROLE="Code Reviewer"
      REVIEWER_PERSONA="standard-reviewer"
      REVIEWER_DESCRIPTION="Standard code review focusing on functionality and quality."
      EXPECTED_ROUNDS="1"
      REQUIRED_DELIVERABLES="
- [ ] Basic functionality verification
- [ ] Code style compliance"
      ;;
  esac
  
  # Domain-specific additions
  case $domain in
    "performance"|"optimization")
      ADDITIONAL_INSTRUCTIONS="
**Performance Focus**: Demand empirical evidence for all optimization claims. Request benchmarks comparing before/after. Challenge premature optimizations. Verify no regressions in other metrics."
      FOCUS_AREAS="Lock-free algorithms, cache optimization, memory pooling, parallel processing"
      ;;
    "security")
      ADDITIONAL_INSTRUCTIONS="
**Security Focus**: Assume adversarial mindset. Look for injection points, race conditions, privilege escalations. Demand threat model documentation."
      FOCUS_AREAS="Input validation, authentication, authorization, cryptography"
      ;;
    "architecture")
      ADDITIONAL_INSTRUCTIONS="
**Architecture Focus**: Evaluate long-term maintainability, scalability, and evolvability. Question unnecessary complexity. Verify SOLID principles."
      FOCUS_AREAS="System design, component boundaries, dependency management"
      ;;
    "algorithm")
      ADDITIONAL_INSTRUCTIONS="
**Algorithm Focus**: Verify correctness proofs, complexity analysis, edge cases. Demand formal verification for critical paths."
      FOCUS_AREAS="Computational complexity, correctness, optimization"
      ;;
    *)
      ADDITIONAL_INSTRUCTIONS=""
      FOCUS_AREAS="General code quality and correctness"
      ;;
  esac
}
```

### Phase 7: DEBATE & REFINEMENT
**Adaptive debate rounds based on complexity and review feedback**

#### Debate Trigger Matrix

| Condition | Min Rounds | Max Rounds | Focus Areas | Approval Required |
|-----------|------------|------------|-------------|-------------------|
| Complexity ≥ 9/10 | 3-4 | Until Approved | Architecture, Performance, Security | Explicit "APPROVED" or "READY FOR MERGE" |
| Complexity 7-8/10 | 2-3 | Until Approved | Trade-offs, Implementation | Explicit approval statement |
| Questions > 25% | 3+ | Until Resolved | Justification, Alternatives | All questions answered + approval |
| Performance Issues | 2+ | Until Fixed | Optimization | Performance validated + approval |
| Security Concerns | 3+ | Until Secure | Threat Model | Security verified + approval |

**CRITICAL**: Monitoring continues UNTIL explicit approval, regardless of round count

#### Debate Protocol

**Round 1: Initial Review (0-24h)**
```bash
# Monitor for Claude's review AND CI status
monitor_pr_review ${PR_NUMBER} &
REVIEW_PID=$!

# Start CI monitoring with auto-fix
/home/dev/workspace/oppie-autonav/scripts/git-push-with-ci-monitor.sh pr ${PR_NUMBER} &
CI_PID=$!

# Wait for both to complete
wait $REVIEW_PID
wait $CI_PID

# Parse review feedback
QUESTIONS=$(parse_review_questions)
CONCERNS=$(parse_review_concerns)
CI_STATUS=$(gh pr checks ${PR_NUMBER} --json conclusion -q '.[].conclusion' | grep -c "SUCCESS" || echo 0)
TOTAL_CHECKS=$(gh pr checks ${PR_NUMBER} --json conclusion -q '.[].conclusion' | wc -l)

# Trigger debate if needed
if [[ $QUESTIONS > 25% ]] || [[ $COMPLEXITY >= 7 ]] || [[ $CI_STATUS -ne $TOTAL_CHECKS ]]; then
  trigger_debate_round 2
fi
```

**Round 2: Evidence-Based Response (24-48h)**
```bash
# Prepare evidence
collect_benchmarks > evidence/benchmarks.md
collect_test_results > evidence/tests.md
generate_architecture_diagrams > evidence/architecture.md

# Post response with evidence
gh pr comment ${PR_NUMBER} --body "@claude 
Round 2 Response:

Evidence Supporting Implementation:
- Benchmarks: [link]
- Test Coverage: [link]
- Architecture: [link]

Addressing Concerns:
${POINT_BY_POINT_RESPONSE}

Questions for Clarification:
${CLARIFYING_QUESTIONS}
"

# Request specialized agent if needed
if [[ $DOMAIN == "algorithm" ]]; then
  request_agent alphazero-muzero-planner
fi
```

**Round 3: Synthesis & Action Items (48-72h)**
```bash
# Synthesize agreements
AGREEMENTS=$(extract_agreements)
DISAGREEMENTS=$(extract_disagreements)
ACTION_ITEMS=$(generate_action_items)

# Document decisions
write_memory("debate_outcome_${TASK_ID}", {
  agreements: $AGREEMENTS,
  disagreements: $DISAGREEMENTS,
  action_items: $ACTION_ITEMS
})

# Create follow-up tasks
for item in $ACTION_ITEMS; do
  task-master add-task --prompt="$item" --dependencies="${TASK_ID}"
done
```

**Round 4+: Escalation (if needed)**
```bash
# Bring in human reviewers
gh pr edit ${PR_NUMBER} --add-reviewer "@team-lead,@architect"

# Schedule sync discussion
create_calendar_event "Architecture Review: Task ${TASK_ID}"

# Document in ADR
create_adr "decisions/adr-${TASK_ID}.md"
```

## Integration with oppie-autonav

### PR Monitor Integration
The command uses the existing oppie-autonav infrastructure:

```bash
# Located at: /home/dev/workspace/oppie-autonav/hooks/pr-review/pr-monitor.sh
OPPIE_AUTONAV_PATH="/home/dev/workspace/oppie-autonav"
PR_MONITOR="$OPPIE_AUTONAV_PATH/hooks/pr-review/pr-monitor.sh"

# Start monitoring with specialized parameters
"$PR_MONITOR" monitor "$PR_NUMBER" "$TASK_COMPLEXITY"
```

### CI Monitoring Integration
```bash
# Uses oppie-autonav CI monitoring
CI_MONITOR="$OPPIE_AUTONAV_PATH/scripts/git-push-with-ci-monitor.sh"

# Automatic CI monitoring with push
git push origin "feature/task-${TASK_ID}"
# This triggers automatic CI monitoring via git hooks
```

### Debate Response Automation
The oppie-autonav daemon handles:
- **Comment Detection**: Monitors for @claude responses
- **Response Analysis**: Categorizes approval, concerns, questions
- **Evidence Collection**: Runs benchmarks, tests, performance metrics
- **Auto-Response**: Generates evidence-based responses
- **State Management**: Tracks debate rounds and approval status

## Success Metrics

### Review Quality
- **Response Time**: < 2h for initial review
- **Depth**: Line-by-line for complexity ≥ 7
- **Evidence**: Benchmarks + tests for all claims

### Debate Effectiveness
- **Consensus Rate**: > 80% within 3 rounds
- **Action Items**: Average 2-3 per debate
- **Knowledge Capture**: 100% decisions documented

### Implementation Impact
- **Bug Detection**: > 90% before merge
- **Performance**: No regressions vs baseline
- **Architecture**: All decisions traced to requirements

## Example Execution

### Task 12.7: Performance Optimization (Complexity 9/10)
```bash
# Step 1: Complete implementation with TDD
/otw/research-tdd-implementation 12.7

# Step 2: Create PR and initiate review with monitoring
/otw/research-tdd-pr-review 12.7 --complexity 9 --domain performance

# What happens automatically:
1. ✅ Commits with comprehensive message
2. ✅ Creates PR with research context
3. ✅ Posts specialized review request with Chief Scientist persona
4. 🔄 **ACTIVELY MONITORS PR for Claude's responses**
5. 🔄 **AUTO-RESPONDS to Claude's reviews with evidence**
6. 🔄 **CONTINUES DEBATE for 3-4 rounds as needed**
7. ✅ Documents all architectural decisions
8. ✅ Marks task complete when approved

# Step 3: Monitor the PR actively (if not already running)
~/workspace/oppie-autonav/hooks/pr-review/pr-monitor.sh monitor 27 9

# The monitor will:
- Check every 2 minutes for new comments
- Detect Claude's responses from GitHub Actions
- Automatically generate evidence-based responses
- Post responses and continue debate
- Mark task complete when approved
- Exit after 1 hour of inactivity or approval
```

### Active Monitoring Workflow
```bash
# After creating PR and posting @claude review request:

# Option 1: Full automated workflow (recommended)
/otw/research-tdd-pr-review 12.7 --complexity 9 --monitor

# Option 2: Manual monitoring (if workflow interrupted)
~/workspace/oppie-autonav/hooks/pr-review/pr-monitor.sh monitor PR_NUMBER TASK_ID COMPLEXITY

# Option 3: Check monitoring status
~/workspace/oppie-autonav/hooks/pr-review/pr-monitor.sh status

# Option 4: Resume monitoring after interruption
~/workspace/oppie-autonav/hooks/pr-review/pr-monitor.sh monitor 27 9  # Resume monitoring
```

### What the Monitor Does
1. **Detects Claude's Comments**: Checks for responses from github-actions[bot] or @claude mentions
2. **Analyzes Response Type**: 
   - 🔴 Critical issues → Generates defense with benchmarks
   - 🟡 Questions → Provides clarifications
   - ✅ Approved → Marks task complete
3. **Collects Evidence**: Runs benchmarks, race tests, profiling
4. **Posts Responses**: Automatically continues debate
5. **Tracks State**: Maintains debate round, last comment ID
6. **Handles Completion**: Updates TaskMaster when approved

## Integration Points

### TaskMaster
- Auto-updates task status throughout workflow
- Creates follow-up tasks from action items
- Tracks debate outcomes in task details

### Serena Memory
- Stores debate transcripts
- Maintains decision log
- Preserves architectural rationale

### GitHub Integration
- PR creation and management
- Review request automation
- Comment thread management

### oppie-autonav Daemon
- Specialized review based on domain
- Multi-agent debate coordination
- Evidence collection and analysis
- CI monitoring with auto-fix

## Boundaries

**Will:**
- Enforce PR review for complex tasks
- Facilitate evidence-based debates
- Document all architectural decisions
- Escalate when consensus not reached

**Will Not:**
- Merge without review approval
- Skip debate for high-complexity tasks
- Ignore security or performance concerns
- Make decisions without evidence