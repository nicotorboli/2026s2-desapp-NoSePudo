import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
  getPlayerById,
  getPlayerScore,
  getPlayerScores,
  getPlayerStatistics,
  getPlayersRanking,
  PlayerDetailDTO,
  PlayerRankingItemDTO,
  PlayerScoreDTO,
  PlayerStatisticsDTO,
  RankingSort,
} from '../../../api/players';
import { PlayerTable } from '../../PlayerTable/PlayerTable';
import { Pagination } from '../../Pagination/Pagination';
import { PlayerCard } from '../../PlayerCard/PlayerCard';
import { PlayerScore } from '../../PlayerScore/PlayerScore';
import { PlayerStatistics } from '../../PlayerStatistics/PlayerStatistics';
import { StatusMessage } from '../AdminFeedback/AdminFeedback';
import './CatalogSection.css';

const PAGE_SIZE = 20;

export const CatalogSection: React.FC = () => {
  const [searchInput, setSearchInput] = useState('');
  const [search, setSearch] = useState('');
  const [includeInactive, setIncludeInactive] = useState(true);
  const [sort, setSort] = useState<RankingSort>('name');
  const [page, setPage] = useState(1);

  const [items, setItems] = useState<PlayerRankingItemDTO[]>([]);
  const [total, setTotal] = useState(0);
  const [isLoading, setIsLoading] = useState(false);
  const [listError, setListError] = useState<string | null>(null);

  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [detail, setDetail] = useState<PlayerDetailDTO | null>(null);
  const [score, setScore] = useState<PlayerScoreDTO | null>(null);
  const [history, setHistory] = useState<PlayerScoreDTO[]>([]);
  const [stats, setStats] = useState<PlayerStatisticsDTO[]>([]);
  const [detailLoading, setDetailLoading] = useState(false);
  const requestRef = useRef(0);

  const loadPlayers = useCallback(
    async (
      currentSearch: string,
      currentSort: RankingSort,
      currentPage: number,
      currentIncludeInactive: boolean,
    ) => {
      setIsLoading(true);
      setListError(null);
      try {
        const result = await getPlayersRanking({
          search: currentSearch || undefined,
          sort: currentSort,
          offset: (currentPage - 1) * PAGE_SIZE,
          limit: PAGE_SIZE,
          includeInactive: currentIncludeInactive,
        });
        setItems(result.items);
        setTotal(result.total);
      } catch {
        setListError('No se pudo cargar el catálogo de jugadores.');
      } finally {
        setIsLoading(false);
      }
    },
    [],
  );

  useEffect(() => {
    void loadPlayers(search, sort, page, includeInactive);
  }, [search, sort, page, includeInactive, loadPlayers]);

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setPage(1);
    setSearch(searchInput.trim());
  };

  const handleSelect = async (id: number) => {
    const requestId = requestRef.current + 1;
    requestRef.current = requestId;

    setSelectedId(id);
    setDetailLoading(true);
    setDetail(null);
    setScore(null);
    setHistory([]);
    setStats([]);

    const [detailResult, scoreResult, historyResult, statsResult] = await Promise.allSettled([
      getPlayerById(id),
      getPlayerScore(id),
      getPlayerScores(id),
      getPlayerStatistics(id),
    ]);

    if (requestRef.current !== requestId) {
      return;
    }

    setDetail(detailResult.status === 'fulfilled' ? detailResult.value : null);
    setScore(scoreResult.status === 'fulfilled' ? scoreResult.value : null);
    setHistory(historyResult.status === 'fulfilled' ? historyResult.value : []);
    setStats(statsResult.status === 'fulfilled' ? statsResult.value : []);
    setDetailLoading(false);
  };

  const handleSortChange = (nextSort: RankingSort) => {
    setSort(nextSort);
    setPage(1);
  };

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  return (
    <section className="catalog-section" id="admin-catalogo">
      <header className="catalog-section__header">
        <h2 className="catalog-section__title">Catálogo</h2>
        <p className="catalog-section__subtitle">
          Buscá jugadores y revisá su score, historial y últimas estadísticas.
        </p>
      </header>

      <form className="catalog-section__search" onSubmit={handleSubmit}>
        <label className="catalog-section__search-label" htmlFor="admin-catalog-search">
          Buscar por nombre
        </label>
        <div className="catalog-section__search-row">
          <input
            id="admin-catalog-search"
            className="catalog-section__input"
            type="text"
            placeholder="ej. Bukayo Saka"
            value={searchInput}
            onChange={(event) => setSearchInput(event.target.value)}
          />
          <label className="catalog-section__check">
            <input
              type="checkbox"
              checked={includeInactive}
              onChange={(event) => {
                setIncludeInactive(event.target.checked);
                setPage(1);
              }}
            />
            Incluir inactivos
          </label>
          <button type="submit" className="catalog-section__button">
            Buscar
          </button>
        </div>
      </form>

      {listError && <StatusMessage kind="error">{listError}</StatusMessage>}

      {isLoading ? (
        <p className="catalog-section__meta">Cargando catálogo...</p>
      ) : items.length === 0 ? (
        <p className="catalog-section__meta">No se encontraron jugadores.</p>
      ) : (
        <>
          <PlayerTable
            players={items}
            sort={sort}
            onSortChange={handleSortChange}
            onSelectPlayer={handleSelect}
          />
          <Pagination
            page={page}
            totalPages={totalPages}
            total={total}
            onPageChange={setPage}
          />
        </>
      )}

      {selectedId !== null && (
        <article className="catalog-section__detail">
          <header className="catalog-section__detail-header">
            <h3 className="catalog-section__detail-title">Detalle del jugador</h3>
            <button
              type="button"
              className="catalog-section__button"
              onClick={() => setSelectedId(null)}
            >
              Cerrar detalle
            </button>
          </header>

          {detailLoading ? (
            <p className="catalog-section__meta">Cargando jugador...</p>
          ) : detail ? (
            <div className="catalog-section__detail-content">
              <PlayerCard player={detail} />
              <PlayerScore score={score} history={history} />
              <PlayerStatistics stats={stats} />
            </div>
          ) : (
            <StatusMessage kind="error">No se pudo cargar el jugador seleccionado.</StatusMessage>
          )}
        </article>
      )}
    </section>
  );
};
