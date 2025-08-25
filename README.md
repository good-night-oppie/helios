# Helios Engine

Helios is a user-space virtual state tree (VST) engine for fast snapshots and content management.

## Security Boundaries

**Helios operates strictly in user-space only.** The following mechanisms are strictly forbidden:

- `overlayfs`
- `mount` / `unmount` operations
- `cgroups`
- `namespaces` (CLONE_NEW*)
- `CAP_SYS_ADMIN` (or any other privileged capability)
- Any kernel-level privileged operations

This constraint is enforced by automated policy guard tests that scan the codebase for forbidden syscalls.

## Architecture

- **VST (Virtual State Tree)**: In-memory working set with content-addressed snapshots
- **L1 Cache**: LRU cache with compression for hot data
- **L2 Store**: RocksDB-based persistent object storage
- **CLI**: Command-line interface for commit, restore, diff, and materialize operations

## Metrics

The engine exposes performance metrics via the `stats` command:

### L1 Cache Metrics
- `hits`: Cache hit count
- `misses`: Cache miss count
- `evictions`: Number of evicted items
- `size`: Current cache size in bytes
- `items`: Number of cached items

### Engine Metrics
- `commit_latency_us_p50`: 50th percentile commit latency (microseconds)
- `commit_latency_us_p95`: 95th percentile commit latency (microseconds)
- `commit_latency_us_p99`: 99th percentile commit latency (microseconds)
- `new_objects`: Total number of new objects committed
- `new_bytes`: Total bytes of new data committed

## Usage

```bash
# Initialize and commit files
helios-cli commit

# Show performance statistics
helios-cli stats

# Restore to a specific snapshot
helios-cli restore <snapshot-id>

# Show differences between snapshots
helios-cli diff <from-id> <to-id>

# Export snapshot to filesystem
helios-cli materialize <snapshot-id> <output-dir>
```

## Development

```bash
# Run all tests with race detection
make test

# Check test coverage (requires ≥85%)
make cover

# Run fuzz tests
go test -fuzz=FuzzPathRoundTrip -fuzztime=30s ./pkg/helios/vst/
```

## Testing

The codebase includes comprehensive testing:

- **Unit tests**: Individual component testing
- **Integration tests**: L1/L2 storage integration
- **Fuzz tests**: Path handling and selector robustness
- **Robustness tests**: L2 store failure scenarios
- **Race detection**: Concurrent access safety
- **Policy guard tests**: Enforces user-space only constraint

Test coverage is maintained at ≥85% across all packages.
