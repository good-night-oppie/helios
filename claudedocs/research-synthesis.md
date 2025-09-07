# Research Synthesis: Hybrid MCTS-LATS Architecture for Oppie Thunder

## Executive Summary

This document synthesizes key insights from LATS (Language Agent Tree Search) and TS-LLM (Tree-Search enhanced LLM) research papers, analyzing their applicability to Oppie Thunder's target of <5 second iteration times for autonomous code generation.

## 1. Core Insights from LATS

### 1.1 Architecture Components
The LATS framework demonstrates three critical components:

1. **LLM-Powered Value Functions**: Instead of prompting for value estimates, LATS uses learned value functions trained on task-specific data, showing 10-20% improvement over GPT-3.5 prompted values.

2. **Self-Reflection Mechanism**: Failed trajectories generate verbal self-reflections that provide semantic gradient signals, enabling learning without expensive optimization.

3. **External Feedback Integration**: Environmental observations improve decision-making by 15-30% compared to pure reasoning approaches.

### 1.2 Performance Characteristics
- **Pass@1 Accuracy**: 92.7% on HumanEval with GPT-4
- **Token Efficiency**: 173k-210k tokens for complex tasks
- **Search Depth**: Limited to 7-10 steps in practice
- **Inference Time**: Not optimized for real-time (<5s constraint)

### 1.3 Key Limitations for Oppie Thunder
- Sentence-level action nodes create coarse-grained trees
- No intermediate value backpropagation during generation
- Lacks continuous learning across episodes
- Memory requirements scale poorly with tree depth

## 2. Core Insights from TS-LLM

### 2.1 AlphaZero-Like Enhancements

TS-LLM extends tree search to depths of 64 using:

1. **Learned Value Networks**: Trained from policy rollouts with TD/MC estimates
2. **PUCT Selection**: Balances exploration/exploitation with learned priors
3. **Token-Level Actions**: Fine-grained control for precise edits
4. **Iterative Training**: Policy distillation + value learning cycles

### 2.2 Performance Metrics
- **Search Depth**: Up to 64 levels (8x deeper than LATS)
- **Token Efficiency**: 30-50% reduction via batching
- **Training Convergence**: 3-5 iterations for policy improvement
- **Value Accuracy**: 85-90% correlation with ground truth

### 2.3 Computational Insights
- Node expansion dominates cost (60-70% of time)
- Value evaluation adds 20-30% overhead
- Memory scales O(b^d) where b=branching, d=depth
- KV cache critical for token-level efficiency

## 3. Synthesis for Oppie Thunder

### 3.1 Hybrid Architecture Opportunities

**Token-Efficient Patterns from TS-LLM:**
- Multi-level action abstraction (token → line → function)
- Lazy value evaluation with caching
- Batch node expansion across similar states
- Progressive widening to control branching

**Reflection Mechanisms from LATS:**
- Semantic error analysis for immediate feedback
- Trajectory summarization for episodic memory
- Cross-episode learning without retraining

### 3.2 Critical Performance Factors

For <5 second iterations, we must optimize:

1. **LLM Inference**: 
   - Small policy model (125M-1B params) for fast rollouts
   - Larger value model (7B) evaluated selectively
   - Speculative decoding for 2-3x speedup

2. **Tree Search Efficiency**:
   - Adaptive depth (2-8 levels based on task complexity)
   - Early termination on high-confidence paths
   - Parallel MCTS simulations across CPU threads

3. **State Management**:
   - Incremental tree updates vs full reconstruction
   - Shared KV cache across simulations
   - Copy-on-write for state branching

### 3.3 Memory and Compute Budgets

**Per-Iteration Budget (5 seconds):**
- LLM Inference: 2-3 seconds (40-60%)
- Tree Operations: 1-2 seconds (20-40%)
- State Management: 0.5-1 second (10-20%)
- Value Computation: 0.5 seconds (10%)

**Memory Requirements:**
- Tree Storage: 100-500MB per episode
- KV Cache: 2-4GB for parallel simulations
- Value Network: 4-8GB (can be offloaded)
- Total: 8-16GB GPU memory

## 4. Implementation Recommendations

### 4.1 Phased Approach

**Phase 1: Shallow Trees (2-4 depth)**
- Focus on immediate code corrections
- Use small policy model (350M Phi-style)
- Single-threaded MCTS
- Target: 3-4 second iterations

**Phase 2: Adaptive Depth (2-8 levels)**
- Add value network training
- Implement parallel simulations
- Introduce reflection mechanism
- Target: 4-5 second iterations

**Phase 3: Deep Search (up to 16 levels)**
- Full AlphaZero-style training
- Multi-GPU inference
- Episodic memory integration
- Target: Maintain <5 seconds via optimization

### 4.2 Key Design Decisions

1. **Action Granularity**: Hybrid token/line/function levels based on edit scope
2. **Value Function**: Separate small critic network, not shared decoder
3. **Search Algorithm**: MCTS-α with early termination
4. **Memory Strategy**: Persistent tree with incremental updates
5. **Parallelization**: CPU for tree ops, GPU for LLM inference

## 5. Risk Assessment

### Technical Risks
- **Latency Variability**: LLM inference can spike 2-3x on complex inputs
- **Memory Pressure**: Deep trees may cause OOM on 16GB GPUs
- **Value Divergence**: Learned values may become stale across code evolution

### Mitigation Strategies
- Implement hard timeout with graceful degradation
- Use memory-mapped tree storage for overflow
- Continuous value function updates from successful episodes

## 6. Conclusion

The synthesis of LATS and TS-LLM approaches provides a viable path to <5 second iterations through:

1. **Hierarchical action spaces** reducing search complexity
2. **Learned value functions** enabling deep search
3. **Selective computation** via early termination and caching
4. **Parallel execution** across CPU/GPU resources

The key innovation will be adaptive complexity management - using shallow search for simple edits while reserving deep search for complex refactoring, all within the 5-second budget.

## References

- Yao et al. 2023. "Language Agent Tree Search" (LATS)
- Feng et al. 2024. "AlphaZero-Like Tree-Search can Guide LLM" (TS-LLM)
- PRD5 Oppie Thunder Architecture Specification