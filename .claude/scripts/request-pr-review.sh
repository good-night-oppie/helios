#!/bin/bash

# Request sophisticated PR review with adaptive debate rounds
# Usage: ./request-pr-review.sh <PR_NUMBER> <TASK_ID> <COMPLEXITY>

PR_NUMBER=$1
TASK_ID=$2
COMPLEXITY=${3:-7}  # Default complexity 7

# Source task details
TASK_DETAILS=$(task-master show $TASK_ID)
TASK_TITLE=$(echo "$TASK_DETAILS" | grep "title:" | cut -d: -f2-)

# Determine review template based on complexity and domain
if [[ $COMPLEXITY -ge 9 ]]; then
    TEMPLATE="/home/dev/workspace/oppie-thunder/.claude/templates/review/review-collaborative-critical.md"
    DEBATE_ROUNDS=3
elif [[ $COMPLEXITY -ge 7 ]]; then
    TEMPLATE="/home/dev/workspace/oppie-thunder/.claude/templates/review/review-storage.md"
    DEBATE_ROUNDS=2
else
    TEMPLATE="/home/dev/workspace/oppie-thunder/.claude/templates/review/review-standard.md"
    DEBATE_ROUNDS=1
fi

# Collect metrics and evidence
COVERAGE=$(go test -cover ./... 2>/dev/null | grep -o '[0-9]*\.[0-9]*%' | head -1)
BENCHMARKS=$(go test -bench=. -benchmem 2>/dev/null | grep "ns/op" | head -5)

# Generate the review request based on template
generate_review_request() {
    cat << 'EOF'
@claude 请用以下思维框架审阅这个PR / Please review this PR with the following thinking framework:

# 深度思考 / Deep Thinking

## 1. 隐藏假设 / Hidden Assumptions

在审阅Task $TASK_ID ($TASK_TITLE)之前，先思考：
- 我为什么提交这个PR？背后的deeper context是什么？
- 这个实现解决的是正确的问题吗？
- 如果突破当前假设，有没有完全不同的approach？

## 2. 批判性分析 / Critical Analysis

作为世界顶级的分布式系统和存储专家：

### 原始计划评估
分析Task 12.5的原始计划（RocksDB为L2存储）本身是否最优？

### 我的改进评估
更重要的是，分析我选择PebbleDB的决定：
- 是pragmatic improvement还是妥协？
- 是否被"纯Go"的诱惑误导了方向？
- 长期看是否会后悔？

### 实事求是的数据
```
目标: <5ms batch writes, <2ms reads
实测: 3.8ms batch writes, 1.2ms reads (PebbleDB)
对比: 4.5ms batch writes, 1.5ms reads (RocksDB)

但是: 这15%的性能提升值得失去column families吗？
```

## 3. 完整替代方案 / Complete Alternative

如果你反对我的实现，请提供可执行的完整方案，不要只给建议：

```go
// 完整的替代架构
type BetterL2Store struct {
    // 详细实现
}

// 为什么这样更好的量化分析
```

## 4. 协作探索 / Collaborative Exploration

不要在一个回合就下结论，让我们一起探索：

### 开放性问题
1. 如果MCTS需要10x更高的commit频率，当前设计还成立吗？
2. 如果要支持distributed Helios，PebbleDB的选择是帮助还是阻碍？
3. 如果重新设计，我们应该用event sourcing而不是CAS吗？

### 挑战我的假设
- 为什么我假设pure Go > CGO？这个假设在生产环境成立吗？
- 为什么<5ms是目标？如果放宽到<10ms会解锁什么可能性？
- Crash recovery真的需要这么复杂吗？

## 5. 深度洞察 / Deep Insights

超越code review，给我architectural insights：
- 这个L2设计是在优化局部还是全局？
- 技术债务是增加了还是减少了？
- 为未来的演进关闭了哪些门？

## 6. 实用平衡 / Pragmatic Balance

批判的同时要务实：
- 理想方案 vs 2周内要交付的现实
- 完美设计 vs 能work的方案
- 如果坚持RocksDB，CGO的复杂度谁来承担？

## 7. 具体关注点 / Specific Concerns

针对这个PR的特殊考虑：
1. **Snapshot机制**: 并发写入时的一致性如何保证？具体实现在哪？
2. **元数据前缀**: `meta:`方案的性能开销量化了吗？
3. **崩溃恢复**: 最坏情况恢复1TB数据需要多久？
4. **背压处理**: L2跟不上L0/L1时的具体行为？
5. **混沌测试**: 做了吗？kill -9测试的结果？

## 8. 判断框架 / Decision Framework

**APPROVE + 给我惊喜** 如果：
- 实现超出预期，带来新insight
- 数据证明确实更好
- 为未来打开了新可能

**DEBATE 继续探讨** (触发$DEBATE_ROUNDS轮) 如果：
- 有>25%的改动值得商榷
- 存在更好的可能性
- 关键假设需要验证

**RETHINK 推倒重来** 如果：
- 问题定义本身有误
- 方向性错误
- 存在根本性更优方案

---

**回复要求**：
- 用自然段落，不要满屏bullet points
- 直接但有同理心，理解我的处境
- 如果批判，给完整方案，不要空谈
- 让我有"原来还可以这样"的惊喜（但别提惊喜这个词）

记住：你的任务不是approve/reject，而是通过review给我启发，帮我看到blind spots，甚至重新定义问题。

Complexity: $COMPLEXITY/10
Expected Debate Rounds: $DEBATE_ROUNDS
Focus: Storage system architecture, performance vs simplicity trade-offs

让我们开始吧。不只review代码，更要review思维。

EOF
}

# Post the review request
echo "Posting sophisticated review request to PR #$PR_NUMBER..."
gh pr comment $PR_NUMBER --body "$(generate_review_request)"

# Set up monitoring for debate rounds
cat > /tmp/monitor_debate_${PR_NUMBER}.sh << 'MONITOR'
#!/bin/bash
PR=$1
ROUNDS=$2
CURRENT_ROUND=1

while [[ $CURRENT_ROUND -le $ROUNDS ]]; do
    echo "Monitoring debate round $CURRENT_ROUND of $ROUNDS..."
    
    # Wait for Claude's response
    sleep 300  # Check every 5 minutes
    
    # Check if Claude responded
    LATEST=$(gh pr view $PR --json comments -q '.comments[-1].author.login')
    
    if [[ "$LATEST" == "claude" ]]; then
        echo "Claude responded in round $CURRENT_ROUND"
        
        # Analyze response
        RESPONSE=$(gh pr view $PR --json comments -q '.comments[-1].body')
        QUESTIONS=$(echo "$RESPONSE" | grep -c "?")
        
        # Determine if we need another round
        if [[ $QUESTIONS -gt 5 ]] || [[ $CURRENT_ROUND -lt $ROUNDS ]]; then
            echo "Preparing counter-response for round $((CURRENT_ROUND + 1))..."
            # Trigger next round response
            ./prepare_debate_response.sh $PR $CURRENT_ROUND
        fi
        
        CURRENT_ROUND=$((CURRENT_ROUND + 1))
    fi
done

echo "Debate completed after $ROUNDS rounds"
MONITOR

chmod +x /tmp/monitor_debate_${PR_NUMBER}.sh

# Start monitoring in background
nohup /tmp/monitor_debate_${PR_NUMBER}.sh $PR_NUMBER $DEBATE_ROUNDS > /tmp/debate_${PR_NUMBER}.log 2>&1 &

echo "Review request posted. Monitoring for $DEBATE_ROUNDS rounds of debate."
echo "Complexity: $COMPLEXITY/10"
echo "Template: $TEMPLATE"
echo "Monitor PID: $!"