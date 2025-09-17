# Performance Metrics Tracking Strategy

## Overview
This document outlines the approach for tracking performance metrics over time to detect gradual regressions in the Helios engine.

## Current Performance Thresholds

### Time Travel (MCTS)
- **Threshold**: 5µs average per state restoration
- **Local Target**: <1µs on modern hardware
- **CI Tolerance**: 5µs to account for virtualized environments

### VST Integration (CAS)
- **Threshold**: 1ms for 100-file batch operations
- **Per-file Target**: ~10µs per operation
- **Local Performance**: 200-400µs typical

## Recommended Tracking Approach

### 1. Structured Logging
Add performance metrics to test output in machine-readable format:
```
PERF_METRIC|test_name|duration_us|timestamp|environment
```

### 2. Metrics Collection
Options for collecting and analyzing metrics:
- **GitHub Actions Artifacts**: Store test results as artifacts for historical analysis
- **Time-series Database**: InfluxDB or Prometheus for long-term tracking
- **CSV Export**: Simple format for spreadsheet analysis

### 3. Key Metrics to Track
- P50, P95, P99 percentiles
- Min/max values
- Standard deviation
- Environment tags (CI vs local, OS, hardware specs)

### 4. Regression Detection
- Set up alerts when P95 exceeds threshold by >20%
- Track week-over-week performance trends
- Compare CI vs local development baselines

### 5. Implementation Steps
1. Add benchmark tags to performance-critical tests
2. Export metrics in structured format
3. Create dashboard for visualization
4. Set up automated regression alerts
5. Review metrics in PR checks

## Environment-Specific Thresholds
Future enhancement: Detect execution environment and apply appropriate thresholds:
- CI environments: Relaxed thresholds (current values)
- Local development: Stricter thresholds for early detection
- Production: Separate monitoring with SLA-based alerts

## Example Implementation
```go
func recordPerformance(testName string, duration time.Duration) {
    if os.Getenv("PERF_TRACKING") == "true" {
        fmt.Printf("PERF_METRIC|%s|%d|%d|%s\n",
            testName,
            duration.Microseconds(),
            time.Now().Unix(),
            runtime.GOOS)
    }
}
```