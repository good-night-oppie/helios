# CoW Write Amplification vs Block Size (4KB vs 16KB)

Key points:
- Smaller blocks increase dedup granularity and reduce write amplification, but raise metadata/lookup overhead.
- Choose block size by balancing locality of edits vs metadata cost.

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
  subgraph Small[Block size: 4KB]
    S1[Higher dedup\nLower write amplification]:::good --> S2[More ChunkRefs\nLarger indexes]:::warn
  end
  subgraph Large[Block size: 16KB]
    L1[Lower dedup\nHigher write amp relative]:::warn --> L2[Fewer ChunkRefs\nSmaller indexes]:::good
  end
  S1 -. Good for: frequent small edits .-> L1
  L2 -. Good for: large sequential appends .-> S2

  classDef good fill:#e6fff6,stroke:#16a085;
  classDef warn fill:#fff7e6,stroke:#f39c12;
```
