import React, { useEffect, useState, useCallback } from 'react';
import { getPlayers, PaginatedPlayersDTO, PlayerFilterParams } from '../../api/players';
import { PlayerFilter } from '../../components/PlayerFilter/PlayerFilter';
import { PlayerTable } from '../../components/PlayerTable/PlayerTable';
import { Pagination } from '../../components/Pagination/Pagination';
import './PlayerListPage.css';

interface PlayerListPageProps {
  onSelectPlayer: (id: number) => void;
}

const defaultFilters: PlayerFilterParams = {
  page: 1,
  limit: 20,
  league: '',
  club: '',
  position: '',
  search: '',
  includeInactive: false,
};

export const PlayerListPage: React.FC<PlayerListPageProps> = ({ onSelectPlayer }) => {
  const [data, setData] = useState<PaginatedPlayersDTO | null>(null);
  const [filters, setFilters] = useState<PlayerFilterParams>(defaultFilters);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  const fetchPlayers = useCallback(async (currentFilters: PlayerFilterParams) => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await getPlayers(currentFilters);
      setData(result);
    } catch {
      setError('No se pudieron cargar los jugadores. Por favor, intenta de nuevo más tarde.');
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchPlayers(filters);
  }, [filters, fetchPlayers]);

  const handleFilterChange = (newFilters: PlayerFilterParams) => {
    setFilters(newFilters);
  };

  const handleResetFilters = () => {
    setFilters(defaultFilters);
  };

  const handlePageChange = (newPage: number) => {
    setFilters((prev) => ({
      ...prev,
      page: newPage,
    }));
  };

  return (
    <main className="player-list-page">
      <header className="player-list-page__header">
        <h1 className="player-list-page__title">Catálogo de Jugadores de Fútbol Europeo</h1>
        <p className="player-list-page__subtitle">
          Explora perfiles de jugadores de la Premier League, Bundesliga, La Liga, Serie A y Ligue 1.
        </p>
      </header>

      <PlayerFilter
        filters={filters}
        onChange={handleFilterChange}
        onReset={handleResetFilters}
      />

      <section className="player-list-page__content">
        {isLoading && (
          <div className="player-list-page__loading" role="status">
            <span className="player-list-page__spinner" />
            <p>Cargando catálogo de jugadores...</p>
          </div>
        )}

        {error && !isLoading && (
          <div className="player-list-page__error" role="alert">
            <p>{error}</p>
            <button
              type="button"
              className="player-list-page__retry-button"
              onClick={() => fetchPlayers(filters)}
            >
              Reintentar
            </button>
          </div>
        )}

        {!isLoading && !error && data && data.items.length === 0 && (
          <div className="player-list-page__empty">
            <p className="player-list-page__empty-title">No se encontraron jugadores</p>
            <p className="player-list-page__empty-desc">
              Intenta ajustar los filtros de búsqueda o restablecerlos para ver más jugadores.
            </p>
            <button
              type="button"
              className="player-list-page__reset-button"
              onClick={handleResetFilters}
            >
              Restablecer filtros
            </button>
          </div>
        )}

        {!isLoading && !error && data && data.items.length > 0 && (
          <>
            <PlayerTable players={data.items} onSelectPlayer={onSelectPlayer} />
            <Pagination
              page={data.page}
              totalPages={data.totalPages}
              total={data.total}
              onPageChange={handlePageChange}
            />
          </>
        )}
      </section>
    </main>
  );
};
