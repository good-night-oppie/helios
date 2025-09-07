# Research-Enhanced TDD Workflow

## Core Principle
**MANDATORY**: Every task must begin with a Research Phase before any implementation.

## Workflow: Research → Red → Green → Refactor → Validate

### 1. RESEARCH PHASE (Required)
When setting a task to `in-progress`, automatically perform parallel research:

```bash
# Triggered by:
task-master set-status --id=X.Y --status=in-progress

# Research Tools (Execute in Parallel):
1. Context7 - Official documentation, API references, framework guides
2. DeepWiki - Technical concepts, algorithms, data structures
3. Exa Deep Research - Industry best practices, case studies, recent developments
```

### Research Focus Areas:
- **Architecture**: Design patterns, system architecture, component relationships
- **Performance**: Optimization techniques, benchmarks, bottlenecks
- **Security**: Vulnerabilities, authentication, authorization, data protection
- **Testing**: Test strategies, edge cases, property-based testing
- **Best Practices**: Industry standards, common patterns, anti-patterns

### Research Output:
```bash
# Record findings to task
task-master update-subtask --id=X.Y --prompt="
Research Findings:
- Architecture: [key patterns discovered]
- Performance: [optimization opportunities]
- Security: [considerations identified]
- Testing: [strategies to implement]
- Risks: [potential issues to avoid]
"
```

### 2. RED PHASE
Based on research, write comprehensive tests:
- Include edge cases discovered during research
- Add security-related test cases
- Implement performance benchmarks if applicable
- Use property-based testing for invariants

### 3. GREEN PHASE
Implement with research insights:
- Apply discovered best practices
- Avoid identified anti-patterns
- Include security measures from research
- Optimize based on performance findings

### 4. REFACTOR PHASE
Refine using research knowledge:
- Apply design patterns discovered
- Optimize algorithms based on research
- Enhance error handling
- Improve code organization

### 5. VALIDATE PHASE
Verify against research criteria:
- Performance meets researched benchmarks
- Security measures properly implemented
- Best practices followed
- All edge cases handled

## Memory Integration

This workflow is now part of the project's permanent memory and will be:
1. Automatically triggered when tasks begin
2. Enforced through hooks and CI/CD
3. Logged in task history for future reference
4. Used to build institutional knowledge

## Benefits of Research-First Approach

1. **Reduced Rework**: Discover issues before implementation
2. **Better Architecture**: Informed design decisions
3. **Security by Design**: Vulnerabilities identified early
4. **Performance Optimization**: Know bottlenecks upfront
5. **Knowledge Transfer**: Research logged for team learning

## Example: VST Implementation Research

```bash
# Research performed for Task 12.3:
Context7: "Arc smart pointers in Rust for zero-copy"
DeepWiki: "Trie data structures and path compression"
Exa: "Copy-on-write implementations in modern databases"

# Key Findings Applied:
- Use Arc<Node> for memory sharing between snapshots
- Implement path compression in trie for space efficiency
- Batch operations for atomic multi-file changes
- Lazy evaluation for performance optimization
```

---

*This enhanced workflow ensures every implementation is informed by current best practices and documented knowledge.*