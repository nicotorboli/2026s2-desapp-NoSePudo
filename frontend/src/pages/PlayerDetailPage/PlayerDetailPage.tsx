import React, { useEffect, useState } from 'react';
import { useAuth } from '../../context/AuthContext';
import { getPlayerById, PlayerDetailDTO } from '../../api/players';
import { PlayerCard } from '../../components/PlayerCard/PlayerCard';
import { PlayerTrade } from '../../components/PlayerTrade/PlayerTrade';
import './PlayerDetailPage.css';

interface PlayerDetailPageProps {
  playerId: number;
  onBack: () => void;
  onOpenAuth?: (tab?: 'login' | 'register') => void;
}

export const PlayerDetailPage: React.FC<PlayerDetailPageProps> = ({
  playerId,
  onBack,
  onOpenAuth,
}) => {
  const { isAuthenticated } = useAuth();
  const [player, setPlayer] = useState<PlayerDetailDTO | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!isAuthenticated) {
      setPlayer(null);
      setIsLoading(false);
      setError(null);
      return;
    }

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
  }, [playerId, isAuthenticated]);

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
        {!isAuthenticated && (
          <div className="player-detail-page__unauth">
            <h2 className="player-detail-page__unauth-title">Iniciá sesión para ver la ficha</h2>
            <p className="player-detail-page__unauth-desc">
              Para ver el detalle completo de este jugador y operar tokens en el mercado, necesitás iniciar sesión con tu cuenta o registrarte en segundos.
            </p>
            {onOpenAuth && (
              <div className="player-detail-page__unauth-actions">
                <button
                  type="button"
                  className="player-detail-page__auth-button"
                  onClick={() => onOpenAuth('login')}
                  aria-label="Iniciar Sesión / Registrarse"
                >
                  Iniciar Sesión / Registrarse
                </button>
                <button
                  type="button"
                  className="player-detail-page__auth-button player-detail-page__auth-button--register"
                  onClick={() => onOpenAuth('register')}
                >
                  Crear Cuenta (1,000 CR Gratis)
                </button>
              </div>
            )}
          </div>
        )}

        {isAuthenticated && isLoading && (
          <div className="player-detail-page__loading" role="status">
            <span className="player-detail-page__spinner" />
            <p>Cargando detalles del jugador...</p>
          </div>
        )}

        {isAuthenticated && error && !isLoading && (
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

        {isAuthenticated && !isLoading && !error && player && (
          <div className="player-detail-page__grid">
            <div className="player-detail-page__card-col">
              <PlayerCard player={player} />
            </div>
            <div className="player-detail-page__trade-col">
              <PlayerTrade
                playerId={player.id}
                playerName={player.name}
                onOpenAuth={onOpenAuth}
              />
            </div>
          </div>
        )}
      </section>
    </main>
  );
};
