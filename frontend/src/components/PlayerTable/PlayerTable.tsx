import React from 'react';
import { PlayerRankingItemDTO, RankingSort } from '../../api/players';
import { ScoreBar } from '../ScoreBar/ScoreBar';
import { translatePosition } from '../../utils/translations';
import { formatCredits } from '../../utils/format';
import './PlayerTable.css';

interface PlayerTableProps {
  players: PlayerRankingItemDTO[];
  sort: RankingSort;
  onSortChange: (sort: RankingSort) => void;
  onSelectPlayer: (id: number) => void;
}

export const PlayerTable: React.FC<PlayerTableProps> = ({
  players,
  sort,
  onSortChange,
  onSelectPlayer,
}) => {
  const toggleScoreSort = () => {
    onSortChange(sort === 'score_desc' ? 'score_asc' : 'score_desc');
  };

  const scoreIndicator = sort === 'score_desc' ? ' ↓' : sort === 'score_asc' ? ' ↑' : '';

  return (
    <div className="player-table-container">
      <table className="player-table">
        <thead className="player-table__head">
          <tr className="player-table__row player-table__row--header">
            <th className="player-table__cell player-table__cell--header">
              <button
                type="button"
                className="player-table__sort"
                onClick={() => onSortChange('name')}
              >
                Nombre del jugador{sort === 'name' ? ' ↑' : ''}
              </button>
            </th>
            <th className="player-table__cell player-table__cell--header">Club</th>
            <th className="player-table__cell player-table__cell--header">Liga</th>
            <th className="player-table__cell player-table__cell--header">Posición</th>
            <th className="player-table__cell player-table__cell--header">
              <button type="button" className="player-table__sort" onClick={toggleScoreSort}>
                Score{scoreIndicator}
              </button>
            </th>
            <th className="player-table__cell player-table__cell--header">Créditos</th>
          </tr>
        </thead>
        <tbody className="player-table__body">
          {players.map((player) => (
            <tr
              key={player.id}
              className="player-table__row player-table__row--interactive"
              onClick={() => onSelectPlayer(player.id)}
              tabIndex={0}
              role="button"
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  onSelectPlayer(player.id);
                }
              }}
              aria-label={`Ver perfil de ${player.name}`}
            >
              <td className="player-table__cell player-table__cell--name">{player.name}</td>
              <td className="player-table__cell">{player.club}</td>
              <td className="player-table__cell">{player.league}</td>
              <td className="player-table__cell">
                <span
                  className={`player-table__position player-table__position--${player.position.toLowerCase()}`}
                >
                  {translatePosition(player.position)}
                </span>
              </td>
              <td className="player-table__cell">
                <ScoreBar score={player.score} />
              </td>
              <td className="player-table__cell player-table__cell--credits">
                {player.price === null ? (
                  '—'
                ) : (
                  <span className="player-table__credits">{`${formatCredits(player.price)} Créditos`}</span>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};
