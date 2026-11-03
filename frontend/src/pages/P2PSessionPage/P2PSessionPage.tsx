import React, { useState, useEffect, useCallback, useRef } from 'react';
import { useAuth } from '../../context/AuthContext';
import { getP2PSession, acceptP2PSession, cancelP2PSession, P2PSessionDTO, P2PTradeResponseDTO } from '../../api/p2p';
import './P2PSessionPage.css';

interface P2PSessionPageProps {
  token: string;
  onBack: () => void;
  onOpenAuth: () => void;
  onGoToPortfolio?: () => void;
}

export const P2PSessionPage: React.FC<P2PSessionPageProps> = ({
  token,
  onBack,
  onOpenAuth,
  onGoToPortfolio,
}) => {
  const { user, isAuthenticated } = useAuth();
  const [session, setSession] = useState<P2PSessionDTO | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [tradeResult, setTradeResult] = useState<P2PTradeResponseDTO | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [secondsRemaining, setSecondsRemaining] = useState<number | null>(null);
  const intervalRef = useRef<number | null>(null);

  const loadSession = useCallback(async () => {
    setIsLoading(true);
    setErrorMessage(null);
    try {
      const data = await getP2PSession(token);
      setSession(data);
    } catch (err: unknown) {
      const axiosErr = err as { response?: { status?: number; data?: { error?: string } } };
      if (axiosErr.response?.status === 404) {
        setErrorMessage('La sesión P2P no existe o el enlace es inválido.');
      } else if (axiosErr.response?.status === 410) {
        setErrorMessage('Esta oferta ya venció.');
      } else {
        setErrorMessage(axiosErr.response?.data?.error || 'Error al cargar la oferta P2P.');
      }
    } finally {
      setIsLoading(false);
    }
  }, [token]);

  useEffect(() => {
    if (isAuthenticated) {
      loadSession();
    }
  }, [isAuthenticated, loadSession]);

  // Manejo del contador regresivo
  useEffect(() => {
    if (!session || session.status !== 'PENDING') {
      setSecondsRemaining(null);
      return;
    }

    const updateCountdown = () => {
      const expires = new Date(session.expiresAt).getTime();
      const now = Date.now();
      const diffSecs = Math.max(0, Math.floor((expires - now) / 1000));
      setSecondsRemaining(diffSecs);

      if (diffSecs === 0) {
        if (intervalRef.current) clearInterval(intervalRef.current);
      }
    };

    updateCountdown();
    intervalRef.current = window.setInterval(updateCountdown, 1000);

    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [session]);

  if (!isAuthenticated) {
    return (
      <div className="p2p-session">
        <div className="p2p-session__card p2p-session__card--center">
          <h2 className="p2p-session__title">Oferta Directa P2P</h2>
          <p className="p2p-session__text">
            Debes iniciar sesión con tu cuenta para ver los detalles de esta oferta personalizada y operar.
          </p>
          <div className="p2p-session__actions">
            <button className="p2p-session__btn p2p-session__btn--primary" onClick={onOpenAuth}>
              Iniciar Sesión
            </button>
            <button className="p2p-session__btn p2p-session__btn--secondary" onClick={onBack}>
              Ir al Inicio
            </button>
          </div>
        </div>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="p2p-session">
        <div className="p2p-session__card p2p-session__card--center">
          <p className="p2p-session__text">Cargando detalles de la oferta...</p>
        </div>
      </div>
    );
  }

  if (errorMessage && !session) {
    return (
      <div className="p2p-session">
        <div className="p2p-session__card p2p-session__card--center">
          <div className="p2p-session__badge p2p-session__badge--error">Error</div>
          <h2 className="p2p-session__title">No se pudo cargar la oferta</h2>
          <p className="p2p-session__text">{errorMessage}</p>
          <button className="p2p-session__btn p2p-session__btn--secondary" onClick={onBack}>
            Volver al Catálogo
          </button>
        </div>
      </div>
    );
  }

  if (!session) return null;

  const isSeller = user?.id === session.seller.id;
  const isExpired = session.status === 'EXPIRED' || (session.status === 'PENDING' && secondsRemaining === 0);

  const handleAccept = async () => {
    setIsSubmitting(true);
    setErrorMessage(null);
    try {
      const result = await acceptP2PSession(token);
      setTradeResult(result);
    } catch (err: unknown) {
      const axiosErr = err as { response?: { status?: number; data?: { error?: string } } };
      const status = axiosErr.response?.status;
      const apiErr = axiosErr.response?.data?.error;

      if (status === 409) {
        setErrorMessage(apiErr || 'La oferta ya fue tomada por otro usuario o ya no está disponible.');
        loadSession();
      } else if (status === 410) {
        setErrorMessage('La oferta venció y ya no puede ser aceptada.');
        loadSession();
      } else if (status === 422) {
        setErrorMessage(apiErr || 'Saldo insuficiente de créditos para aceptar esta transacción.');
      } else {
        setErrorMessage(apiErr || 'No se pudo completar la transacción.');
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleCancel = async () => {
    if (!window.confirm('¿Seguro que deseas cancelar esta oferta? Los tokens reservados serán liberados.')) {
      return;
    }
    setIsSubmitting(true);
    try {
      await cancelP2PSession(token);
      loadSession();
    } catch (err: unknown) {
      const axiosErr = err as { response?: { data?: { error?: string } } };
      setErrorMessage(axiosErr.response?.data?.error || 'Error al cancelar la sesión.');
    } finally {
      setIsSubmitting(false);
    }
  };

  const formatCountdown = (secs: number) => {
    const mins = Math.floor(secs / 60);
    const rem = secs % 60;
    return `${mins.toString().padStart(2, '0')}:${rem.toString().padStart(2, '0')}`;
  };

  // 1. Caso de Éxito al Aceptar
  if (tradeResult) {
    return (
      <div className="p2p-session">
        <div className="p2p-session__card p2p-session__card--success">
          <div className="p2p-session__badge p2p-session__badge--success">¡Transacción Completada!</div>
          <h2 className="p2p-session__title">Compra Exitosa</h2>
          <p className="p2p-session__text">
            Has adquirido <strong>{tradeResult.quantity} tokens</strong> del jugador por un total de{' '}
            <strong>{tradeResult.totalAmount} créditos</strong>.
          </p>
          <div className="p2p-session__grid">
            <div className="p2p-session__grid-item">
              <span>Operación ID:</span>
              <strong>#{tradeResult.tradeId}</strong>
            </div>
            <div className="p2p-session__grid-item">
              <span>Vendedor:</span>
              <strong>{session.seller.email}</strong>
            </div>
            <div className="p2p-session__grid-item">
              <span>Comprador:</span>
              <strong>{user?.email}</strong>
            </div>
            <div className="p2p-session__grid-item">
              <span>Fecha:</span>
              <strong>{new Date(tradeResult.executedAt).toLocaleString()}</strong>
            </div>
          </div>
          <div className="p2p-session__actions">
            {onGoToPortfolio && (
              <button className="p2p-session__btn p2p-session__btn--primary" onClick={onGoToPortfolio}>
                Ver Mi Portfolio
              </button>
            )}
            <button className="p2p-session__btn p2p-session__btn--secondary" onClick={onBack}>
              Volver al Catálogo
            </button>
          </div>
        </div>
      </div>
    );
  }

  // 2. Caso Oferta Vencida
  if (isExpired) {
    return (
      <div className="p2p-session">
        <div className="p2p-session__card p2p-session__card--expired">
          <div className="p2p-session__badge p2p-session__badge--warning">Tiempo Agotado</div>
          <h2 className="p2p-session__title">Oferta Vencida</h2>
          <p className="p2p-session__text">
            Esta oferta personalizada superó el límite de <strong>5 minutos</strong> y ha caducado
            automáticamente. Los tokens reservados han sido liberados.
          </p>
          <div className="p2p-session__actions">
            <button className="p2p-session__btn p2p-session__btn--secondary" onClick={onBack}>
              Volver al Inicio
            </button>
          </div>
        </div>
      </div>
    );
  }

  // 3. Caso Oferta Ya Tomada por Otro Usuario
  if (session.status === 'COMPLETED') {
    return (
      <div className="p2p-session">
        <div className="p2p-session__card p2p-session__card--completed">
          <div className="p2p-session__badge p2p-session__badge--info">Finalizada</div>
          <h2 className="p2p-session__title">Oferta Ya Aceptada</h2>
          <p className="p2p-session__text">
            Esta transacción ya fue completada por otro usuario y no admite más compras.
          </p>
          <div className="p2p-session__actions">
            <button className="p2p-session__btn p2p-session__btn--secondary" onClick={onBack}>
              Volver al Catálogo
            </button>
          </div>
        </div>
      </div>
    );
  }

  // 4. Caso Oferta Cancelada por el Vendedor
  if (session.status === 'CANCELLED') {
    return (
      <div className="p2p-session">
        <div className="p2p-session__card p2p-session__card--cancelled">
          <div className="p2p-session__badge p2p-session__badge--error">Cancelada</div>
          <h2 className="p2p-session__title">Oferta Cancelada</h2>
          <p className="p2p-session__text">
            El vendedor ha cancelado esta oferta directa de tokens.
          </p>
          <div className="p2p-session__actions">
            <button className="p2p-session__btn p2p-session__btn--secondary" onClick={onBack}>
              Volver al Catálogo
            </button>
          </div>
        </div>
      </div>
    );
  }

  // 5. Caso Principal: Oferta PENDING Activa
  return (
    <div className="p2p-session">
      <div className="p2p-session__card">
        <div className="p2p-session__header">
          <div>
            <span className="p2p-session__badge p2p-session__badge--live">Oferta P2P Activa</span>
            <h1 className="p2p-session__title">Intercambio Directo de Tokens</h1>
          </div>
          {secondsRemaining !== null && (
            <div className={`p2p-session__timer ${secondsRemaining < 60 ? 'p2p-session__timer--urgent' : ''}`}>
              <span className="p2p-session__timer-label">Tiempo restante</span>
              <span className="p2p-session__timer-value">{formatCountdown(secondsRemaining)}</span>
            </div>
          )}
        </div>

        {errorMessage && (
          <div className="p2p-session__alert p2p-session__alert--error">
            {errorMessage}
          </div>
        )}

        <div className="p2p-session__participants">
          <div className="p2p-session__party">
            <span className="p2p-session__party-label">Vendedor (quien ofrece)</span>
            <span className="p2p-session__party-email">{session.seller.email}</span>
            {isSeller && <span className="p2p-session__tag">Tú</span>}
          </div>
          <div className="p2p-session__party-arrow">&rarr;</div>
          <div className="p2p-session__party">
            <span className="p2p-session__party-label">Comprador (en sesión)</span>
            <span className="p2p-session__party-email">{user?.email}</span>
            {!isSeller && <span className="p2p-session__tag">Tú</span>}
          </div>
        </div>

        <div className="p2p-session__offer-box">
          <div className="p2p-session__grid-item">
            <span>Jugador:</span>
            <strong>{session.player.name} ({session.player.clubName})</strong>
          </div>
          <div className="p2p-session__grid-item">
            <span>Cantidad a transferir:</span>
            <strong>{session.quantity} tokens</strong>
          </div>
          <div className="p2p-session__grid-item">
            <span>Precio por token:</span>
            <strong>{session.pricePerToken} créditos</strong>
          </div>
          <div className="p2p-session__grid-item p2p-session__grid-item--highlight">
            <span>Total a pagar:</span>
            <strong className="p2p-session__total">{session.totalAmount} créditos</strong>
          </div>
        </div>

        <div className="p2p-session__actions">
          {isSeller ? (
            <>
              <button
                className="p2p-session__btn p2p-session__btn--danger"
                onClick={handleCancel}
                disabled={isSubmitting}
              >
                {isSubmitting ? 'Cancelando...' : 'Cancelar Mi Oferta'}
              </button>
              <button className="p2p-session__btn p2p-session__btn--secondary" onClick={onBack}>
                Volver
              </button>
            </>
          ) : (
            <>
              <button
                className="p2p-session__btn p2p-session__btn--primary"
                onClick={handleAccept}
                disabled={isSubmitting || secondsRemaining === 0}
              >
                {isSubmitting ? 'Procesando...' : `Aceptar y Comprar (${session.totalAmount} créditos)`}
              </button>
              <button
                className="p2p-session__btn p2p-session__btn--secondary"
                onClick={onBack}
                disabled={isSubmitting}
              >
                Rechazar y Salir
              </button>
            </>
          )}
        </div>
      </div>
    </div>
  );
};
