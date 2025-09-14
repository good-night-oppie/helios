import React from 'react';
import { motion } from 'framer-motion';

const TimingComparison = ({ universeCount }) => {
  const serialTime = universeCount * 5 * 60 * 1000; // 5 minutes per test
  const heliosTime = Math.ceil(universeCount / 8) * 5 * 60 * 1000; // 8 parallel instances

  const formatTime = (ms) => {
    const hours = Math.floor(ms / (1000 * 60 * 60));
    const minutes = Math.floor((ms % (1000 * 60 * 60)) / (1000 * 60));
    const seconds = Math.floor((ms % (1000 * 60)) / 1000);

    if (hours > 0) {
      return `${hours}h ${minutes}m`;
    } else if (minutes > 0) {
      return `${minutes}m ${seconds > 0 ? `${seconds}s` : ''}`;
    } else {
      return `${seconds}s`;
    }
  };

  const getBarWidth = (time, maxTime) => {
    return Math.max((time / maxTime) * 100, 2); // Minimum 2% for visibility
  };

  const speedup = serialTime / heliosTime;
  const efficiency = ((serialTime - heliosTime) / serialTime) * 100;

  return (
    <div className="panel timing-comparison">
      <h3>
        <span>⚡</span>
        Performance Comparison
        <span className="comparison-count">({universeCount} tests)</span>
      </h3>

      <div className="comparison-content">
        {/* Serial Execution */}
        <div className="timing-row serial">
          <div className="timing-header">
            <div className="timing-title">
              <span className="timing-icon">🐌</span>
              <span>Traditional Serial</span>
            </div>
            <div className="timing-value serial-value">
              {formatTime(serialTime)}
            </div>
          </div>
          <div className="timing-bar-container">
            <motion.div
              className="timing-bar serial-bar"
              initial={{ width: 0 }}
              animate={{ width: '100%' }}
              transition={{ duration: 1.5, delay: 0.5 }}
            >
              <div className="bar-label">One test at a time</div>
            </motion.div>
          </div>
          <div className="timing-details">
            <span>• 1 license per test</span>
            <span>• Full memory per instance</span>
            <span>• No parallelization</span>
          </div>
        </div>

        {/* Helios Parallel */}
        <div className="timing-row helios">
          <div className="timing-header">
            <div className="timing-title">
              <span className="timing-icon">⚡</span>
              <span>Helios Parallel</span>
            </div>
            <div className="timing-value helios-value">
              {formatTime(heliosTime)}
            </div>
          </div>
          <div className="timing-bar-container">
            <motion.div
              className="timing-bar helios-bar"
              initial={{ width: 0 }}
              animate={{ width: `${getBarWidth(heliosTime, serialTime)}%` }}
              transition={{ duration: 1.5, delay: 1 }}
            >
              <div className="bar-label">Parallel universes</div>
            </motion.div>
          </div>
          <div className="timing-details">
            <span>• 8 licenses shared</span>
            <span>• COW memory efficiency</span>
            <span>• Microsecond state switching</span>
          </div>
        </div>

        {/* Savings Summary */}
        <motion.div
          className="savings-summary"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 1.5 }}
        >
          <div className="savings-grid">
            <div className="savings-item">
              <div className="savings-label">Time Saved</div>
              <div className="savings-value time-saved">
                {formatTime(serialTime - heliosTime)}
              </div>
            </div>
            <div className="savings-item">
              <div className="savings-label">Speedup</div>
              <div className="savings-value speedup">
                {speedup.toFixed(1)}x
              </div>
            </div>
            <div className="savings-item">
              <div className="savings-label">Efficiency</div>
              <div className="savings-value efficiency">
                {efficiency.toFixed(1)}%
              </div>
            </div>
          </div>

          <div className="cost-analysis">
            <h4>Cost Analysis</h4>
            <div className="cost-grid">
              <div className="cost-item">
                <span className="cost-label">Traditional Licenses:</span>
                <span className="cost-value">${(universeCount * 50000).toLocaleString()}</span>
              </div>
              <div className="cost-item">
                <span className="cost-label">Helios Licenses:</span>
                <span className="cost-value">${(8 * 50000).toLocaleString()}</span>
              </div>
              <div className="cost-item savings">
                <span className="cost-label">Savings:</span>
                <span className="cost-value">${((universeCount - 8) * 50000).toLocaleString()}</span>
              </div>
            </div>
          </div>
        </motion.div>

        {/* Real-world Examples */}
        <div className="examples-section">
          <h4>Real-world Impact</h4>
          <div className="example-grid">
            <div className="example-item">
              <div className="example-icon">🔬</div>
              <div className="example-content">
                <div className="example-title">EDA Chip Design</div>
                <div className="example-desc">
                  Timing closure verification: {universeCount} layout variants
                </div>
              </div>
            </div>
            <div className="example-item">
              <div className="example-icon">💰</div>
              <div className="example-content">
                <div className="example-title">Risk Analysis</div>
                <div className="example-desc">
                  Portfolio stress testing: {universeCount} market scenarios
                </div>
              </div>
            </div>
            <div className="example-item">
              <div className="example-icon">🧬</div>
              <div className="example-content">
                <div className="example-title">Drug Discovery</div>
                <div className="example-desc">
                  Molecular simulations: {universeCount} compound variants
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <style jsx>{`
        .timing-comparison {
          height: auto;
        }

        .comparison-count {
          font-size: 0.9rem;
          color: #888;
          font-weight: normal;
          margin-left: 0.5rem;
        }

        .comparison-content {
          display: flex;
          flex-direction: column;
          gap: 1.5rem;
        }

        /* Timing Rows */
        .timing-row {
          background: rgba(255, 255, 255, 0.03);
          border: 1px solid rgba(255, 255, 255, 0.05);
          border-radius: 12px;
          padding: 1.2rem;
          transition: all 0.3s ease;
        }

        .timing-row:hover {
          background: rgba(255, 255, 255, 0.05);
          border-color: rgba(255, 255, 255, 0.1);
        }

        .timing-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 1rem;
        }

        .timing-title {
          display: flex;
          align-items: center;
          gap: 0.5rem;
          font-weight: 600;
          font-size: 1.1rem;
        }

        .timing-icon {
          font-size: 1.3rem;
        }

        .timing-value {
          font-size: 1.4rem;
          font-weight: 700;
        }

        .serial-value {
          color: #ef4444;
        }

        .helios-value {
          color: #00d4ff;
        }

        /* Timing Bars */
        .timing-bar-container {
          height: 50px;
          background: rgba(0, 0, 0, 0.3);
          border-radius: 25px;
          overflow: hidden;
          margin-bottom: 0.8rem;
          position: relative;
        }

        .timing-bar {
          height: 100%;
          display: flex;
          align-items: center;
          justify-content: center;
          position: relative;
          border-radius: 25px;
        }

        .serial-bar {
          background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
          box-shadow: 0 0 20px rgba(239, 68, 68, 0.3);
        }

        .helios-bar {
          background: linear-gradient(135deg, #00d4ff 0%, #0284c7 100%);
          box-shadow: 0 0 20px rgba(0, 212, 255, 0.4);
        }

        .bar-label {
          color: white;
          font-weight: 600;
          font-size: 0.9rem;
          text-shadow: 0 1px 2px rgba(0, 0, 0, 0.5);
        }

        .timing-details {
          display: flex;
          flex-direction: column;
          gap: 0.3rem;
          font-size: 0.8rem;
          color: #999;
        }

        /* Savings Summary */
        .savings-summary {
          background: linear-gradient(135deg, rgba(34, 197, 94, 0.1) 0%, rgba(16, 185, 129, 0.1) 100%);
          border: 1px solid rgba(34, 197, 94, 0.2);
          border-radius: 12px;
          padding: 1.5rem;
        }

        .savings-grid {
          display: grid;
          grid-template-columns: repeat(3, 1fr);
          gap: 1rem;
          margin-bottom: 1.5rem;
        }

        .savings-item {
          text-align: center;
        }

        .savings-label {
          font-size: 0.9rem;
          color: #ccc;
          margin-bottom: 0.3rem;
        }

        .savings-value {
          font-size: 1.8rem;
          font-weight: 700;
        }

        .time-saved {
          color: #22c55e;
        }

        .speedup {
          color: #00d4ff;
        }

        .efficiency {
          color: #f59e0b;
        }

        /* Cost Analysis */
        .cost-analysis h4 {
          color: #22c55e;
          margin-bottom: 1rem;
          font-size: 1rem;
        }

        .cost-grid {
          display: flex;
          flex-direction: column;
          gap: 0.5rem;
        }

        .cost-item {
          display: flex;
          justify-content: space-between;
          font-size: 0.9rem;
        }

        .cost-item.savings {
          border-top: 1px solid rgba(34, 197, 94, 0.3);
          padding-top: 0.5rem;
          font-weight: 600;
        }

        .cost-label {
          color: #ccc;
        }

        .cost-value {
          color: #22c55e;
          font-weight: 600;
        }

        /* Examples Section */
        .examples-section h4 {
          color: #00d4ff;
          margin-bottom: 1rem;
          font-size: 1rem;
        }

        .example-grid {
          display: flex;
          flex-direction: column;
          gap: 0.8rem;
        }

        .example-item {
          display: flex;
          align-items: center;
          gap: 1rem;
          padding: 0.8rem;
          background: rgba(255, 255, 255, 0.03);
          border-radius: 8px;
          transition: background 0.3s ease;
        }

        .example-item:hover {
          background: rgba(255, 255, 255, 0.05);
        }

        .example-icon {
          font-size: 1.8rem;
          flex-shrink: 0;
        }

        .example-content {
          flex: 1;
        }

        .example-title {
          font-weight: 600;
          color: white;
          margin-bottom: 0.2rem;
        }

        .example-desc {
          font-size: 0.8rem;
          color: #999;
        }

        /* Responsive */
        @media (max-width: 768px) {
          .savings-grid {
            grid-template-columns: 1fr;
            gap: 0.8rem;
          }

          .timing-header {
            flex-direction: column;
            align-items: flex-start;
            gap: 0.5rem;
          }

          .savings-value {
            font-size: 1.4rem;
          }
        }
      `}</style>
    </div>
  );
};

export default TimingComparison;