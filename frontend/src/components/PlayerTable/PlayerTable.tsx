import React from 'react';
import { PlayerListItemDTO } from '../../api/players';
import { translatePosition } from '../../utils/translations';
import './PlayerTable.css';

interface PlayerTableProps {
  players: PlayerListItemDTO[];
  onSelectPlayer: (id: number) => void;
}

export const PlayerTable: React.FC<PlayerTableProps> = ({ players, onSelectPlayer }) => {
  return (
    <div className="player-table-container">
      <table className="player-table">
        <thead className="player-table__head">
          <tr className="player-table__row player-table__row--header">
            <th className="player-table__cell player-table__cell--header">Nombre del jugador</th>
            <th className="player-table__cell player-table__cell--header">Club</th>
            <th className="player-table__cell player-table__cell--header">Liga</th>
            <th className="player-table__cell player-table__cell--header">Posición</th>
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
                <span className={`player-table__position player-table__position--${player.position.toLowerCase()}`}>
                  {translatePosition(player.position)}
                </span>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};
