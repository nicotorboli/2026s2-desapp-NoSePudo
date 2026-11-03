import React, { useEffect, useState, useCallback } from 'react';
import {
  getPlayersRanking,
  PlayerFilterParams,
  PlayerRankingItemDTO,
  RankingSort,
} from '../../api/players';
import { PlayerFilter } from '../../components/PlayerFilter/PlayerFilter';
import { PlayerTable } from '../../components/PlayerTable/PlayerTable';
import { Pagination } from '../../components/Pagination/Pagination';
import './PlayerListPage.css';

const PAGE_SIZE = 25;

interface PlayerListPageProps {
  onSelectPlayer: (id: number) => void;
}

const defaultFilters: PlayerFilterParams = {
  page: 1,
  limit: PAGE_SIZE,
  league: '',
  club: '',
  position: '',
  search: '',
  includeInactive: false,
};

export const PlayerListPage: React.FC<PlayerListPageProps> = ({ onSelectPlayer }) => {
  const [items, setItems] = useState<PlayerRankingItemDTO[]>([]);
  const [total, setTotal] = useState<number>(0);
  const [filters, setFilters] = useState<PlayerFilterParams>(defaultFilters);
  const [sort, setSort] = useState<RankingSort>('name');
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  const fetchPlayers = useCallback(
    async (currentFilters: PlayerFilterParams, currentSort: RankingSort) => {
      setIsLoading(true);
      setError(null);
      try {
        const page = currentFilters.page ?? 1;
        const result = await getPlayersRanking({
          search: currentFilters.search,
          league: currentFilters.league,
          club: currentFilters.club,
          position: currentFilters.position,
          includeInactive: currentFilters.includeInactive,
          sort: currentSort,
          offset: (page - 1) * PAGE_SIZE,
          limit: PAGE_SIZE,
        });
        setItems(result.items);
        setTotal(result.total);
      } catch {
        setError('No se pudieron cargar los jugadores. Por favor, intenta de nuevo más tarde.');
      } finally {
        setIsLoading(false);
      }
    },
    []
  );

  useEffect(() => {
    fetchPlayers(filters, sort);
  }, [filters, sort, fetchPlayers]);

  const handleFilterChange = (newFilters: PlayerFilterParams) => {
    setFilters(newFilters);
  };

  const handleResetFilters = () => {
    setFilters(defaultFilters);
    setSort('name');
  };

  const handlePageChange = (newPage: number) => {
    setFilters((prev) => ({
      ...prev,
      page: newPage,
    }));
  };

  const handleSortChange = (newSort: RankingSort) => {
    setSort(newSort);
    setFilters((prev) => ({
      ...prev,
      page: 1,
    }));
  };

  const page = filters.page ?? 1;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

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
              onClick={() => fetchPlayers(filters, sort)}
            >
              Reintentar
            </button>
          </div>
        )}

        {!isLoading && !error && items.length === 0 && (
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

        {!isLoading && !error && items.length > 0 && (
          <>
            <PlayerTable
              players={items}
              sort={sort}
              onSortChange={handleSortChange}
              onSelectPlayer={onSelectPlayer}
            />
            <Pagination
              page={page}
              totalPages={totalPages}
              total={total}
              onPageChange={handlePageChange}
            />
          </>
        )}
      </section>
    </main>
  );
};
