#!/bin/bash
# SPDX-FileCopyrightText: 2025 Good Night Oppie  
# SPDX-License-Identifier: MIT
# 
# Research-TDD Workflow Validation
# Validates that commits follow Research → Red → Green → Refactor → Validate cycle

set -euo pipefail

readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m'

# Track validation results
validation_score=0
total_checks=0

# Increment check counters
check() {
    total_checks=$((total_checks + 1))
    if [ $1 -eq 0 ]; then
        validation_score=$((validation_score + 1))
    fi
}

# Check for research evidence
check_research_phase() {
    echo "🔍 Validating Research Phase..."
    
    local research_indicators=0
    local changed_files=$(git diff --cached --name-only)
    
    # Check for research documentation
    local research_docs=$(echo "$changed_files" | grep -E "research|analysis|spec|requirements" || true)
    if [ -n "$research_docs" ]; then
        echo -e "${GREEN}  ✅ Research documentation present${NC}"
        research_indicators=$((research_indicators + 1))
    fi
    
    # Check commit message for research references
    local commit_msg=$(git log --format=%B -n 1 HEAD 2>/dev/null || echo "")
    if echo "$commit_msg" | grep -qi "research\|analysis\|specification\|behavior"; then
        echo -e "${GREEN}  ✅ Research references in commit message${NC}"
        research_indicators=$((research_indicators + 1))
    fi
    
    # Check for external research tools usage
    if echo "$commit_msg" | grep -qi "context7\|deepwiki\|exa\|research-driven"; then
        echo -e "${GREEN}  ✅ External research tools referenced${NC}"
        research_indicators=$((research_indicators + 1))
    fi
    
    if [ $research_indicators -gt 0 ]; then
        echo -e "${GREEN}Research Phase: VALIDATED${NC}"
        check 0
    else
        echo -e "${YELLOW}Research Phase: WEAK (consider documenting research)${NC}"
        check 1
    fi
}

# Check Red phase - tests written first
check_red_phase() {
    echo "🔴 Validating Red Phase (Tests First)..."
    
    local test_files=$(git diff --cached --name-only | grep -E "_test\.go$|\.test\.(ts|js)$|test\.py$" || true)
    
    if [ -n "$test_files" ]; then
        echo -e "${GREEN}  ✅ Test files modified/added${NC}"
        
        # Check if tests were added before implementation
        local test_lines=$(git diff --cached --numstat | grep -E "_test\.go$|\.test\.(ts|js)$" | awk '{sum += $1} END {print sum+0}')
        local code_lines=$(git diff --cached --numstat | grep -E "\.go$|\.ts$|\.js$" | grep -v -E "_test\.go$|\.test\.(ts|js)$" | awk '{sum += $1} END {print sum+0}')
        
        if [ "${test_lines:-0}" -gt 0 ]; then
            echo -e "${GREEN}  ✅ Tests have substantial additions (${test_lines} lines)${NC}"
            echo -e "${GREEN}Red Phase: VALIDATED${NC}"
            check 0
        else
            echo -e "${YELLOW}  ⚠️  Test changes minimal${NC}"
            echo -e "${YELLOW}Red Phase: WEAK${NC}"
            check 1
        fi
    else
        local code_changes=$(git diff --cached --name-only | grep -E "\.go$|\.ts$|\.js$" | wc -l)
        if [ "$code_changes" -gt 0 ]; then
            echo -e "${YELLOW}  ⚠️  Code changes without corresponding tests${NC}"
            echo -e "${YELLOW}Red Phase: NOT DETECTED${NC}"
            check 1
        else
            echo -e "${BLUE}  ℹ️  No code changes detected${NC}"
            check 0
        fi
    fi
}

# Check Green phase - minimal implementation
check_green_phase() {
    echo "🟢 Validating Green Phase (Minimal Implementation)..."
    
    local code_files=$(git diff --cached --name-only | grep -E "\.go$|\.ts$|\.js$" | grep -v -E "_test\.|test\." || true)
    
    if [ -n "$code_files" ]; then
        # Check for implementation patterns that suggest minimal approach
        local impl_quality=0
        
        # Check for simple, focused implementations
        local simple_patterns=$(git diff --cached | grep -E "^\+.*return|^\+.*if.*{|^\+.*func " | wc -l)
        if [ "$simple_patterns" -gt 0 ]; then
            echo -e "${GREEN}  ✅ Implementation patterns detected${NC}"
            impl_quality=$((impl_quality + 1))
        fi
        
        # Check for absence of over-engineering patterns
        local complex_patterns=$(git diff --cached | grep -E "^\+.*(interface{}|reflect|unsafe)" | wc -l)
        if [ "$complex_patterns" -eq 0 ]; then
            echo -e "${GREEN}  ✅ No over-engineering detected${NC}"
            impl_quality=$((impl_quality + 1))
        fi
        
        if [ $impl_quality -gt 0 ]; then
            echo -e "${GREEN}Green Phase: VALIDATED${NC}"
            check 0
        else
            echo -e "${YELLOW}Green Phase: UNCERTAIN${NC}"
            check 1
        fi
    else
        echo -e "${BLUE}  ℹ️  No implementation changes detected${NC}"
        check 0
    fi
}

# Check Refactor phase - pattern application
check_refactor_phase() {
    echo "🔄 Validating Refactor Phase (Pattern Application)..."
    
    # Check commit message for refactoring indicators
    local commit_msg=$(git log --format=%B -n 1 HEAD 2>/dev/null || echo "")
    local refactor_indicators=0
    
    if echo "$commit_msg" | grep -qi "refactor\|optimize\|clean\|improve\|pattern"; then
        echo -e "${GREEN}  ✅ Refactoring indicators in commit message${NC}"
        refactor_indicators=$((refactor_indicators + 1))
    fi
    
    # Check for code organization improvements
    local code_changes=$(git diff --cached --numstat | awk '{sum += $1 + $2} END {print sum+0}')
    local file_count=$(git diff --cached --name-only | wc -l)
    
    if [ "${code_changes:-0}" -gt 0 ] && [ "${file_count:-0}" -gt 1 ]; then
        echo -e "${GREEN}  ✅ Multi-file changes suggest refactoring${NC}"
        refactor_indicators=$((refactor_indicators + 1))
    fi
    
    # Check for removal of duplicated code
    local deletions=$(git diff --cached --numstat | awk '{sum += $2} END {print sum+0}')
    if [ "${deletions:-0}" -gt 10 ]; then
        echo -e "${GREEN}  ✅ Significant deletions suggest cleanup${NC}"
        refactor_indicators=$((refactor_indicators + 1))
    fi
    
    if [ $refactor_indicators -gt 0 ]; then
        echo -e "${GREEN}Refactor Phase: VALIDATED${NC}"
        check 0
    else
        echo -e "${BLUE}Refactor Phase: NOT REQUIRED${NC}"
        check 0
    fi
}

# Check Validate phase - coverage and performance
check_validate_phase() {
    echo "✅ Validating Validate Phase (Coverage & Performance)..."
    
    local validation_indicators=0
    
    # Check for benchmark files
    local benchmark_files=$(git diff --cached --name-only | grep -E "bench.*test\.go$|benchmark" || true)
    if [ -n "$benchmark_files" ]; then
        echo -e "${GREEN}  ✅ Benchmark files present${NC}"
        validation_indicators=$((validation_indicators + 1))
    fi
    
    # Check commit message for validation indicators
    local commit_msg=$(git log --format=%B -n 1 HEAD 2>/dev/null || echo "")
    if echo "$commit_msg" | grep -qi "coverage\|benchmark\|performance\|validation\|metrics"; then
        echo -e "${GREEN}  ✅ Validation metrics in commit message${NC}"
        validation_indicators=$((validation_indicators + 1))
    fi
    
    # Check for test coverage improvements
    local test_additions=$(git diff --cached --numstat | grep -E "_test\.go$|\.test\." | awk '{sum += $1} END {print sum+0}')
    if [ "${test_additions:-0}" -gt 5 ]; then
        echo -e "${GREEN}  ✅ Substantial test additions (${test_additions} lines)${NC}"
        validation_indicators=$((validation_indicators + 1))
    fi
    
    if [ $validation_indicators -gt 0 ]; then
        echo -e "${GREEN}Validate Phase: VALIDATED${NC}"
        check 0
    else
        echo -e "${YELLOW}Validate Phase: WEAK (consider adding metrics)${NC}"
        check 1
    fi
}

# Check overall TDD cycle completeness
check_tdd_cycle_completeness() {
    echo "🔄 Checking TDD Cycle Completeness..."
    
    local commit_msg=$(git log --format=%B -n 1 HEAD 2>/dev/null || echo "")
    
    # Check for TDD methodology references
    if echo "$commit_msg" | grep -qi "research.*red.*green\|tdd\|test.*driven"; then
        echo -e "${GREEN}  ✅ TDD methodology referenced${NC}"
        check 0
    else
        echo -e "${YELLOW}  ⚠️  TDD methodology not explicitly referenced${NC}"
        check 1
    fi
    
    # Check for complete cycle indicators
    local cycle_score=0
    if echo "$commit_msg" | grep -qi "research"; then cycle_score=$((cycle_score + 1)); fi
    if echo "$commit_msg" | grep -qi "test\|coverage"; then cycle_score=$((cycle_score + 1)); fi
    if echo "$commit_msg" | grep -qi "implement\|minimal"; then cycle_score=$((cycle_score + 1)); fi
    if echo "$commit_msg" | grep -qi "refactor\|pattern"; then cycle_score=$((cycle_score + 1)); fi
    if echo "$commit_msg" | grep -qi "validate\|performance\|benchmark"; then cycle_score=$((cycle_score + 1)); fi
    
    if [ $cycle_score -ge 3 ]; then
        echo -e "${GREEN}  ✅ Complete TDD cycle indicators (${cycle_score}/5)${NC}"
        check 0
    else
        echo -e "${YELLOW}  ⚠️  Partial TDD cycle indicators (${cycle_score}/5)${NC}"
        check 1
    fi
}

# Main validation
main() {
    echo "🧪 Research-TDD Workflow Validation"
    echo "===================================="
    echo ""
    
    check_research_phase
    echo ""
    check_red_phase
    echo ""
    check_green_phase  
    echo ""
    check_refactor_phase
    echo ""
    check_validate_phase
    echo ""
    check_tdd_cycle_completeness
    
    echo ""
    echo "===================================="
    
    local success_rate=$((validation_score * 100 / total_checks))
    
    if [ $success_rate -ge 80 ]; then
        echo -e "${GREEN}✅ TDD Workflow Validation PASSED${NC}"
        echo -e "${GREEN}   Success Rate: ${success_rate}% (${validation_score}/${total_checks})${NC}"
    elif [ $success_rate -ge 60 ]; then
        echo -e "${YELLOW}⚠️  TDD Workflow Validation WARNING${NC}"
        echo -e "${YELLOW}   Success Rate: ${success_rate}% (${validation_score}/${total_checks})${NC}"
        echo -e "${YELLOW}   Consider improving TDD methodology adherence${NC}"
    else
        echo -e "${RED}❌ TDD Workflow Validation FAILED${NC}"
        echo -e "${RED}   Success Rate: ${success_rate}% (${validation_score}/${total_checks})${NC}"
        echo ""
        echo "Research-TDD Methodology:"
        echo "1. Research: Use Context7, DeepWiki, Exa for behavior specifications"
        echo "2. Red: Write comprehensive tests based on research"
        echo "3. Green: Implement minimal code to pass tests"  
        echo "4. Refactor: Apply patterns from research"
        echo "5. Validate: Verify coverage and performance"
        exit 1
    fi
}

main "$@"