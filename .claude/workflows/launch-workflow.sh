#!/bin/bash
# Oppie Thunder Workflow Launcher
# Orchestrates the multi-agent implementation workflow

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Workflow configuration
WORKFLOW_DIR="$(dirname "$0")"
AGENTS_DIR="$WORKFLOW_DIR/../agents"
PROJECT_ROOT="$(dirname "$(dirname "$WORKFLOW_DIR")")"

# Function to print colored output
log() {
    echo -e "${GREEN}[WORKFLOW]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

# Function to check if agent exists
check_agent() {
    local agent=$1
    if [ ! -f "$AGENTS_DIR/$agent.md" ]; then
        error "Agent definition not found: $agent"
    fi
}

# Function to launch agent task
launch_agent() {
    local agent=$1
    local task=$2
    local phase=$3
    
    log "Launching $agent for Phase $phase"
    info "Task: $task"
    
    # Use Claude Code with the Task tool to launch the agent
    claude-code <<EOF
I need to use the Task tool to launch the $agent agent with the following task:
$task

This is for Phase $phase of the Oppie Thunder implementation workflow.
EOF
}

# Function to create TaskMaster task
create_task() {
    local title=$1
    local agent=$2
    local priority=${3:-normal}
    local depends_on=${4:-}
    
    log "Creating TaskMaster task: $title"
    
    if [ -n "$depends_on" ]; then
        task-master create --title "$title" \
            --description "Agent: $agent" \
            --priority "$priority" \
            --depends-on "$depends_on"
    else
        task-master create --title "$title" \
            --description "Agent: $agent" \
            --priority "$priority"
    fi
}

# Main workflow execution
main() {
    local phase=${1:-all}
    
    log "Starting Oppie Thunder Implementation Workflow"
    info "Workflow location: $WORKFLOW_DIR/oppie-thunder-implementation.md"
    
    case $phase in
        1|research)
            log "=== PHASE 1: Research & Architecture Foundation ==="
            check_agent "chief-scientist-deepmind"
            check_agent "alphazero-muzero-planner"
            
            create_task "LATS/MCTS Research Synthesis" \
                "chief-scientist-deepmind" \
                "high"
            
            create_task "MCTS Architecture Review" \
                "alphazero-muzero-planner" \
                "high" \
                "research-synthesis"
            
            launch_agent "chief-scientist-deepmind" \
                "Synthesize LATS and TS-LLM approaches for Oppie Thunder architecture" \
                "1"
            ;;
            
        2|engine)
            log "=== PHASE 2: Core Engine Implementation ==="
            check_agent "alphazero-muzero-planner"
            check_agent "alphaevolve-scientist"
            check_agent "eval-safety-infra-gatekeeper"
            
            # Create parallel tasks
            create_task "LATS Engine Implementation" \
                "alphazero-muzero-planner" \
                "high"
            
            create_task "State Management System" \
                "alphazero-muzero-planner" \
                "high"
            
            create_task "V8 Isolate Sandbox" \
                "eval-safety-infra-gatekeeper" \
                "high"
            
            info "Launching parallel implementation tasks..."
            
            # Launch agents in parallel using background processes
            launch_agent "alphazero-muzero-planner" \
                "Implement LATS engine with tree search and LLM value functions" \
                "2" &
            
            launch_agent "alphazero-muzero-planner" \
                "Build high-performance state management with L0/L1/L2 tiers" \
                "2" &
            
            launch_agent "eval-safety-infra-gatekeeper" \
                "Setup V8 isolate sandbox infrastructure for secure execution" \
                "2" &
            
            wait
            log "Phase 2 parallel tasks launched"
            ;;
            
        3|evolution)
            log "=== PHASE 3: Evolutionary Optimization ==="
            check_agent "alphaevolve-scientist"
            check_agent "alphafold2-structural-scientist"
            
            create_task "Trajectory Replay Buffer" \
                "alphaevolve-scientist" \
                "high"
            
            create_task "DPO Reward Training" \
                "alphaevolve-scientist" \
                "high" \
                "trajectory-buffer"
            
            launch_agent "alphaevolve-scientist" \
                "Implement trajectory replay and population-based training system" \
                "3"
            ;;
            
        4|safety)
            log "=== PHASE 4: Safety & Infrastructure ==="
            check_agent "eval-safety-infra-gatekeeper"
            
            create_task "Safety Mechanisms Implementation" \
                "eval-safety-infra-gatekeeper" \
                "critical"
            
            create_task "Deployment Infrastructure" \
                "eval-safety-infra-gatekeeper" \
                "high"
            
            launch_agent "eval-safety-infra-gatekeeper" \
                "Implement production safety mechanisms and deployment infrastructure" \
                "4"
            ;;
            
        5|advanced)
            log "=== PHASE 5: Advanced Capabilities ==="
            check_agent "alphafold2-structural-scientist"
            
            create_task "Code Structure Analysis" \
                "alphafold2-structural-scientist" \
                "normal"
            
            create_task "Structure-Aware Planning" \
                "alphafold2-structural-scientist" \
                "normal"
            
            launch_agent "alphafold2-structural-scientist" \
                "Implement code structure analysis and pattern recognition" \
                "5"
            ;;
            
        all)
            log "Executing all phases sequentially..."
            $0 1
            $0 2
            $0 3
            $0 4
            $0 5
            log "All phases completed!"
            ;;
            
        status)
            log "Checking workflow status..."
            task-master list --filter "Oppie Thunder"
            ;;
            
        *)
            error "Unknown phase: $phase"
            echo "Usage: $0 [1|2|3|4|5|research|engine|evolution|safety|advanced|all|status]"
            exit 1
            ;;
    esac
    
    log "Workflow phase completed successfully"
}

# Parse command line arguments
if [ $# -eq 0 ]; then
    echo "Oppie Thunder Workflow Launcher"
    echo "================================"
    echo ""
    echo "Usage: $0 [phase]"
    echo ""
    echo "Phases:"
    echo "  1, research   - Research & Architecture Foundation"
    echo "  2, engine     - Core Engine Implementation" 
    echo "  3, evolution  - Evolutionary Optimization"
    echo "  4, safety     - Safety & Infrastructure"
    echo "  5, advanced   - Advanced Capabilities"
    echo "  all           - Execute all phases"
    echo "  status        - Check workflow status"
    echo ""
    echo "Example:"
    echo "  $0 1          # Start Phase 1"
    echo "  $0 all        # Run complete workflow"
    echo "  $0 status     # Check current status"
    exit 0
fi

main "$@"