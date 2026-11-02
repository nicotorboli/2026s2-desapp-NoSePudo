import React from 'react';
import { PlayerScoreDTO } from '../../api/players';
import { ScoreBar } from '../ScoreBar/ScoreBar';
import './PlayerScore.css';

interface PlayerScoreProps {
  score: PlayerScoreDTO | null;
  history: PlayerScoreDTO[];
}

export const PlayerScore: React.FC<PlayerScoreProps> = ({ score, history }) => {
  return (
    <section className="player-score" aria-label="Score del jugador">
      <h2 className="player-score__title">Score</h2>

      {score === null ? (
        <p className="player-score__empty">Sin score calculado (N/R)</p>
      ) : (
        <>
          <div className="player-score__current">
            <ScoreBar score={score.score} />
            <dl className="player-score__breakdown">
              <div className="player-score__item">
                <dt className="player-score__label">Percentil</dt>
                <dd className="player-score__value">{(score.percentile * 100).toFixed(2)}%</dd>
              </div>
              <div className="player-score__item">
                <dt className="player-score__label">Coeficiente de liga</dt>
                <dd className="player-score__value">{score.leagueCoefficient.toFixed(2)}</dd>
              </div>
              <div className="player-score__item">
                <dt className="player-score__label">Estrategia</dt>
                <dd className="player-score__value">{score.strategyName || '—'}</dd>
              </div>
            </dl>
          </div>

          {history.length > 1 && (
            <table className="player-score__history">
              <thead>
                <tr>
                  <th scope="col">Fecha</th>
                  <th scope="col">Score</th>
                  <th scope="col">Percentil</th>
                </tr>
              </thead>
              <tbody>
                {history.map((entry) => (
                  <tr key={entry.computedAt}>
                    <td>{new Date(entry.computedAt).toLocaleString('es-ES')}</td>
                    <td>
                      <ScoreBar score={entry.score} />
                    </td>
                    <td>{(entry.percentile * 100).toFixed(2)}%</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </>
      )}
    </section>
  );
};
