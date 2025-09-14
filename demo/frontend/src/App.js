import React, { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import io from 'socket.io-client';
import UniverseVisualizer from './components/UniverseVisualizer';
import StatsPanel from './components/StatsPanel';
import ControlPanel from './components/ControlPanel';
import TimingComparison from './components/TimingComparison';
import './App.css';

function App() {
  const [socket, setSocket] = useState(null);
  const [stats, setStats] = useState({
    universeCount: 0,
    memoryUsed: 0,
    avgCommitLatency: 0,
    memoryEfficiency: 1.0,
    uptime: 0
  });
  const [universes, setUniverses] = useState([]);
  const [isConnected, setIsConnected] = useState(false);
  const [demoPhase, setDemoPhase] = useState('intro'); // intro, creating, monitoring, comparison

  useEffect(() => {
    const socketConnection = io(process.env.REACT_APP_BACKEND_URL || 'http://localhost:3002');

    socketConnection.on('connect', () => {
      setIsConnected(true);
      console.log('🚀 Connected to Helios Engine');
    });

    socketConnection.on('disconnect', () => {
      setIsConnected(false);
    });

    socketConnection.on('stats_update', (newStats) => {
      setStats(newStats);
    });

    socketConnection.on('universes_created', (newUniverses) => {
      setUniverses(prev => [...prev, ...newUniverses]);
    });

    socketConnection.on('creation_progress', (progress) => {
      setUniverses(prev => [...prev, ...progress.batch]);
    });

    socketConnection.on('creation_complete', (result) => {
      console.log(`✨ Created ${result.totalCreated} parallel universes`);
    });

    socketConnection.on('universe_rollback', (data) => {
      setUniverses(prev =>
        prev.filter(u => u.id !== data.universeId && u.parentId !== data.universeId)
      );
    });

    socketConnection.on('gc_complete', (data) => {
      console.log(`🗑️ Garbage collected ${data.collected} universes`);
    });

    setSocket(socketConnection);

    return () => {
      socketConnection.disconnect();
    };
  }, []);

  const handleStartDemo = () => {
    setDemoPhase('creating');
    if (socket) {
      socket.emit('create_demo_universes', { count: 1000 });
    }
  };

  const handleCreateUniverses = (count) => {
    if (socket) {
      socket.emit('create_demo_universes', { count });
    }
  };

  const handleRunGC = () => {
    if (socket) {
      socket.emit('run_gc');
    }
  };

  const formatBytes = (bytes) => {
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
    if (bytes === 0) return '0 Bytes';
    const i = parseInt(Math.floor(Math.log(bytes) / Math.log(1024)));
    return Math.round(bytes / Math.pow(1024, i) * 100) / 100 + ' ' + sizes[i];
  };

  const formatMicroseconds = (microseconds) => {
    if (microseconds < 1000) {
      return `${microseconds.toFixed(1)}μs`;
    }
    return `${(microseconds / 1000).toFixed(1)}ms`;
  };

  return (
    <div className="app">
      <motion.header
        className="app-header"
        initial={{ opacity: 0, y: -50 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.8 }}
      >
        <div className="header-content">
          <h1 className="title">
            <span className="helios-logo">⚡</span>
            Helios
            <span className="subtitle">Parallel Universe Engine</span>
          </h1>
          <div className="connection-status">
            <div className={`status-indicator ${isConnected ? 'connected' : 'disconnected'}`}>
              {isConnected ? '🟢 Connected' : '🔴 Disconnected'}
            </div>
          </div>
        </div>
      </motion.header>

      <main className="main-content">
        {demoPhase === 'intro' && (
          <motion.div
            className="intro-section"
            initial={{ opacity: 0, scale: 0.9 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ duration: 0.6 }}
          >
            <div className="hero-text">
              <h2>From 83 Hours to 5 Minutes</h2>
              <p className="hero-description">
                Experience how Helios transforms AI verification by creating thousands of
                lightweight "parallel universes" instead of expensive physical copies.
              </p>
              <div className="timing-preview">
                <div className="timing-bar serial">
                  <span className="timing-label">Traditional Serial:</span>
                  <div className="bar-container">
                    <div className="bar serial-bar">83 hours</div>
                  </div>
                </div>
                <div className="timing-bar parallel">
                  <span className="timing-label">Helios Parallel:</span>
                  <div className="bar-container">
                    <div className="bar helios-bar">5 minutes</div>
                  </div>
                </div>
              </div>
              <button
                className="start-demo-btn"
                onClick={handleStartDemo}
                disabled={!isConnected}
              >
                🚀 Start Interactive Demo
              </button>
            </div>
          </motion.div>
        )}

        {demoPhase !== 'intro' && (
          <div className="demo-grid">
            <motion.div
              className="stats-section"
              initial={{ opacity: 0, x: -50 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ delay: 0.2 }}
            >
              <StatsPanel
                stats={stats}
                formatBytes={formatBytes}
                formatMicroseconds={formatMicroseconds}
              />
            </motion.div>

            <motion.div
              className="visualizer-section"
              initial={{ opacity: 0, y: 50 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.4 }}
            >
              <UniverseVisualizer
                universes={universes.slice(0, 200)} // Limit for performance
                stats={stats}
              />
            </motion.div>

            <motion.div
              className="controls-section"
              initial={{ opacity: 0, x: 50 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ delay: 0.6 }}
            >
              <ControlPanel
                onCreateUniverses={handleCreateUniverses}
                onRunGC={handleRunGC}
                isConnected={isConnected}
              />
            </motion.div>

            <motion.div
              className="comparison-section"
              initial={{ opacity: 0, y: 50 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.8 }}
            >
              <TimingComparison
                universeCount={stats.universeCount}
              />
            </motion.div>
          </div>
        )}
      </main>

      <footer className="app-footer">
        <p>
          🌌 Experience the power of parallel universe state management
          <span className="footer-link">
            <a href="https://github.com/helios-engine" target="_blank" rel="noopener noreferrer">
              View on GitHub
            </a>
          </span>
        </p>
      </footer>
    </div>
  );
}

export default App;