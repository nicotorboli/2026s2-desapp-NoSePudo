import React, { useState, useEffect, useCallback } from 'react';
import { useAuth } from '../../context/AuthContext';
import { getUserPortfolio, PortfolioPositionDTO } from '../../api/portfolio';
import { createP2PSession, P2PSessionDTO, CreateP2PSessionRequestDTO } from '../../api/p2p';
import './P2PCreatePage.css';

interface P2PCreatePageProps {
  onBack: () => void;
  onOpenSession: (token: string) => void;
  onOpenAuth: () => void;
}

export const P2PCreatePage: React.FC<P2PCreatePageProps> = ({
  onBack,
  onOpenSession,
  onOpenAuth,
}) => {
  const { user, isAuthenticated } = useAuth();
  const [positions, setPositions] = useState<PortfolioPositionDTO[]>([]);
  const [selectedPlayerId, setSelectedPlayerId] = useState<number | ''>('');
  const [quantity, setQuantity] = useState<number>(1);
  const [pricePerToken, setPricePerToken] = useState<number>(10000);
  const [isLoading, setIsLoading] = useState(false);
  const [createdSession, setCreatedSession] = useState<P2PSessionDTO | null>(null);
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadPositions = useCallback(async () => {
    if (!user) return;
    setIsLoading(true);
    try {
      const port = await getUserPortfolio(user.id);
      const available = (port.positions || []).filter((p) => (p.availableQuantity ?? p.quantity) > 0);
      setPositions(available);
      if (available.length > 0 && selectedPlayerId === '') {
        setSelectedPlayerId(available[0].playerId);
      }
    } catch {
      setError('Error al cargar los jugadores de tu portfolio.');
    } finally {
      setIsLoading(false);
    }
  }, [user, selectedPlayerId]);

  useEffect(() => {
    if (isAuthenticated && user) {
      loadPositions();
    }
  }, [isAuthenticated, user, loadPositions]);

  if (!isAuthenticated) {
    return (
      <div className="p2p-create">
        <div className="p2p-create__card">
          <h2 className="p2p-create__title">Transacción P2P de Tokens</h2>
          <p className="p2p-create__description">
            Debes iniciar sesión para crear una oferta directa de compra/venta.
          </p>
          <div className="p2p-create__actions">
            <button className="p2p-create__btn p2p-create__btn--primary" onClick={onOpenAuth}>
              Iniciar Sesión
            </button>
            <button className="p2p-create__btn p2p-create__btn--secondary" onClick={onBack}>
              Volver
            </button>
          </div>
        </div>
      </div>
    );
  }

  const selectedPosition = positions.find((p) => p.playerId === Number(selectedPlayerId));
  const maxAvailable = selectedPosition ? (selectedPosition.availableQuantity ?? selectedPosition.quantity) : 0;
  const totalAmount = quantity * pricePerToken;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedPlayerId || quantity <= 0 || pricePerToken <= 0) {
      setError('Por favor completa todos los campos con valores válidos.');
      return;
    }
    if (quantity > maxAvailable) {
      setError(`No tienes suficientes tokens disponibles. Máximo: ${maxAvailable}`);
      return;
    }

    setIsLoading(true);
    setError(null);
    try {
      const payload: CreateP2PSessionRequestDTO = {
        playerId: Number(selectedPlayerId),
        quantity,
        pricePerToken,
      };
      const session = await createP2PSession(payload);
      setCreatedSession(session);
    } catch (err: unknown) {
      const axiosErr = err as { response?: { data?: { error?: string } } };
      setError(axiosErr.response?.data?.error || 'No se pudo crear la oferta P2P.');
    } finally {
      setIsLoading(false);
    }
  };

  const getFullUrl = (token: string) => {
    const origin = window.location.origin;
    return `${origin}/?token=${token}`;
  };

  const handleCopyLink = () => {
    if (!createdSession) return;
    const link = getFullUrl(createdSession.token);
    navigator.clipboard.writeText(link);
    setCopied(true);
    setTimeout(() => setCopied(false), 3000);
  };

  return (
    <div className="p2p-create">
      <div className="p2p-create__header">
        <button className="p2p-create__back-btn" onClick={onBack}>
          &larr; Volver
        </button>
        <h1 className="p2p-create__heading">Nueva Oferta Directa P2P</h1>
      </div>

      {error && <div className="p2p-create__alert p2p-create__alert--error">{error}</div>}

      {createdSession ? (
        <div className="p2p-create__success-card">
          <div className="p2p-create__success-badge">Oferta Creada Exitosamente</div>
          <h2 className="p2p-create__success-title">Enlace de Transacción Generado</h2>
          <p className="p2p-create__success-info">
            Tus <strong>{createdSession.quantity} tokens</strong> han sido reservados en escrow.
            Comparte este enlace con el comprador. La oferta vence en <strong>5 minutos</strong>.
          </p>

          <div className="p2p-create__link-box">
            <input
              type="text"
              readOnly
              value={getFullUrl(createdSession.token)}
              className="p2p-create__link-input"
            />
            <button
              type="button"
              className="p2p-create__btn p2p-create__btn--primary"
              onClick={handleCopyLink}
            >
              {copied ? '¡Copiado!' : 'Copiar Enlace'}
            </button>
          </div>

          <div className="p2p-create__success-details">
            <div><strong>Jugador:</strong> {createdSession.player.name}</div>
            <div><strong>Cantidad:</strong> {createdSession.quantity} tokens</div>
            <div><strong>Precio por token:</strong> {createdSession.pricePerToken} créditos</div>
            <div><strong>Total a cobrar:</strong> {createdSession.totalAmount} créditos</div>
          </div>

          <div className="p2p-create__actions">
            <button
              className="p2p-create__btn p2p-create__btn--primary"
              onClick={() => onOpenSession(createdSession.token)}
            >
              Ver Pantalla de la Oferta
            </button>
            <button
              className="p2p-create__btn p2p-create__btn--secondary"
              onClick={() => {
                setCreatedSession(null);
                loadPositions();
              }}
            >
              Crear Otra Oferta
            </button>
          </div>
        </div>
      ) : (
        <form className="p2p-create__form" onSubmit={handleSubmit}>
          {positions.length === 0 ? (
            <div className="p2p-create__empty">
              <p>No tienes tokens disponibles en tu portfolio para vender en este momento.</p>
              <button
                type="button"
                className="p2p-create__btn p2p-create__btn--secondary"
                onClick={onBack}
              >
                Ir al Catálogo
              </button>
            </div>
          ) : (
            <>
              <div className="p2p-create__form-group">
                <label className="p2p-create__label" htmlFor="playerSelect">
                  Seleccionar Jugador a Vender:
                </label>
                <select
                  id="playerSelect"
                  className="p2p-create__select"
                  value={selectedPlayerId}
                  onChange={(e) => {
                    const id = Number(e.target.value);
                    setSelectedPlayerId(id);
                    setQuantity(1);
                  }}
                >
                  {positions.map((p) => (
                    <option key={p.playerId} value={p.playerId}>
                      {p.playerName} ({p.clubName}) — Disponibles: {p.availableQuantity ?? p.quantity} tokens
                    </option>
                  ))}
                </select>
              </div>

              <div className="p2p-create__form-group">
                <label className="p2p-create__label" htmlFor="quantityInput">
                  Cantidad de Tokens (Disponibles: {maxAvailable}):
                </label>
                <input
                  id="quantityInput"
                  type="number"
                  min="1"
                  max={maxAvailable}
                  className="p2p-create__input"
                  value={quantity}
                  onChange={(e) => setQuantity(Number(e.target.value))}
                  required
                />
              </div>

              <div className="p2p-create__form-group">
                <label className="p2p-create__label" htmlFor="priceInput">
                  Precio por Token (en créditos):
                </label>
                <input
                  id="priceInput"
                  type="number"
                  min="1"
                  className="p2p-create__input"
                  value={pricePerToken}
                  onChange={(e) => setPricePerToken(Number(e.target.value))}
                  required
                />
              </div>

              <div className="p2p-create__summary">
                <div className="p2p-create__summary-item">
                  <span>Tokens a vender:</span>
                  <strong>{quantity}</strong>
                </div>
                <div className="p2p-create__summary-item">
                  <span>Total a recibir:</span>
                  <strong className="p2p-create__total">{totalAmount} créditos</strong>
                </div>
              </div>

              <div className="p2p-create__actions">
                <button
                  type="submit"
                  className="p2p-create__btn p2p-create__btn--primary"
                  disabled={isLoading || maxAvailable === 0}
                >
                  {isLoading ? 'Generando Enlace...' : 'Crear Oferta y Generar Link'}
                </button>
                <button
                  type="button"
                  className="p2p-create__btn p2p-create__btn--secondary"
                  onClick={onBack}
                >
                  Cancelar
                </button>
              </div>
            </>
          )}
        </form>
      )}
    </div>
  );
};
