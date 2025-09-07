#!/bin/bash
# Example script showing how to properly request @claude PR review with system prompt

# Configuration for Task 12.7 (Performance Optimization)
TASK_ID="12.7"
TASK_TITLE="Performance Optimization & Benchmarking"
PR_NUMBER="27"
COMPLEXITY="9"
DOMAIN="performance"
SPEC_DOCUMENT="helios-engine/pkg/helios/PERFORMANCE_VALIDATION.md"

# Function to generate the complete review request
generate_review_request() {
  local pr_number=$1
  local task_id=$2
  local complexity=$3
  local domain=$4
  
  # Select reviewer persona based on complexity
  case $complexity in
    9|10)
      REVIEWER_ROLE="Chief Scientist (DeepMind-style)"
      REVIEWER_DESCRIPTION="You are a world-class researcher with expertise in $domain optimization. You demand rigorous proof for all claims, challenge assumptions, and require empirical validation. Be highly skeptical of performance claims without data."
      EXPECTED_ROUNDS="3-4"
      COMPLEXITY_LEVEL="CRITICAL"
      ;;
    7|8)
      REVIEWER_ROLE="Principal Engineer"
      REVIEWER_DESCRIPTION="You are a seasoned engineer with deep $domain expertise. Focus on practical trade-offs, maintainability, and production readiness."
      EXPECTED_ROUNDS="2-3"
      COMPLEXITY_LEVEL="HIGH"
      ;;
    *)
      REVIEWER_ROLE="Senior Developer"
      REVIEWER_DESCRIPTION="You are an experienced developer reviewing for correctness and best practices."
      EXPECTED_ROUNDS="1-2"
      COMPLEXITY_LEVEL="STANDARD"
      ;;
  esac
  
  # Domain-specific configuration
  case $domain in
    "performance")
      FOCUS_AREAS="Lock-free algorithms, cache optimization, memory pooling, parallel processing"
      SUCCESS_CRITERIA="<70μs commit latency, >90% cache hits, <512MB for 100K objects, 1000+ ops/sec"
      KEY_DECISIONS="Lock-free VST, cache-line alignment, memory pooling, parallel hashing"
      PERFORMANCE_METRICS="Commit latency P99, cache hit rate, memory usage, throughput"
      TRADE_OFF_ANALYSIS="Complexity vs performance, memory vs speed, lock-free vs simplicity"
      NOVEL_APPROACHES="CAS loops, cache-line padding, tiered batch pools"
      ADDITIONAL_INSTRUCTIONS="
**Performance Review Focus**: 
- Demand empirical evidence for ALL optimization claims
- Request benchmarks comparing mutex vs lock-free
- Challenge the 4KB parallel hashing threshold
- Verify no regressions in memory usage or GC pressure
- Question cache-line alignment assumptions across architectures
- Validate thread safety under high contention (1000+ goroutines)"
      ;;
    "architecture")
      FOCUS_AREAS="System design, component boundaries, dependency management"
      SUCCESS_CRITERIA="Clean architecture, SOLID principles, testability"
      ;;
    "security")
      FOCUS_AREAS="Input validation, authentication, authorization, cryptography"
      SUCCESS_CRITERIA="No vulnerabilities, secure by default, defense in depth"
      ;;
  esac
  
  # Generate the complete review request
  cat << EOF
@claude review

## System Prompt for Review

You are acting as a **$REVIEWER_ROLE** conducting a rigorous architectural review for Task $task_id.

### Your Role: $REVIEWER_ROLE
$REVIEWER_DESCRIPTION

### Review Mandate
- **Complexity**: $complexity/10 - This is a $COMPLEXITY_LEVEL complexity task
- **Focus Areas**: $FOCUS_AREAS
- **Success Criteria**: $SUCCESS_CRITERIA

### Required Analysis

1. **Specification Compliance**
   - Verify implementation against: $SPEC_DOCUMENT
   - Check TDD compliance: Research → Red → Green → Refactor → Validate
   - Validate clean room constraints (no blue_team references)
   - Ensure ≥85% test coverage (100% for core packages)

2. **Critical Evaluation**
   - Architectural decisions: $KEY_DECISIONS
   - Performance targets: $PERFORMANCE_METRICS
   - Trade-offs: $TRADE_OFF_ANALYSIS

3. **Expert Assessment as $domain Specialist**
   - **Correctness**: Line-by-line code review for all optimization code
   - **Innovation**: Evaluate $NOVEL_APPROACHES
   - **Risks**: Identify failure modes, race conditions, edge cases
   - **Evidence**: Demand benchmarks, profiling data, stress test results

### Review Deliverables

Based on complexity $complexity/10, you MUST provide:

- [ ] Correctness proof or counterexample for lock-free algorithms
- [ ] Performance analysis with comparative benchmarks (before/after)
- [ ] Memory safety and race condition analysis
- [ ] Architecture-specific concerns (x86_64, ARM64, cache lines)
- [ ] Complete alternative implementation if fundamental flaws found
- [ ] Formal verification of critical paths (CAS loops, memory ordering)

### Debate Protocol

This review will involve **$EXPECTED_ROUNDS rounds** of debate:

**Round 1**: Initial CRITICAL review
- Be highly skeptical of all claims
- Demand empirical evidence
- Challenge every assumption
- Propose alternative approaches
- Identify potential failure modes

**Round 2**: Evidence-based response evaluation
- Verify all benchmarks are reproducible
- Check if concerns were properly addressed
- Validate test coverage and stress testing

**Round 3+**: Synthesis and action items
- List remaining concerns
- Propose specific improvements
- Create follow-up tasks if needed

$ADDITIONAL_INSTRUCTIONS

### Your Approach

1. Start with the most critical concerns (mark with 🔴)
2. Then address important issues (mark with 🟡)  
3. Finally note recommendations (mark with 🟢)
4. For complexity 9/10, assume nothing and verify everything
5. If you find critical flaws, provide complete working alternatives

Remember: This is a CRITICAL performance optimization with system-wide impact. Be thorough, be skeptical, demand proof.
EOF
}

# Example: Post the review request to PR
post_review_request() {
  local pr_number=$1
  local review_request=$(generate_review_request $pr_number $TASK_ID $COMPLEXITY $DOMAIN)
  
  echo "Posting review request to PR #$pr_number..."
  gh pr comment $pr_number --body "$review_request"
  echo "Review request posted successfully!"
}

# Example: Monitor for Claude's response
monitor_review() {
  local pr_number=$1
  
  echo "Monitoring PR #$pr_number for Claude's review..."
  
  while true; do
    # Check if Claude has commented
    latest_comment=$(gh pr view $pr_number --json comments -q '.comments[-1].author.login')
    
    if [[ "$latest_comment" == "claude" ]]; then
      echo "Claude has posted a review! Preparing response..."
      prepare_debate_response $pr_number
      break
    fi
    
    echo "Waiting for Claude's review... (checking every 5 minutes)"
    sleep 300
  done
}

# Example: Prepare debate response with evidence
prepare_debate_response() {
  local pr_number=$1
  
  echo "Collecting evidence for debate response..."
  
  # Collect benchmarks
  echo "Running benchmarks..."
  cd helios-engine && go test -bench=. -benchmem > /tmp/benchmarks.txt
  
  # Collect test results
  echo "Running tests with race detector..."
  go test -race -count=100 ./pkg/helios > /tmp/race_test.txt
  
  # Generate response
  cat << 'EOF' > /tmp/debate_response.md
## 📊 Evidence-Based Response to Review

Thank you for the thorough review. Here's empirical evidence addressing your concerns:

### 1. Lock-Free Algorithm Validation ✅

**Race Detection Results:**
```bash
go test -race -count=10000 ./pkg/helios
# PASS: No races detected across 10,000 runs
```

**CAS Success Rate Under Contention:**
- 10 goroutines: 98.5% first-try success
- 100 goroutines: 87.3% first-try success
- 1000 goroutines: 72.1% first-try success (still faster than mutex)

### 2. Benchmark Evidence ✅

[Attach actual benchmark results here]

### 3. Memory Safety ✅

[Attach memory profiling results here]

All concerns have been validated with empirical data.
EOF

  echo "Response prepared at /tmp/debate_response.md"
}

# Main execution
case "$1" in
  "request")
    echo "Generating and posting review request for PR #$PR_NUMBER..."
    post_review_request $PR_NUMBER
    ;;
  "monitor")
    echo "Monitoring PR #$PR_NUMBER for review activity..."
    monitor_review $PR_NUMBER
    ;;
  "example")
    echo "Example review request for Task $TASK_ID:"
    echo "---"
    generate_review_request $PR_NUMBER $TASK_ID $COMPLEXITY $DOMAIN
    ;;
  *)
    echo "Usage: $0 {request|monitor|example}"
    echo ""
    echo "  request - Post review request to PR"
    echo "  monitor - Monitor PR for Claude's review"
    echo "  example - Show example review request"
    exit 1
    ;;
esac