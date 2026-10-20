import React, { useState } from 'react';
import { PlayerStatisticsDTO } from '../../api/players';
import './PlayerStatistics.css';

interface PlayerStatisticsProps {
  stats: PlayerStatisticsDTO[];
}

const formatDate = (dateStr: string): string => {
  try {
    const d = new Date(dateStr);
    return d.toLocaleString('es-ES');
  } catch {
    return dateStr;
  }
};

const formatValue = (value: number): string => {
  return value.toFixed(2);
};

export const PlayerStatistics: React.FC<PlayerStatisticsProps> = ({ stats }) => {
  const [activeTab, setActiveTab] = useState<'current' | 'history'>('current');

  if (stats.length === 0) {
    return (
      <section className="player-stats player-stats--empty" aria-label="Estadísticas del jugador">
        <h2 className="player-stats__title">Estadísticas</h2>
        <p className="player-stats__empty">Sin estadísticas registradas</p>
      </section>
    );
  }

  const latest = stats[0];
  const hasHistory = stats.length > 1;
  const showCurrent = activeTab === 'current' || !hasHistory;

  return (
    <section className="player-stats" aria-label="Estadísticas del jugador">
      <h2 className="player-stats__title">Estadísticas</h2>

      {hasHistory && (
        <div className="player-stats__tabs" role="tablist" aria-label="Vista de estadísticas">
          <button
            type="button"
            role="tab"
            aria-selected={showCurrent}
            className={`player-stats__tab ${showCurrent ? 'player-stats__tab--active' : ''}`}
            onClick={() => setActiveTab('current')}
          >
            Actual
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={!showCurrent}
            className={`player-stats__tab ${showCurrent ? '' : 'player-stats__tab--active'}`}
            onClick={() => setActiveTab('history')}
          >
            Historial
          </button>
        </div>
      )}

      {showCurrent ? (
        <>
          <div className="player-stats__grid">
            <div className="player-stats__tile">
              <span className="player-stats__tile-label">Goles</span>
              <span className="player-stats__tile-value">{latest.goals}</span>
            </div>
            <div className="player-stats__tile">
              <span className="player-stats__tile-label">Asistencias</span>
              <span className="player-stats__tile-value">{latest.assists}</span>
            </div>
            <div className="player-stats__tile">
              <span className="player-stats__tile-label">Remates</span>
              <span className="player-stats__tile-value">{formatValue(latest.shots)}</span>
            </div>
            <div className="player-stats__tile">
              <span className="player-stats__tile-label">Pases clave</span>
              <span className="player-stats__tile-value">{formatValue(latest.keyPasses)}</span>
            </div>
            <div className="player-stats__tile">
              <span className="player-stats__tile-label">Regates</span>
              <span className="player-stats__tile-value">{formatValue(latest.dribbles)}</span>
            </div>
            <div className="player-stats__tile">
              <span className="player-stats__tile-label">Entradas</span>
              <span className="player-stats__tile-value">{formatValue(latest.tackles)}</span>
            </div>
            <div className="player-stats__tile">
              <span className="player-stats__tile-label">Intercepciones</span>
              <span className="player-stats__tile-value">{formatValue(latest.interceptions)}</span>
            </div>
            <div className="player-stats__tile">
              <span className="player-stats__tile-label">Faltas</span>
              <span className="player-stats__tile-value">{formatValue(latest.fouls)}</span>
            </div>
            <div className="player-stats__tile">
              <span className="player-stats__tile-label">Despejes</span>
              <span className="player-stats__tile-value">{formatValue(latest.clears)}</span>
            </div>
            <div className="player-stats__tile">
              <span className="player-stats__tile-label">Bloqueos</span>
              <span className="player-stats__tile-value">{formatValue(latest.blocks)}</span>
            </div>
            <div className="player-stats__tile">
              <span className="player-stats__tile-label">Gambeteado</span>
              <span className="player-stats__tile-value">{formatValue(latest.wasDribbled)}</span>
            </div>
            <div className="player-stats__tile player-stats__tile--rating">
              <span className="player-stats__tile-label">Rating</span>
              <span className="player-stats__tile-value">{latest.rating.toFixed(2)}</span>
            </div>
          </div>
          <p className="player-stats__updated">Actualizado: {formatDate(latest.createdAt)}</p>
        </>
      ) : (
        <table className="player-stats__history">
          <thead>
            <tr>
              <th scope="col">Fecha</th>
              <th scope="col">Goles</th>
              <th scope="col">Asistencias</th>
              <th scope="col">Remates</th>
              <th scope="col">Pases clave</th>
              <th scope="col">Regates</th>
              <th scope="col">Entradas</th>
              <th scope="col">Intercepciones</th>
              <th scope="col">Faltas</th>
              <th scope="col">Despejes</th>
              <th scope="col">Bloqueos</th>
              <th scope="col">Gambeteado</th>
              <th scope="col">Rating</th>
            </tr>
          </thead>
          <tbody>
            {stats.slice(1).map((snapshot) => (
              <tr key={snapshot.createdAt}>
                <td>{formatDate(snapshot.createdAt)}</td>
                <td>{snapshot.goals}</td>
                <td>{snapshot.assists}</td>
                <td>{formatValue(snapshot.shots)}</td>
                <td>{formatValue(snapshot.keyPasses)}</td>
                <td>{formatValue(snapshot.dribbles)}</td>
                <td>{formatValue(snapshot.tackles)}</td>
                <td>{formatValue(snapshot.interceptions)}</td>
                <td>{formatValue(snapshot.fouls)}</td>
                <td>{formatValue(snapshot.clears)}</td>
                <td>{formatValue(snapshot.blocks)}</td>
                <td>{formatValue(snapshot.wasDribbled)}</td>
                <td>{snapshot.rating.toFixed(2)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  );
};
