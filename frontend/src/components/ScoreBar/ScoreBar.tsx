import React from 'react';
import './ScoreBar.css';

interface ScoreBarProps {
  score: number | null;
}

export const ScoreBar: React.FC<ScoreBarProps> = ({ score }) => {
  if (score === null) {
    return <span className="score-bar score-bar--missing">N/R</span>;
  }

  const percentage = score * 100;
  const width = Math.max(0, Math.min(percentage, 100));

  return (
    <div className="score-bar" aria-label={`Score ${percentage.toFixed(2)} por ciento`}>
      <span className="score-bar__value">{percentage.toFixed(2)}%</span>
      <span className="score-bar__track" aria-hidden="true">
        <span className="score-bar__fill" style={{ width: `${width}%` }} />
      </span>
    </div>
  );
};
