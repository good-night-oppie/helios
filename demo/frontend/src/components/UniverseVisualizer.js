import React, { useEffect, useState } from 'react';
import { motion } from 'framer-motion';

const UniverseVisualizer = ({ universes, stats }) => {
  const [visibleUniverses, setVisibleUniverses] = useState([]);
  const [viewMode, setViewMode] = useState('grid'); // grid, flow, tree

  useEffect(() => {
    // Limit visible universes for performance
    setVisibleUniverses(universes.slice(0, 100));
  }, [universes]);

  const renderGridView = () => (
    <div className="universe-grid">
      {visibleUniverses.map((universe, index) => (
        <motion.div
          key={universe.id}
          className="universe-node"
          initial={{ opacity: 0, scale: 0 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{
            delay: index * 0.01,
            duration: 0.3
          }}
          whileHover={{ scale: 1.1 }}
          style={{
            backgroundColor: getUniverseColor(universe),
            animationDelay: `${index * 50}ms`
          }}
        >
          <div className="universe-id">#{index + 1}</div>
          <div className="universe-status">
            {universe.status === 'active' ? '🟢' : '🔵'}
          </div>
        </motion.div>
      ))}
    </div>
  );

  const renderFlowView = () => (
    <div className="universe-flow">
      <svg width="100%" height="100%" className="flow-svg">
        {/* Background grid */}
        <defs>
          <pattern
            id="grid"
            width="20"
            height="20"
            patternUnits="userSpaceOnUse"
          >
            <path
              d="M 20 0 L 0 0 0 20"
              fill="none"
              stroke="rgba(255,255,255,0.1)"
              strokeWidth="1"
            />
          </pattern>
        </defs>
        <rect width="100%" height="100%" fill="url(#grid)" />

        {/* Universe nodes with connections */}
        {visibleUniverses.map((universe, index) => {
          const x = (index % 10) * 60 + 30;
          const y = Math.floor(index / 10) * 60 + 30;

          return (
            <g key={universe.id}>
              {/* Connection lines to parent */}
              {universe.parentId && (
                <motion.line
                  x1={x}
                  y1={y}
                  x2={x - 60}
                  y2={y - 60}
                  stroke="#00d4ff"
                  strokeWidth="2"
                  opacity="0.6"
                  initial={{ pathLength: 0 }}
                  animate={{ pathLength: 1 }}
                  transition={{ duration: 0.8, delay: index * 0.05 }}
                />
              )}

              {/* Universe node */}
              <motion.circle
                cx={x}
                cy={y}
                r="15"
                fill={getUniverseColor(universe)}
                stroke="#00d4ff"
                strokeWidth="2"
                initial={{ r: 0 }}
                animate={{ r: 15 }}
                transition={{ duration: 0.5, delay: index * 0.02 }}
                className="universe-circle"
              />

              {/* Node label */}
              <text
                x={x}
                y={y + 5}
                textAnchor="middle"
                fill="white"
                fontSize="10"
                fontWeight="bold"
              >
                {index + 1}
              </text>
            </g>
          );
        })}
      </svg>
    </div>
  );

  const renderTreeView = () => (
    <div className="universe-tree">
      <div className="tree-container">
        {/* Root node */}
        <div className="tree-level">
          <div className="tree-node root-node">
            <span>🌟</span>
            <span>Origin</span>
          </div>
        </div>

        {/* Child universes grouped by generation */}
        {[1, 2, 3, 4].map((level) => {
          const levelUniverses = visibleUniverses
            .filter((u, i) => Math.floor(i / 8) === level - 1)
            .slice(0, 8);

          if (levelUniverses.length === 0) return null;

          return (
            <div key={level} className="tree-level">
              {levelUniverses.map((universe, index) => (
                <motion.div
                  key={universe.id}
                  className="tree-node"
                  initial={{ opacity: 0, y: 20 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ delay: level * 0.2 + index * 0.1 }}
                >
                  <span className="tree-node-icon">🌌</span>
                  <span className="tree-node-label">U{index + 1}</span>
                  <div className="tree-node-status">
                    {universe.status === 'active' ? '🟢' : '🔵'}
                  </div>
                </motion.div>
              ))}
            </div>
          );
        })}
      </div>
    </div>
  );

  function getUniverseColor(universe) {
    const age = Date.now() - universe.createdAt;
    if (age < 1000) return 'rgba(0, 212, 255, 0.9)'; // New - bright blue
    if (age < 5000) return 'rgba(0, 212, 255, 0.7)'; // Recent - blue
    if (age < 15000) return 'rgba(0, 212, 255, 0.5)'; // Older - dim blue
    return 'rgba(0, 212, 255, 0.3)'; // Old - very dim
  }

  return (
    <div className="panel visualizer-panel">
      <div className="visualizer-header">
        <h3>
          <span>🌌</span>
          Parallel Universes
          <span className="universe-count">({stats.universeCount || 0})</span>
        </h3>
        <div className="view-controls">
          {['grid', 'flow', 'tree'].map((mode) => (
            <button
              key={mode}
              className={`view-btn ${viewMode === mode ? 'active' : ''}`}
              onClick={() => setViewMode(mode)}
            >
              {mode === 'grid' && '⊞'}
              {mode === 'flow' && '⚡'}
              {mode === 'tree' && '🌳'}
            </button>
          ))}
        </div>
      </div>

      <div className="visualizer-content">
        {viewMode === 'grid' && renderGridView()}
        {viewMode === 'flow' && renderFlowView()}
        {viewMode === 'tree' && renderTreeView()}
      </div>

      {visibleUniverses.length < universes.length && (
        <div className="overflow-indicator">
          Showing {visibleUniverses.length} of {universes.length} universes
        </div>
      )}

      <style jsx>{`
        .visualizer-panel {
          height: 100%;
          display: flex;
          flex-direction: column;
        }

        .visualizer-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 1rem;
          padding-bottom: 1rem;
          border-bottom: 1px solid rgba(255, 255, 255, 0.1);
        }

        .universe-count {
          font-size: 0.9rem;
          color: #888;
          font-weight: normal;
          margin-left: 0.5rem;
        }

        .view-controls {
          display: flex;
          gap: 0.5rem;
        }

        .view-btn {
          padding: 0.5rem 0.8rem;
          background: rgba(255, 255, 255, 0.1);
          border: 1px solid rgba(255, 255, 255, 0.2);
          border-radius: 8px;
          color: white;
          cursor: pointer;
          font-size: 1.2rem;
          transition: all 0.3s ease;
        }

        .view-btn:hover {
          background: rgba(255, 255, 255, 0.2);
        }

        .view-btn.active {
          background: rgba(0, 212, 255, 0.3);
          border-color: #00d4ff;
        }

        .visualizer-content {
          flex: 1;
          overflow: hidden;
          position: relative;
        }

        /* Grid View Styles */
        .universe-grid {
          display: grid;
          grid-template-columns: repeat(auto-fill, minmax(40px, 1fr));
          gap: 8px;
          height: 100%;
          overflow-y: auto;
          padding: 1rem;
        }

        .universe-node {
          width: 40px;
          height: 40px;
          border-radius: 50%;
          display: flex;
          flex-direction: column;
          align-items: center;
          justify-content: center;
          cursor: pointer;
          position: relative;
          animation: pulse 2s infinite;
        }

        .universe-id {
          font-size: 0.7rem;
          font-weight: bold;
          color: white;
        }

        .universe-status {
          font-size: 0.6rem;
          position: absolute;
          bottom: -2px;
          right: -2px;
        }

        /* Flow View Styles */
        .universe-flow {
          height: 100%;
          overflow: auto;
        }

        .flow-svg {
          min-height: 400px;
        }

        .universe-circle {
          cursor: pointer;
          filter: drop-shadow(0 0 10px rgba(0, 212, 255, 0.5));
        }

        .universe-circle:hover {
          filter: drop-shadow(0 0 15px rgba(0, 212, 255, 0.8));
        }

        /* Tree View Styles */
        .universe-tree {
          height: 100%;
          overflow-y: auto;
          padding: 1rem;
        }

        .tree-container {
          display: flex;
          flex-direction: column;
          align-items: center;
          gap: 2rem;
        }

        .tree-level {
          display: flex;
          gap: 1rem;
          justify-content: center;
          flex-wrap: wrap;
        }

        .tree-node {
          display: flex;
          flex-direction: column;
          align-items: center;
          gap: 0.3rem;
          padding: 0.8rem;
          background: rgba(255, 255, 255, 0.1);
          border: 1px solid rgba(255, 255, 255, 0.2);
          border-radius: 12px;
          min-width: 60px;
          cursor: pointer;
          transition: all 0.3s ease;
          position: relative;
        }

        .tree-node:hover {
          background: rgba(255, 255, 255, 0.15);
          transform: translateY(-2px);
        }

        .tree-node.root-node {
          background: rgba(0, 212, 255, 0.2);
          border-color: #00d4ff;
        }

        .tree-node-icon {
          font-size: 1.5rem;
        }

        .tree-node-label {
          font-size: 0.8rem;
          font-weight: bold;
          color: white;
        }

        .tree-node-status {
          font-size: 0.8rem;
        }

        .overflow-indicator {
          text-align: center;
          padding: 1rem;
          color: #888;
          font-size: 0.9rem;
          border-top: 1px solid rgba(255, 255, 255, 0.1);
          margin-top: auto;
        }

        @keyframes pulse {
          0%, 100% {
            box-shadow: 0 0 0 0 rgba(0, 212, 255, 0.7);
          }
          70% {
            box-shadow: 0 0 0 10px rgba(0, 212, 255, 0);
          }
        }
      `}</style>
    </div>
  );
};

export default UniverseVisualizer;