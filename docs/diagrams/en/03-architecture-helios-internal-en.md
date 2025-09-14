# Helios Internal Architecture

Key points:
- L1 in-memory index stores immutable pointers for snapshots/dirs/files; Copy-on-Write occurs only on modification.
- Stats/diff at pointer level; execution triggers L2 to materialize from L0 to disk on demand.

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
flowchart TD
  A[CLI / API] --> B[L1 Root Index];
  B --> C[Ops];
  C --> B1[Snapshot Root];
  C --> B2[Pointer traversal & compare];
  C --> D[L2 Materializer];
  B --> E[L0 CAS blocks BLAKE3];
  D --> F[(Filesystem)];

  subgraph DataFlow[Data read/write]
    E <--> D
  end

  B -.-> NOTE1["Snapshot tree = immutable pointer structure<br/>Copy on write (CoW)"]
```
