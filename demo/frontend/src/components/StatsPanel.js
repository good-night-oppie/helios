import React from 'react';
import { motion } from 'framer-motion';

const StatsPanel = ({ stats, formatBytes, formatMicroseconds }) => {
  const statsData = [
    {
      label: 'Active Universes',
      value: stats.universeCount?.toLocaleString() || '0',
      icon: '🌌',
      color: '#00d4ff'
    },
    {
      label: 'Memory Used',
      value: formatBytes(stats.memoryUsed || 0),
      icon: '💾',
      color: '#10b981',
      subtitle: `Efficiency: ${((stats.memoryEfficiency || 1) * 100).toFixed(1)}%`
    },
    {
      label: 'VST Commit Latency',
      value: formatMicroseconds(stats.avgCommitLatency || 0),
      icon: '⚡',
      color: '#f59e0b',
      subtitle: 'P95 Target: <70μs'
    },
    {
      label: 'Total Created',
      value: stats.totalUniverses?.toLocaleString() || '0',
      icon: '📊',
      color: '#8b5cf6'
    },
    {
      label: 'Uptime',
      value: formatUptime(stats.uptime || 0),
      icon: '⏱️',
      color: '#ef4444'
    }
  ];

  function formatUptime(ms) {
    const seconds = Math.floor(ms / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);

    if (hours > 0) {
      return `${hours}h ${minutes % 60}m`;
    } else if (minutes > 0) {
      return `${minutes}m ${seconds % 60}s`;
    } else {
      return `${seconds}s`;
    }
  }

  return (
    <div className="panel stats-panel">
      <h3>
        <span>📈</span>
        Engine Statistics
      </h3>
      <div className="stats-grid">
        {statsData.map((stat, index) => (
          <motion.div
            key={stat.label}
            className="stat-item"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: index * 0.1 }}
            whileHover={{ scale: 1.02 }}
          >
            <div className="stat-header">
              <span className="stat-icon">{stat.icon}</span>
              <span className="stat-label">{stat.label}</span>
            </div>
            <div className="stat-value" style={{ color: stat.color }}>
              {stat.value}
            </div>
            {stat.subtitle && (
              <div className="stat-subtitle">{stat.subtitle}</div>
            )}
            <div className="stat-bar">
              <motion.div
                className="stat-fill"
                style={{ backgroundColor: stat.color }}
                initial={{ width: 0 }}
                animate={{ width: getBarWidth(stat.label, stats) }}
                transition={{ duration: 0.8, delay: index * 0.1 }}
              />
            </div>
          </motion.div>
        ))}
      </div>

      <div className="memory-breakdown">
        <h4>Memory Breakdown</h4>
        <div className="breakdown-item">
          <span>Base Project:</span>
          <span>{formatBytes(50 * 1024 * 1024 * 1024)}</span>
        </div>
        <div className="breakdown-item">
          <span>Universe Diffs:</span>
          <span>{formatBytes((stats.memoryUsed || 0) - (50 * 1024 * 1024 * 1024))}</span>
        </div>
        <div className="breakdown-item total">
          <span>Total Memory:</span>
          <span>{formatBytes(stats.memoryUsed || 0)}</span>
        </div>
        <div className="savings-indicator">
          <span>💰 Traditional Cost: </span>
          <span className="savings-amount">
            {formatBytes((stats.universeCount || 0) * 50 * 1024 * 1024 * 1024)}
          </span>
        </div>
      </div>

      <style jsx>{`
        .stats-panel {
          height: auto;
        }

        .stats-grid {
          display: flex;
          flex-direction: column;
          gap: 1rem;
          margin-bottom: 1.5rem;
        }

        .stat-item {
          background: rgba(255, 255, 255, 0.03);
          border-radius: 12px;
          padding: 1rem;
          border: 1px solid rgba(255, 255, 255, 0.05);
          transition: all 0.3s ease;
        }

        .stat-item:hover {
          border-color: rgba(255, 255, 255, 0.1);
          background: rgba(255, 255, 255, 0.05);
        }

        .stat-header {
          display: flex;
          align-items: center;
          gap: 0.5rem;
          margin-bottom: 0.5rem;
        }

        .stat-icon {
          font-size: 1.2rem;
        }

        .stat-label {
          font-size: 0.9rem;
          color: #ccc;
          font-weight: 500;
        }

        .stat-value {
          font-size: 1.5rem;
          font-weight: 700;
          margin-bottom: 0.3rem;
        }

        .stat-subtitle {
          font-size: 0.8rem;
          color: #888;
          margin-bottom: 0.5rem;
        }

        .stat-bar {
          height: 4px;
          background: rgba(255, 255, 255, 0.1);
          border-radius: 2px;
          overflow: hidden;
        }

        .stat-fill {
          height: 100%;
          transition: width 0.8s ease;
          border-radius: 2px;
        }

        .memory-breakdown {
          border-top: 1px solid rgba(255, 255, 255, 0.1);
          padding-top: 1rem;
        }

        .memory-breakdown h4 {
          color: #00d4ff;
          margin-bottom: 0.8rem;
          font-size: 1rem;
        }

        .breakdown-item {
          display: flex;
          justify-content: space-between;
          margin-bottom: 0.5rem;
          font-size: 0.9rem;
        }

        .breakdown-item.total {
          border-top: 1px solid rgba(255, 255, 255, 0.1);
          padding-top: 0.5rem;
          font-weight: 600;
          color: #00d4ff;
        }

        .savings-indicator {
          background: rgba(34, 197, 94, 0.1);
          border: 1px solid rgba(34, 197, 94, 0.2);
          border-radius: 8px;
          padding: 0.8rem;
          margin-top: 1rem;
          display: flex;
          justify-content: space-between;
          font-size: 0.9rem;
        }

        .savings-amount {
          color: #22c55e;
          font-weight: 600;
        }
      `}</style>
    </div>
  );
};

function getBarWidth(label, stats) {
  switch (label) {
    case 'Active Universes':
      return `${Math.min((stats.universeCount / 1000) * 100, 100)}%`;
    case 'Memory Used':
      return `${Math.min((stats.memoryEfficiency * 50), 100)}%`;
    case 'VST Commit Latency':
      return `${Math.min((stats.avgCommitLatency / 70) * 100, 100)}%`;
    case 'Total Created':
      return `${Math.min((stats.totalUniverses / 5000) * 100, 100)}%`;
    case 'Uptime':
      return `${Math.min((stats.uptime / (1000 * 60 * 30)) * 100, 100)}%`;
    default:
      return '0%';
  }
}

export default StatsPanel;