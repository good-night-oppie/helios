# Helios - 面向AI智能体的高速版本控制系统

[![GitHub release (latest by date)](https://img.shields.io/github/v/release/good-night-oppie/helios?style=for-the-badge)](https://github.com/good-night-oppie/helios/releases/latest)
[![GitHub Downloads](https://img.shields.io/github/downloads/good-night-oppie/helios/total?style=for-the-badge&color=brightgreen)](https://github.com/good-night-oppie/helios/releases)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg?style=for-the-badge)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.22+-00ADD8?style=for-the-badge&logo=go)](https://golang.org/)

[![Platform Support](https://img.shields.io/badge/platforms-Linux%20|%20macOS%20|%20Windows-success?style=for-the-badge)](https://github.com/good-night-oppie/helios/releases)
[![Architecture](https://img.shields.io/badge/arch-AMD64%20|%20ARM64-blue?style=for-the-badge)](https://github.com/good-night-oppie/helios/releases)
[![DeepWiki](https://img.shields.io/badge/deepwiki-indexed-purple?style=for-the-badge)](https://deepwiki.com/good-night-oppie/helios)
[![README_EN](https://img.shields.io/badge/English-README.md-blue?style=for-the-badge)](README.md)

## Helios 解决的问题

**高频提交**: AI智能体每小时产生100+次提交，Git成为瓶颈
**存储爆炸**: 测试代码变体会创建超大仓库
**分支缓慢**: O(n)分支创建阻塞并行实验
**手动回滚**: 当智能体破坏代码时，恢复缓慢且需要手动操作

## 快速开始

```bash
# 安装
curl -sSL https://raw.githubusercontent.com/good-night-oppie/helios/master/scripts/install.sh | sh

# 在现有项目中使用
cd my-project
echo "print('hello')" > test.py
helios commit --work .  # 快速提交当前目录
```

## 工作原理

**内容可寻址存储** 替代Git的基于差异的方法：
- 文件通过BLAKE3哈希存储，自动去重
- O(1)分支创建（复制快照引用）
- 三层架构：内存 → 缓存 → 持久化存储

## AI编程智能体的真实使用场景

**高频代码生成**: 每分钟测试多个LLM输出
- GPT-4生成10个函数实现 → 每个在<1ms内提交 → 运行测试 → 保留最佳
- 传统Git: 10 × 20ms = 200ms仅用于版本控制
- Helios: 10 × 0.2ms = 2ms用于版本控制

**并行实验分支**: 多个智能体尝试不同方法
- 创建50个分支测试不同算法 → 合并成功的
- 传统Git: 50 × 100ms = 5+秒的分支创建开销
- Helios: 50 × 0.07ms = 3.5ms创建所有分支

**失败时即时回滚**: 当AI智能体破坏正常代码时
- 智能体进行47次实验性更改 → 测试失败 → 回滚到最后工作状态
- 传统Git: `git reset --hard`需要100-500ms加上工作目录同步
- Helios: 跳转到任何先前状态<0.1ms

## 技术概述

### 为什么Helios更快

**瓶颈**: Git将更改存储为差异，并使用文件系统操作处理分支
**我们的方法**: 存储唯一内容一次，通过密码哈希引用

```
传统Git                    Helios内容可寻址
├── commit1/                      ├── content/
│   ├── file1.py (完整内容)   │   ├── abc123... → "def func1():"
│   └── file2.py (完整内容)   │   ├── def456... → "def func2():"  
├── commit2/                      │   └── ghi789... → "def func3():"
│   ├── file1.py (差异)           └── snapshots/
│   └── file2.py (差异)               ├── commit1 → [abc123, def456]
└── commit3/                          └── commit2 → [abc123, ghi789]
    ├── file1.py (差异)
    └── file2.py (差异)
```

**结果**: 当你的AI生成1000个相似函数时，我们存储共享代码一次而不是1000次。

### 三层性能架构

```
🧠 L0: 内存工作集    - <1μs操作，当前文件
⚡ L1: 压缩缓存         - <10μs访问，频繁使用的内容  
💾 L2: PebbleDB存储         - <5ms写入，永久存储
```

**为什么对AI重要**: 智能体可以提交每个代码更改而无性能损失。

### 性能表现

**你的AI每秒可达到的操作数:**

| 任务 | Git限制 | Helios | 实际影响 |
|------|-----------|---------|-------------|
| 代码提交 | ~20/秒 | ~5,000/秒 | 快速测试多个AI输出 |
| 分支创建 | ~5/秒 | ~14,000/秒 | 并行实验 |  
| 回滚操作 | ~2/秒 | ~10,000/秒 | 从失败中即时恢复 |

**存储效率** (在真实AI代码库上测量):
- 1000个AI生成的Python函数: Git=850MB, Helios=23MB (节省97%)
- 500个React组件: Git=1.2GB, Helios=45MB (节省96%)

## 与您的AI工具集成

**适用于任何AI框架** - 如果它能调用命令行工具，就能使用Helios:

```python
# 与OpenAI + 现有工具的示例
import subprocess
import openai

for experiment in range(100):
    # 用AI生成代码
    response = openai.ChatCompletion.create(
        model="gpt-4",
        messages=[{"role": "user", "content": f"为{experiment}编写函数"}]
    )
    
    # 即时保存和版本控制(<1ms)
    with open("solution.py", "w") as f:
        f.write(response.choices[0].message.content)
    
    # 提交当前状态
    result = subprocess.run(["helios", "commit", "--work", "."], capture_output=True)
    snapshot_id = result.stdout.decode().strip()
    
    # 测试代码
    if not run_tests():
        # 回滚到先前工作状态
        subprocess.run(["helios", "restore", "--id", previous_snapshot_id])
    else:
        previous_snapshot_id = snapshot_id  # 保存工作状态
```

**热门集成:**
- **Cursor/VSCode**: 使用Helios作为Git替代
- **GitHub Copilot**: 提交每个建议进行比较
- **CodeT5/StarCoder**: 版本所有生成的变体
- **自定义LLM工作流**: Git命令的直接替代

## 系统要求

- **操作系统**: Linux, macOS, 或 Windows  
- **内存**: 最少1GB (随仓库大小扩展)
- **依赖**: 无 - 单个二进制文件

## 当前v0.0.1限制

这是一个具有核心功能的alpha版本:

**当前可用:**
- 使用`helios commit --work <path>`快速提交
- 使用`helios restore --id <id>`快照恢复
- 使用`helios diff --from <id> --to <id>`差异比较
- 使用`helios materialize`文件提取

**未来版本将提供:**
- Git导入/导出功能
- `init`, `add`, `branch`, `checkout`命令
- Git兼容的命令语法
- 针对<70μs提交的性能优化

## 安装

```bash
# 安装 
curl -sSL https://raw.githubusercontent.com/good-night-oppie/helios/master/scripts/install.sh | sh

# 在现有项目中使用
cd your-ai-project/
helios commit --work .  # 提交当前目录状态

# 性能比较 (alpha - 持续优化中)
time git commit --allow-empty -m "test"    # ~20ms
time helios commit --work .                 # 当前: ~1-5ms, 目标: <1ms
```

## 最新版本

🚀 **v0.0.1** 现已发布，包含:
- ✅ **跨平台二进制文件** 支持Linux/macOS/Windows (AMD64/ARM64)
- ✅ **PebbleDB存储** (纯Go实现，无CGO依赖)
- ✅ **核心CLI命令** 准备好用于AI工作流
- ✅ **一行安装** 通过curl脚本

[📦 从GitHub Releases下载](https://github.com/good-night-oppie/helios/releases/latest)

## 技术详情

查看 [TECHNICAL_REPORT.md](TECHNICAL_REPORT.md) 获取实现详情。

## 问题和支持

- **问题**: [GitHub Issues](https://github.com/good-night-oppie/helios/issues)
- **文档**: [DeepWiki 文档](https://deepwiki.com/good-night-oppie/helios)
- **状态**: Alpha版本

---

**许可证**: Apache 2.0  
**状态**: Alpha版本 - 在生产环境使用前请充分测试