import React, { useEffect, useState } from 'react';
import { getPlayerById, PlayerDetailDTO } from '../../api/players';
import { PlayerCard } from '../../components/PlayerCard/PlayerCard';
import './PlayerDetailPage.css';

interface PlayerDetailPageProps {
  playerId: number;
  onBack: () => void;
}

export const PlayerDetailPage: React.FC<PlayerDetailPageProps> = ({ playerId, onBack }) => {
  const [player, setPlayer] = useState<PlayerDetailDTO | null>(null);
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
          setError('Player not found or unable to load details.');
          setIsLoading(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [playerId]);

  return (
    <main className="player-detail-page">
      <nav className="player-detail-page__nav">
        <button
          type="button"
          className="player-detail-page__back-button"
          onClick={onBack}
          aria-label="Back to player catalog"
        >
          ← Back to Catalog
        </button>
      </nav>

      <section className="player-detail-page__content">
        {isLoading && (
          <div className="player-detail-page__loading" role="status">
            <span className="player-detail-page__spinner" />
            <p>Loading player details...</p>
          </div>
        )}

        {error && !isLoading && (
          <div className="player-detail-page__error" role="alert">
            <h2 className="player-detail-page__error-title">Player Not Found</h2>
            <p className="player-detail-page__error-desc">{error}</p>
            <button
              type="button"
              className="player-detail-page__back-button player-detail-page__back-button--error"
              onClick={onBack}
            >
              Return to Catalog
            </button>
          </div>
        )}

        {!isLoading && !error && player && <PlayerCard player={player} />}
      </section>
    </main>
  );
};
