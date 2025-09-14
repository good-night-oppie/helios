# Helios vs. Git Worktree Architecture

Key points:
- Git centers on on-disk files/index; Helios centers on immutable blocks + in-memory index.
- Branch/merge in Helios are pointer-level operations, enabling machine-speed branching.

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
  subgraph Git_Worktree["Git Worktree (human workflow)"]
    GW_REPO[".git object DB + refs"]
    GW_INDEX["Working tree index (disk)"]
    GW_WT1["Worktree #1 (dirs+files)"]
    GW_WT2["Worktree #N (dirs+files)"]
    GW_REPO --> GW_INDEX
    GW_INDEX --> GW_WT1
    GW_INDEX --> GW_WT2
  end

  subgraph Helios["Helios (machine speed)"]
    H_L0["L0: Content-addressed storage (immutable blocks, BLAKE3)"]
    H_L1["L1: Pure in-memory Root Index (snapshots/dirs/files as pointers)"]
    H_L2["L2: Lazy Materializer (on-demand to disk)"]
    H_CLI["CLI / API"]
    H_AGENT["AI Agent / Orchestrator"]
    H_AGENT --> H_CLI --> H_L1
    H_L1 --> H_L0
    H_CLI -- materialize --> H_L2 -- I/O --> FS[(Filesystem)]
  end

  style GW_REPO fill:#e9f0fb,stroke:#5b7bd5
  style GW_INDEX fill:#eef2fb,stroke:#5b7bd5
  style GW_WT1 fill:#f7f9fe,stroke:#5b7bd5
  style GW_WT2 fill:#f7f9fe,stroke:#5b7bd5
  style H_L0 fill:#e6fff6,stroke:#16a085
  style H_L1 fill:#f0fff0,stroke:#27ae60
  style H_L2 fill:#fff7e6,stroke:#f39c12
  style H_AGENT stroke:#7f8c8d,stroke-dasharray: 4 2
```
