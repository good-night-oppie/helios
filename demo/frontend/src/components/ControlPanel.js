import React, { useState } from 'react';
import { motion } from 'framer-motion';

const ControlPanel = ({ onCreateUniverses, onRunGC, isConnected }) => {
  const [selectedCount, setSelectedCount] = useState(100);
  const [isCreating, setIsCreating] = useState(false);

  const presetCounts = [10, 100, 500, 1000, 5000];

  const handleCreateUniverses = async () => {
    if (!isConnected || isCreating) return;

    setIsCreating(true);
    try {
      await onCreateUniverses(selectedCount);
      // Reset after a delay
      setTimeout(() => setIsCreating(false), 2000);
    } catch (error) {
      console.error('Failed to create universes:', error);
      setIsCreating(false);
    }
  };

  const handleRunGC = () => {
    if (!isConnected) return;
    onRunGC();
  };

  return (
    <div className="panel control-panel">
      <h3>
        <span>🎮</span>
        Demo Controls
      </h3>

      <div className="control-section">
        <h4>Create Parallel Universes</h4>
        <div className="count-selector">
          <label htmlFor="count-input">Number of Universes:</label>
          <div className="count-input-group">
            <input
              id="count-input"
              type="number"
              value={selectedCount}
              onChange={(e) => setSelectedCount(Math.max(1, Math.min(10000, parseInt(e.target.value) || 1)))}
              min="1"
              max="10000"
              className="count-input"
            />
            <span className="count-unit">universes</span>
          </div>
        </div>

        <div className="preset-buttons">
          {presetCounts.map((count) => (
            <motion.button
              key={count}
              className={`preset-btn ${selectedCount === count ? 'active' : ''}`}
              onClick={() => setSelectedCount(count)}
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
            >
              {count.toLocaleString()}
            </motion.button>
          ))}
        </div>

        <motion.button
          className="create-btn"
          onClick={handleCreateUniverses}
          disabled={!isConnected || isCreating}
          whileHover={{ scale: isConnected && !isCreating ? 1.02 : 1 }}
          whileTap={{ scale: isConnected && !isCreating ? 0.98 : 1 }}
        >
          {isCreating ? (
            <>
              <span className="spinner">🌀</span>
              Creating...
            </>
          ) : (
            <>
              <span>🚀</span>
              Create {selectedCount.toLocaleString()} Universes
            </>
          )}
        </motion.button>
      </div>

      <div className="control-section">
        <h4>Memory Management</h4>
        <motion.button
          className="gc-btn"
          onClick={handleRunGC}
          disabled={!isConnected}
          whileHover={{ scale: isConnected ? 1.02 : 1 }}
          whileTap={{ scale: isConnected ? 0.98 : 1 }}
        >
          <span>🗑️</span>
          Run Garbage Collection
        </motion.button>
        <div className="gc-description">
          Clean up unused universes and free memory
        </div>
      </div>

      <div className="control-section">
        <h4>Performance Simulation</h4>
        <div className="simulation-info">
          <div className="info-item">
            <span className="info-label">Serial Time:</span>
            <span className="info-value serial">
              {formatTime(selectedCount * 5 * 60 * 1000)}
            </span>
          </div>
          <div className="info-item">
            <span className="info-label">Helios Time:</span>
            <span className="info-value helios">
              {formatTime(Math.ceil(selectedCount / 8) * 5 * 60 * 1000)}
            </span>
          </div>
          <div className="info-item">
            <span className="info-label">Speedup:</span>
            <span className="info-value speedup">
              {(selectedCount / Math.ceil(selectedCount / 8)).toFixed(1)}x
            </span>
          </div>
        </div>
      </div>

      <div className="control-section">
        <h4>Demo Scenarios</h4>
        <div className="scenario-buttons">
          <motion.button
            className="scenario-btn"
            onClick={() => setSelectedCount(1000)}
            whileHover={{ scale: 1.02 }}
          >
            🔬 EDA Simulation
            <span className="scenario-desc">1K parallel tests</span>
          </motion.button>
          <motion.button
            className="scenario-btn"
            onClick={() => setSelectedCount(5000)}
            whileHover={{ scale: 1.02 }}
          >
            💰 Quantitative Finance
            <span className="scenario-desc">5K risk scenarios</span>
          </motion.button>
          <motion.button
            className="scenario-btn"
            onClick={() => setSelectedCount(100)}
            whileHover={{ scale: 1.02 }}
          >
            🧬 Drug Discovery
            <span className="scenario-desc">100 molecular simulations</span>
          </motion.button>
        </div>
      </div>

      <style jsx>{`
        .control-panel {
          height: auto;
          overflow-y: auto;
        }

        .control-section {
          margin-bottom: 2rem;
          padding-bottom: 1.5rem;
          border-bottom: 1px solid rgba(255, 255, 255, 0.05);
        }

        .control-section:last-child {
          border-bottom: none;
          margin-bottom: 0;
        }

        .control-section h4 {
          color: #00d4ff;
          margin-bottom: 1rem;
          font-size: 1rem;
          display: flex;
          align-items: center;
          gap: 0.5rem;
        }

        /* Count Selector */
        .count-selector {
          margin-bottom: 1rem;
        }

        .count-selector label {
          display: block;
          margin-bottom: 0.5rem;
          color: #ccc;
          font-size: 0.9rem;
        }

        .count-input-group {
          display: flex;
          align-items: center;
          gap: 0.5rem;
        }

        .count-input {
          flex: 1;
          padding: 0.5rem 0.8rem;
          background: rgba(255, 255, 255, 0.1);
          border: 1px solid rgba(255, 255, 255, 0.2);
          border-radius: 8px;
          color: white;
          font-size: 1rem;
          outline: none;
        }

        .count-input:focus {
          border-color: #00d4ff;
          box-shadow: 0 0 0 2px rgba(0, 212, 255, 0.2);
        }

        .count-unit {
          color: #888;
          font-size: 0.9rem;
        }

        /* Preset Buttons */
        .preset-buttons {
          display: flex;
          gap: 0.5rem;
          margin-bottom: 1rem;
          flex-wrap: wrap;
        }

        .preset-btn {
          padding: 0.4rem 0.8rem;
          background: rgba(255, 255, 255, 0.1);
          border: 1px solid rgba(255, 255, 255, 0.2);
          border-radius: 6px;
          color: white;
          font-size: 0.9rem;
          cursor: pointer;
          transition: all 0.3s ease;
        }

        .preset-btn:hover {
          background: rgba(255, 255, 255, 0.15);
        }

        .preset-btn.active {
          background: rgba(0, 212, 255, 0.3);
          border-color: #00d4ff;
        }

        /* Action Buttons */
        .create-btn, .gc-btn {
          width: 100%;
          padding: 0.8rem 1rem;
          border: none;
          border-radius: 12px;
          font-size: 1rem;
          font-weight: 600;
          cursor: pointer;
          transition: all 0.3s ease;
          display: flex;
          align-items: center;
          justify-content: center;
          gap: 0.5rem;
        }

        .create-btn {
          background: linear-gradient(135deg, #00d4ff 0%, #0284c7 100%);
          color: white;
          margin-bottom: 1rem;
        }

        .create-btn:hover:not(:disabled) {
          box-shadow: 0 8px 25px rgba(0, 212, 255, 0.3);
        }

        .create-btn:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }

        .gc-btn {
          background: linear-gradient(135deg, #10b981 0%, #059669 100%);
          color: white;
        }

        .gc-btn:hover:not(:disabled) {
          box-shadow: 0 8px 25px rgba(16, 185, 129, 0.3);
        }

        .gc-btn:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }

        .spinner {
          animation: spin 1s linear infinite;
        }

        @keyframes spin {
          from { transform: rotate(0deg); }
          to { transform: rotate(360deg); }
        }

        .gc-description {
          margin-top: 0.5rem;
          font-size: 0.8rem;
          color: #888;
          text-align: center;
        }

        /* Performance Simulation */
        .simulation-info {
          background: rgba(255, 255, 255, 0.03);
          border: 1px solid rgba(255, 255, 255, 0.05);
          border-radius: 8px;
          padding: 1rem;
        }

        .info-item {
          display: flex;
          justify-content: space-between;
          margin-bottom: 0.5rem;
          font-size: 0.9rem;
        }

        .info-item:last-child {
          margin-bottom: 0;
          padding-top: 0.5rem;
          border-top: 1px solid rgba(255, 255, 255, 0.1);
        }

        .info-label {
          color: #ccc;
        }

        .info-value.serial {
          color: #ef4444;
        }

        .info-value.helios {
          color: #00d4ff;
        }

        .info-value.speedup {
          color: #22c55e;
          font-weight: 600;
        }

        /* Scenario Buttons */
        .scenario-buttons {
          display: flex;
          flex-direction: column;
          gap: 0.5rem;
        }

        .scenario-btn {
          padding: 1rem;
          background: rgba(255, 255, 255, 0.05);
          border: 1px solid rgba(255, 255, 255, 0.1);
          border-radius: 12px;
          color: white;
          cursor: pointer;
          text-align: left;
          transition: all 0.3s ease;
          display: flex;
          flex-direction: column;
          gap: 0.3rem;
        }

        .scenario-btn:hover {
          background: rgba(255, 255, 255, 0.1);
          border-color: rgba(255, 255, 255, 0.2);
        }

        .scenario-desc {
          font-size: 0.8rem;
          color: #888;
        }
      `}</style>
    </div>
  );
};

function formatTime(ms) {
  const hours = Math.floor(ms / (1000 * 60 * 60));
  const minutes = Math.floor((ms % (1000 * 60 * 60)) / (1000 * 60));

  if (hours > 0) {
    return `${hours}h ${minutes}m`;
  } else if (minutes > 0) {
    return `${minutes}m`;
  } else {
    return `${Math.floor(ms / 1000)}s`;
  }
}

export default ControlPanel;