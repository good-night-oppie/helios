const express = require('express');
const http = require('http');
const socketIo = require('socket.io');
const cors = require('cors');
const { v4: uuidv4 } = require('uuid');

const app = express();
const server = http.createServer(app);
const io = socketIo(server, {
  cors: {
    origin: "*",
    methods: ["GET", "POST"]
  }
});

app.use(cors());
app.use(express.json());

// Helios Engine Simulation State
class HeliosEngine {
  constructor() {
    this.universes = new Map();
    this.baseMemory = 50 * 1024 * 1024 * 1024; // 50GB base project
    this.startTime = Date.now();
    this.stats = {
      totalUniverses: 0,
      memoryUsed: this.baseMemory,
      avgCommitLatency: 0,
      operationsPerSecond: 0
    };
  }

  // Create a new parallel universe (snapshot)
  createUniverse(parentId = null) {
    const id = uuidv4();
    const universe = {
      id,
      parentId,
      createdAt: Date.now(),
      memoryDiff: Math.random() * 0.002 * this.baseMemory, // 0.1% avg diff
      status: 'active',
      testResults: [],
      operations: 0
    };

    this.universes.set(id, universe);
    this.stats.totalUniverses++;
    this.updateMemoryUsage();

    return universe;
  }

  // Branch from existing universe
  branchUniverse(parentId) {
    const parent = this.universes.get(parentId);
    if (!parent) throw new Error('Parent universe not found');

    return this.createUniverse(parentId);
  }

  // Simulate rollback (destroy universe and children)
  rollbackUniverse(universeId) {
    const universe = this.universes.get(universeId);
    if (!universe) return false;

    // Find and remove all children
    const toRemove = [universeId];
    const findChildren = (parentId) => {
      this.universes.forEach((u, id) => {
        if (u.parentId === parentId) {
          toRemove.push(id);
          findChildren(id);
        }
      });
    };

    findChildren(universeId);
    toRemove.forEach(id => this.universes.delete(id));

    this.stats.totalUniverses -= toRemove.length;
    this.updateMemoryUsage();

    return toRemove.length;
  }

  // Simulate VST commit operation with realistic latency
  commitToUniverse(universeId, changes) {
    const universe = this.universes.get(universeId);
    if (!universe) throw new Error('Universe not found');

    // Simulate <70μs commit latency
    const startTime = process.hrtime.bigint();

    // Simulate commit work
    universe.operations++;
    universe.memoryDiff += changes.size || (Math.random() * 0.001 * this.baseMemory);

    const endTime = process.hrtime.bigint();
    const latencyMicroseconds = Number(endTime - startTime) / 1000;

    // Update running average of commit latency
    this.stats.avgCommitLatency = (this.stats.avgCommitLatency + latencyMicroseconds) / 2;

    return {
      universeId,
      latencyMicroseconds: Math.min(latencyMicroseconds, 70), // Cap at 70μs for demo
      success: true
    };
  }

  // Update memory usage calculation (COW efficiency)
  updateMemoryUsage() {
    const totalDiff = Array.from(this.universes.values())
      .reduce((sum, u) => sum + u.memoryDiff, 0);

    const metadata = this.universes.size * 1024; // 1KB metadata per universe
    this.stats.memoryUsed = this.baseMemory + totalDiff + metadata;
  }

  // Simulate traditional serial execution time calculation
  calculateSerialTime(universeCount) {
    const avgTestTime = 5 * 60 * 1000; // 5 minutes per test in ms
    return universeCount * avgTestTime;
  }

  // Calculate Helios parallel time (licensing model)
  calculateHeliosTime(universeCount) {
    const licenseInstances = Math.min(8, Math.ceil(universeCount / 100)); // 4-8 license instances
    const avgTestTime = 5 * 60 * 1000; // 5 minutes per test
    return Math.ceil(universeCount / licenseInstances) * avgTestTime;
  }

  // Get current engine statistics
  getStats() {
    return {
      ...this.stats,
      universeCount: this.universes.size,
      memoryEfficiency: this.stats.memoryUsed / (this.baseMemory * Math.max(1, this.universes.size)),
      uptime: Date.now() - this.startTime
    };
  }

  // Simulate garbage collection
  runGarbageCollection() {
    let collected = 0;
    const cutoffTime = Date.now() - (5 * 60 * 1000); // 5 minutes TTL

    this.universes.forEach((universe, id) => {
      if (universe.status === 'completed' && universe.createdAt < cutoffTime) {
        this.universes.delete(id);
        collected++;
      }
    });

    this.stats.totalUniverses -= collected;
    this.updateMemoryUsage();
    return collected;
  }
}

const helios = new HeliosEngine();

// REST API Endpoints
app.get('/api/stats', (req, res) => {
  res.json(helios.getStats());
});

app.get('/api/universes', (req, res) => {
  const universeList = Array.from(helios.universes.values())
    .slice(0, 100); // Limit for performance
  res.json(universeList);
});

app.post('/api/universes', (req, res) => {
  try {
    const { parentId, count = 1 } = req.body;
    const created = [];

    for (let i = 0; i < Math.min(count, 1000); i++) {
      const universe = helios.createUniverse(parentId);
      created.push(universe);
    }

    // Broadcast to all connected clients
    io.emit('universes_created', created);
    res.json(created);
  } catch (error) {
    res.status(400).json({ error: error.message });
  }
});

app.post('/api/universes/:id/commit', (req, res) => {
  try {
    const result = helios.commitToUniverse(req.params.id, req.body);
    io.emit('commit_completed', result);
    res.json(result);
  } catch (error) {
    res.status(400).json({ error: error.message });
  }
});

app.delete('/api/universes/:id', (req, res) => {
  try {
    const removedCount = helios.rollbackUniverse(req.params.id);
    io.emit('universe_rollback', { universeId: req.params.id, removedCount });
    res.json({ success: true, removedCount });
  } catch (error) {
    res.status(400).json({ error: error.message });
  }
});

app.post('/api/simulate/:type', (req, res) => {
  const { type } = req.params;
  const { count = 1000 } = req.body;

  let result;
  if (type === 'serial') {
    result = {
      type: 'serial',
      universeCount: count,
      estimatedTime: helios.calculateSerialTime(count),
      description: 'Traditional serial execution'
    };
  } else if (type === 'parallel') {
    result = {
      type: 'parallel',
      universeCount: count,
      estimatedTime: helios.calculateHeliosTime(count),
      description: 'Helios parallel execution'
    };
  }

  res.json(result);
});

// WebSocket Connection Handling
io.on('connection', (socket) => {
  console.log('Client connected:', socket.id);

  // Send initial stats
  socket.emit('stats_update', helios.getStats());

  // Handle real-time universe creation requests
  socket.on('create_demo_universes', (data) => {
    const { count } = data;
    const batchSize = 50;
    let created = 0;

    const createBatch = () => {
      const currentBatch = Math.min(batchSize, count - created);
      const universes = [];

      for (let i = 0; i < currentBatch; i++) {
        const universe = helios.createUniverse();
        universes.push(universe);
      }

      created += currentBatch;

      // Emit progress update
      socket.emit('creation_progress', {
        created,
        total: count,
        batch: universes,
        progress: created / count
      });

      if (created < count) {
        setTimeout(createBatch, 50); // 50ms between batches for smooth animation
      } else {
        socket.emit('creation_complete', { totalCreated: created });
      }
    };

    createBatch();
  });

  // Handle cleanup request
  socket.on('run_gc', () => {
    const collected = helios.runGarbageCollection();
    socket.emit('gc_complete', { collected });
  });

  socket.on('disconnect', () => {
    console.log('Client disconnected:', socket.id);
  });
});

// Periodic stats broadcast
setInterval(() => {
  io.emit('stats_update', helios.getStats());
}, 1000); // Every second

// Periodic GC simulation
setInterval(() => {
  const collected = helios.runGarbageCollection();
  if (collected > 0) {
    io.emit('gc_auto', { collected });
  }
}, 30000); // Every 30 seconds

const PORT = process.env.PORT || 3002;
server.listen(PORT, '0.0.0.0', () => {
  console.log(`🚀 Helios Demo Backend running on port ${PORT}`);
  console.log(`🌌 Parallel Universe Engine initialized`);
  console.log(`📊 Base memory: ${(helios.baseMemory / (1024**3)).toFixed(1)}GB`);
});