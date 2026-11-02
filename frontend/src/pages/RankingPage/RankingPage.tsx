import React, { useEffect, useState } from 'react';
import { getPlayersRanking, PlayerRankingItemDTO, RankingSort } from '../../api/players';
import { ScoreBar } from '../../components/ScoreBar/ScoreBar';
import { Pagination } from '../../components/Pagination/Pagination';
import { translatePosition } from '../../utils/translations';
import './RankingPage.css';

const PAGE_SIZE = 25;
const CHUNK_SIZE = 100;
const PAGES_PER_CHUNK = CHUNK_SIZE / PAGE_SIZE;

interface RankingPageProps {
  onSelectPlayer: (id: number) => void;
}

export const RankingPage: React.FC<RankingPageProps> = ({ onSelectPlayer }) => {
  const [chunk, setChunk] = useState<PlayerRankingItemDTO[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [sort, setSort] = useState<RankingSort>('score_desc');
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const offset = Math.floor((page - 1) / PAGES_PER_CHUNK) * CHUNK_SIZE;

  useEffect(() => {
    let isMounted = true;
    setIsLoading(true);
    setError(null);

    getPlayersRanking({ sort, offset, limit: CHUNK_SIZE })
      .then((data) => {
        if (!isMounted) {
          return;
        }
        setChunk(data.items);
        setTotal(data.total);
        setIsLoading(false);
      })
      .catch(() => {
        if (!isMounted) {
          return;
        }
        setError('No se pudo cargar el ranking. Por favor, intenta de nuevo más tarde.');
        setIsLoading(false);
      });

    return () => {
      isMounted = false;
    };
  }, [sort, offset]);

  const pageWithinChunk = (page - 1) % PAGES_PER_CHUNK;
  const start = pageWithinChunk * PAGE_SIZE;
  const visible = chunk.slice(start, start + PAGE_SIZE);
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  const toggleSort = () => {
    setPage(1);
    setSort((prev) => (prev === 'score_desc' ? 'score_asc' : 'score_desc'));
  };

  return (
    <main className="ranking-page">
      <header className="ranking-page__header">
        <h1 className="ranking-page__title">Ranking de jugadores</h1>
        <p className="ranking-page__subtitle">Los jugadores más performantes según su score.</p>
      </header>

      {isLoading && (
        <div className="ranking-page__loading" role="status">
          <span className="ranking-page__spinner" />
          <p>Cargando ranking...</p>
        </div>
      )}

      {error && !isLoading && (
        <div className="ranking-page__error" role="alert">
          <p>{error}</p>
        </div>
      )}

      {!isLoading && !error && visible.length === 0 && (
        <div className="ranking-page__empty">
          <p>No hay jugadores con score disponible.</p>
        </div>
      )}

      {!isLoading && !error && visible.length > 0 && (
        <>
          <div className="ranking-page__table-container">
            <table className="ranking-page__table">
              <thead>
                <tr>
                  <th className="ranking-page__cell ranking-page__cell--header">Puesto</th>
                  <th className="ranking-page__cell ranking-page__cell--header">Jugador</th>
                  <th className="ranking-page__cell ranking-page__cell--header">Club / Liga</th>
                  <th className="ranking-page__cell ranking-page__cell--header">Posición</th>
                  <th className="ranking-page__cell ranking-page__cell--header">
                    <button type="button" className="ranking-page__sort" onClick={toggleSort}>
                      Score {sort === 'score_desc' ? '↓' : '↑'}
                    </button>
                  </th>
                </tr>
              </thead>
              <tbody>
                {visible.map((player, index) => (
                  <tr
                    key={player.id}
                    className="ranking-page__row"
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
                    <td className="ranking-page__cell ranking-page__cell--rank">
                      #{(page - 1) * PAGE_SIZE + index + 1}
                    </td>
                    <td className="ranking-page__cell ranking-page__cell--name">{player.name}</td>
                    <td className="ranking-page__cell">
                      {player.club} · {player.league}
                    </td>
                    <td className="ranking-page__cell">
                      <span
                        className={`ranking-page__position ranking-page__position--${player.position.toLowerCase()}`}
                      >
                        {translatePosition(player.position)}
                      </span>
                    </td>
                    <td className="ranking-page__cell">
                      <ScoreBar score={player.score} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <Pagination page={page} totalPages={totalPages} total={total} onPageChange={setPage} />
        </>
      )}
    </main>
  );
};
