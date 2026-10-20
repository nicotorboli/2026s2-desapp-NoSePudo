import React, { useEffect, useState } from 'react';
import {
  getPlayerById,
  getPlayerStatistics,
  PlayerDetailDTO,
  PlayerStatisticsDTO,
} from '../../api/players';
import { PlayerCard } from '../../components/PlayerCard/PlayerCard';
import { PlayerStatistics } from '../../components/PlayerStatistics/PlayerStatistics';
import './PlayerDetailPage.css';

interface PlayerDetailPageProps {
  playerId: number;
  onBack: () => void;
}

export const PlayerDetailPage: React.FC<PlayerDetailPageProps> = ({ playerId, onBack }) => {
  const [player, setPlayer] = useState<PlayerDetailDTO | null>(null);
  const [stats, setStats] = useState<PlayerStatisticsDTO[] | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let isMounted = true;
    setIsLoading(true);
    setError(null);

    getPlayerById(playerId)
      .then((data) => {
        if (isMounted) {
          setPlayer(data);
          setIsLoading(false);
        }
      })
      .catch(() => {
        if (isMounted) {
          setError('Jugador no encontrado o no se pudieron cargar los detalles.');
          setIsLoading(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [playerId]);

  useEffect(() => {
    if (!player) {
      return;
    }
    let isMounted = true;

    getPlayerStatistics(playerId)
      .then((data) => {
        if (isMounted) {
          setStats(data);
        }
      })
      .catch(() => {
        if (isMounted) {
          setStats([]);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [player, playerId]);

  return (
    <main className="player-detail-page">
      <nav className="player-detail-page__nav">
        <button
          type="button"
          className="player-detail-page__back-button"
          onClick={onBack}
          aria-label="Volver al catálogo de jugadores"
        >
          ← Volver al catálogo
        </button>
      </nav>

      <section className="player-detail-page__content">
        {isLoading && (
          <div className="player-detail-page__loading" role="status">
            <span className="player-detail-page__spinner" />
            <p>Cargando detalles del jugador...</p>
          </div>
        )}

        {error && !isLoading && (
          <div className="player-detail-page__error" role="alert">
            <h2 className="player-detail-page__error-title">Jugador no encontrado</h2>
            <p className="player-detail-page__error-desc">{error}</p>
            <button
              type="button"
              className="player-detail-page__back-button player-detail-page__back-button--error"
              onClick={onBack}
            >
              Volver al catálogo
            </button>
          </div>
        )}

        {!isLoading && !error && player && (
          <>
            <PlayerCard player={player} />
            {stats !== null && <PlayerStatistics stats={stats} />}
          </>
        )}
      </section>
    </main>
  );
};
