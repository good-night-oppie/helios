# Testing Structure

This directory contains organized tests for the Helios Engine project, categorized by test type for better maintainability and consistency.

## Directory Structure

```
tests/
├── unit/          # Unit tests for individual components
├── integration/   # Integration tests for component interactions
├── benchmark/     # Performance benchmarks
├── fuzz/          # Fuzz tests for robustness
└── stress/        # Stress tests for load validation
```

## Test Categories

### Unit Tests (`unit/`)
- Individual component testing
- Isolated functionality validation
- Fast execution for CI/CD

### Integration Tests (`integration/`)
- Cross-component interaction testing
- End-to-end workflow validation
- Database and external service integration

### Benchmark Tests (`benchmark/`)
- Performance measurement and validation
- Critical path timing verification
- Memory allocation analysis
- **Key Target**: VST commit operations <70μs

### Fuzz Tests (`fuzz/`)
- Randomized input testing
- Edge case discovery
- Robustness validation

### Stress Tests (`stress/`)
- High-load scenario testing
- Resource exhaustion testing
- Production simulation

## Running Tests

### All Tests
```bash
go test ./tests/...
```

### By Category
```bash
# Unit tests
go test ./tests/unit/...

# Integration tests  
go test ./tests/integration/...

# Benchmarks
go test -bench=. ./tests/benchmark/...

# Fuzz tests
go test -fuzz=. ./tests/fuzz/...

# Stress tests
go test ./tests/stress/...
```

### Performance Validation
```bash
# Run critical performance benchmarks
go test -bench=BenchmarkCommitPerformance ./tests/benchmark/ -benchmem

# Validate <70μs target
go test -bench=BenchmarkCommitPerformance/Optimized_100Files_1KB ./tests/benchmark/
```

## Test Organization Principles

1. **Separation of Concerns**: Each test type serves a specific purpose
2. **Consistent Naming**: Clear, descriptive test function names
3. **Performance Targets**: Benchmarks include explicit performance goals
4. **Documentation**: Each test category documents its purpose and execution
5. **CI/CD Integration**: Tests organized for efficient pipeline execution

## Performance Targets

### VST (Versioned State Tree)
- **Commit Operations**: <70μs (achieved: ~25μs)
- **Memory Efficiency**: <10KB allocations per commit
- **I/O Reduction**: 99% reduction vs naive implementation

### L1 Cache
- **Hit Latency**: <1μs
- **Miss Latency**: <10μs

### L2 Storage (RocksDB)
- **Batch Write**: <5ms
- **Point Read**: <1ms

## Contributing

When adding new tests:

1. Choose the appropriate category directory
2. Follow existing naming conventions
3. Include performance targets in benchmarks
4. Add documentation for complex test scenarios
5. Ensure tests can run independently

## Cleanup Benefits

This reorganization provides:
- **Clarity**: Tests categorized by purpose
- **Maintainability**: Easier to locate and update tests
- **Performance**: Consolidated benchmarks reduce redundancy
- **Consistency**: Uniform structure across the project
- **CI/CD Efficiency**: Run specific test categories as needed