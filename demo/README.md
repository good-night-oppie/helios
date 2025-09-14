# Helios Demo - Parallel Universe Engine

> Transform AI verification from 83 hours to 5 minutes through microsecond-latency state management

## 🚀 Quick Start (24-Hour Ready)

### Option 1: Docker (Recommended)
```bash
# Clone and run immediately
git clone https://github.com/your-org/helios-demo
cd helios-demo
./scripts/run.sh
```

Visit http://localhost:3000 to experience the demo!

### Option 2: Docker Compose
```bash
cd docker
docker-compose up
```

### Option 3: Development Mode
```bash
./scripts/dev.sh
```

## 🎯 Demo Experience

### What You'll See:
1. **Interactive Universe Creation**: Create thousands of parallel universes in real-time
2. **Memory Efficiency Visualization**: Watch COW reduce 50TB to 50GB
3. **Performance Comparison**: See 83-hour serial vs 5-minute parallel execution
4. **Live Statistics**: Monitor VST commits, memory usage, and system health
5. **Multiple Visualizations**: Grid, flow, and tree views of parallel universes

### Key Features:
- **Real-time Universe Management**: Create, monitor, and clean up parallel states
- **Performance Benchmarking**: Live comparison with traditional approaches
- **Memory Analytics**: COW efficiency demonstration with actual metrics
- **Industry Scenarios**: EDA, quantitative finance, and drug discovery examples

## 🏗️ Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   React Frontend│◄───┤ Socket.IO/HTTP  │◄───┤  Node.js Backend│
│   (Port 3000)   │    │   Real-time     │    │   (Port 3001)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                                              │
         ▼                                              ▼
┌─────────────────┐                          ┌─────────────────┐
│ Visualization   │                          │ Helios Engine   │
│ Components:     │                          │ Simulation:     │
│ • Universe Grid │                          │ • VST API       │
│ • Stats Panel   │                          │ • COW Memory    │
│ • Controls      │                          │ • GC System     │
│ • Timing Chart  │                          │ • Benchmarking  │
└─────────────────┘                          └─────────────────┘
```

## 🛠️ Setup Requirements

### System Requirements:
- **Docker**: Latest version (recommended approach)
- **Node.js**: 18+ (for development)
- **Memory**: 2GB+ available RAM
- **Ports**: 3000 and 3001 must be available

### Browser Compatibility:
- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+

## 📊 Performance Demonstrations

### Core Metrics Showcased:
- **VST Commit Latency**: <70μs (P95) - visualized in real-time
- **Memory Efficiency**: 1000 universes = ~100GB vs 50TB traditional
- **State Switching**: Microsecond-level universe transitions
- **Parallel Execution**: 8x speedup with license-aware scheduling

### Comparison Scenarios:
1. **EDA Chip Design**: 1000 timing closure tests
2. **Risk Analysis**: 5000 market scenario simulations
3. **Drug Discovery**: 100 molecular compound tests

## 🎮 Interactive Controls

### Universe Management:
- **Create**: Spawn 10, 100, 500, 1K, or 5K parallel universes
- **Visualize**: Switch between grid, flow, and tree views
- **Monitor**: Real-time statistics and memory usage
- **Cleanup**: Manual and automatic garbage collection

### Performance Testing:
- **Serial Simulation**: Calculate traditional execution time
- **Parallel Comparison**: Show Helios efficiency gains
- **Cost Analysis**: Licensing and infrastructure savings
- **Real-world Examples**: Industry-specific use cases

## 🔧 Development

### Local Development:
```bash
# Start both frontend and backend
./scripts/dev.sh

# Or manually:
cd backend && npm install && npm start &
cd frontend && npm install && npm start
```

### Building for Production:
```bash
./scripts/build.sh
```

### Project Structure:
```
helios-demo/
├── backend/          # Node.js + Socket.IO server
│   ├── server.js     # Helios engine simulation
│   └── package.json
├── frontend/         # React application
│   ├── src/
│   │   ├── components/  # UI components
│   │   ├── App.js       # Main application
│   │   └── App.css      # Styles
│   └── package.json
├── docker/           # Containerization
│   ├── Dockerfile
│   └── docker-compose.yml
└── scripts/          # Automation scripts
    ├── build.sh      # Build Docker image
    ├── run.sh        # Production deployment
    └── dev.sh        # Development mode
```

## 🌐 API Endpoints

### Core API:
- `GET /api/stats` - Engine statistics
- `GET /api/universes` - List parallel universes
- `POST /api/universes` - Create new universes
- `POST /api/universes/:id/commit` - Commit changes to universe
- `DELETE /api/universes/:id` - Rollback universe
- `POST /api/simulate/:type` - Performance simulation

### WebSocket Events:
- `stats_update` - Real-time statistics
- `universes_created` - New universe notifications
- `creation_progress` - Batch creation progress
- `universe_rollback` - Rollback events
- `gc_complete` - Garbage collection results

## 🎯 Demo Script (5-Minute Presentation)

### Minute 1: Problem Introduction
> "AI can think in parallel, but verification is stuck in serial execution"

### Minute 2: Traditional Approach Demo
> Show 1000-test scenario: 83 hours, massive resource requirements

### Minute 3: Helios Solution
> Create 1000 parallel universes in seconds, show memory efficiency

### Minute 4: Performance Comparison
> Live metrics: <70μs commits, 8x speedup, 95% cost savings

### Minute 5: Real-world Impact
> EDA, finance, and biotech use cases with actual time/cost calculations

## 🚨 Troubleshooting

### Common Issues:

**Port Already in Use:**
```bash
# Check what's using the ports
lsof -i :3000
lsof -i :3001

# Stop existing processes
docker stop helios-demo
pkill -f "node.*3000"
```

**Build Failures:**
```bash
# Clear Docker cache
docker system prune -a

# Rebuild from scratch
./scripts/build.sh
```

**Performance Issues:**
```bash
# Check system resources
docker stats helios-demo

# View logs
docker logs -f helios-demo
```

## 🔗 Links & Resources

- **Demo URL**: http://localhost:3000 (when running)
- **API Docs**: http://localhost:3001/api/stats
- **GitHub**: https://github.com/your-org/helios-demo
- **Technical Paper**: [Helios Architecture Deep Dive]

## 📝 License

MIT License - Feel free to use this demo for educational and commercial purposes.

## 🤝 Contributing

This demo showcases the core Helios concepts. For production implementations or contributions to the actual Helios engine, please contact the development team.

---

**Ready to experience the future of AI verification?**

```bash
./scripts/run.sh
```

Open http://localhost:3000 and watch thousands of parallel universes come to life! 🌌