# Git Worktree Structure (Supplement)

Key points:
- Multiple worktrees are coupled with refs, on-disk index and filesystem.
- High churn for creating/cleaning trees; not suitable for thousands of short-lived branches per minute.

```mermaid
%%{init: { 'theme': 'base', 'themeVariables': {
  'primaryColor': '#FF6A00',
  'primaryTextColor': '#0A0E14',
  'primaryBorderColor': '#0D1B2A',
  'lineColor': '#0D1B2A',
  'secondaryColor': '#FFB08A',
  'tertiaryColor': '#8A60FF',
  'background': '#F6F4EE'
}} }%%
flowchart LR
  GRepo[".git object DB + refs"] --> GIndex["Index/HEAD"]
  GIndex --> WT1["worktree-1"]
  GIndex --> WTN["worktree-N"]
  WT1 --- FS1[(filesystem dir/files)]
  WTN --- FSN[(filesystem dir/files)]
  WTN -.-> NOTE1["Creating/removing worktrees touches dirs/index/cleanup<br/>Unsuitable for high-frequency short-lived branches"]
```
