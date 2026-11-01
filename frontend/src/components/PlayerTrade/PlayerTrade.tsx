import React, { useState, useEffect, useCallback } from 'react';
import { useAuth } from '../../context/AuthContext';
import {
  placeBuyOrder,
  placeSellOrder,
  getOpenOrders,
  PlaceOrderRequestDTO,
  OpenOrderDTO,
} from '../../api/orders';
import { getUserPortfolio, PortfolioResponseDTO } from '../../api/portfolio';
import { parseCredits, formatCredits } from '../../utils/format';
import './PlayerTrade.css';

interface PlayerTradeProps {
  playerId: number;
  playerName: string;
  onOpenAuth?: () => void;
  onTradeExecuted?: () => void;
}

const MAX_PLAYER_SUPPLY = 100;

export const PlayerTrade: React.FC<PlayerTradeProps> = ({
  playerId,
  playerName,
  onOpenAuth,
  onTradeExecuted,
}) => {
  const { user, isAuthenticated } = useAuth();
  const [side, setSide] = useState<'BUY' | 'SELL'>('BUY');
  const [quantity, setQuantity] = useState<string>('1');
  const [orderType, setOrderType] = useState<'MARKET' | 'LIMIT'>('MARKET');
  const [limitPrice, setLimitPrice] = useState<string>('1.0000');
  const [portfolio, setPortfolio] = useState<PortfolioResponseDTO | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [feedback, setFeedback] = useState<{ type: 'error' | 'success'; message: string } | null>(null);
  const [showOrderBook, setShowOrderBook] = useState(false);
  const [orderBook, setOrderBook] = useState<OpenOrderDTO[]>([]);
  const [isLoadingBook, setIsLoadingBook] = useState(false);

  const loadPortfolio = useCallback(async () => {
    if (!user) return;
    try {
      const data = await getUserPortfolio(user.id);
      setPortfolio(data);
    } catch {
    }
  }, [user]);

  const loadOrderBook = useCallback(async () => {
    try {
      setIsLoadingBook(true);
      const data = await getOpenOrders(playerId, true);
      setOrderBook(data);
    } catch {
    } finally {
      setIsLoadingBook(false);
    }
  }, [playerId]);

  useEffect(() => {
    if (isAuthenticated && user) {
      loadPortfolio();
    }
  }, [isAuthenticated, user, loadPortfolio]);

  useEffect(() => {
    if (showOrderBook) {
      loadOrderBook();
    }
  }, [showOrderBook, loadOrderBook]);

  if (!isAuthenticated) {
    return (
      <div className="player-trade">
        <div className="player-trade__header">
          <h3 className="player-trade__title">Mercado de Tokens</h3>
        </div>
        <div className="player-trade__unauth">
          <p className="player-trade__unauth-text">
            Iniciá sesión o registrate para comprar y vender tokens de <strong>{playerName}</strong>.
          </p>
          {onOpenAuth && (
            <button
              type="button"
              className="player-trade__unauth-btn"
              onClick={onOpenAuth}
            >
              Iniciar Sesión / Registrarse
            </button>
          )}
        </div>
      </div>
    );
  }

  const currentPosition = portfolio?.positions?.find((p) => p.playerId === playerId);
  const totalOwnedTokens = currentPosition?.quantity || 0;
  const availableTokens = currentPosition?.availableQuantity !== undefined
    ? currentPosition.availableQuantity
    : totalOwnedTokens;
  const reservedTokens = currentPosition?.reservedQuantity || 0;
  const availableCash = parseCredits(portfolio?.availableCash);
  const activeQuote = parseCredits(currentPosition?.currentPrice) || 1.0;

  const parsedQty = parseInt(quantity, 10) || 0;
  const parsedPrice = orderType === 'LIMIT' ? parseFloat(limitPrice) || 0 : activeQuote;
  const estimatedTotal = parsedQty * parsedPrice;

  const maxAffordable = Math.min(
    MAX_PLAYER_SUPPLY,
    Math.floor(availableCash / (parsedPrice > 0 ? parsedPrice : 1))
  );

  const handleQuantityChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const raw = e.target.value;
    if (raw === '') {
      setQuantity('');
      return;
    }
    if (!/^\d+$/.test(raw)) return;

    const num = parseInt(raw, 10);
    const maxAllowed = side === 'BUY' ? MAX_PLAYER_SUPPLY : availableTokens;

    if (num > maxAllowed) {
      setQuantity(maxAllowed.toString());
    } else {
      setQuantity(num.toString());
    }
  };

  const handlePriceChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const raw = e.target.value.replace(',', '.');
    if (raw === '') {
      setLimitPrice('');
      return;
    }
    if (/^\d*(\.\d{0,4})?$/.test(raw)) {
      setLimitPrice(raw);
    }
  };

  const handleTabChange = (newSide: 'BUY' | 'SELL') => {
    setSide(newSide);
    setFeedback(null);
    if (newSide === 'BUY') {
      setQuantity('1');
    } else {
      setQuantity(availableTokens > 0 ? '1' : '0');
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setFeedback(null);

    if (parsedQty <= 0) {
      setFeedback({ type: 'error', message: 'La cantidad de tokens debe ser mayor a 0.' });
      return;
    }

    if (parsedQty > MAX_PLAYER_SUPPLY) {
      setFeedback({
        type: 'error',
        message: `El suministro total emitido de cada jugador es de ${MAX_PLAYER_SUPPLY} tokens. No es posible operar una cantidad mayor.`,
      });
      return;
    }

    if (side === 'BUY' && estimatedTotal > availableCash) {
      setFeedback({
        type: 'error',
        message: `Saldo insuficiente. Tenés ${formatCredits(availableCash, 4)} Créditos y la orden requiere ${formatCredits(estimatedTotal, 4)} Créditos.`,
      });
      return;
    }

    if (side === 'SELL' && parsedQty > availableTokens) {
      setFeedback({
        type: 'error',
        message: reservedTokens > 0
          ? `Tenés ${totalOwnedTokens} tokens en cartera pero ${reservedTokens} están reservados en órdenes abiertas. Solo podés vender hasta ${availableTokens} tokens.`
          : `No poseés suficientes tokens disponibles para vender. Tenés ${availableTokens} y querés vender ${parsedQty}.`,
      });
      return;
    }

    const payload: PlaceOrderRequestDTO = {
      playerId,
      quantity: parsedQty,
    };

    if (orderType === 'LIMIT') {
      if (parsedPrice <= 0) {
        setFeedback({ type: 'error', message: 'El precio límite debe ser mayor a 0.' });
        return;
      }
      payload.price = parsedPrice.toFixed(4);
      payload.timeInForce = 'GTC';
    } else {
      payload.timeInForce = 'FOK';
    }

    setIsSubmitting(true);
    try {
      if (side === 'BUY') {
        const res = await placeBuyOrder(payload);
        let msg = '';
        if (res.status === 'FILLED') {
          msg = `¡Compra exitosa! Compraste ${res.executedQuantity} token${res.executedQuantity === 1 ? '' : 's'} de ${playerName}.`;
        } else if (res.status === 'PARTIALLY_FILLED') {
          msg = `Compra parcial: Compraste ${res.executedQuantity} de ${res.quantity} tokens. Los ${res.remainingQuantity} restantes quedaron esperando en el mercado.`;
        } else if (res.status === 'PENDING') {
          msg = `Tu orden límite fue publicada con éxito. Quedó abierta en el mercado esperando una oferta. Podés gestionarla en "Mi Patrimonio".`;
        } else {
          msg = 'Orden procesada correctamente.';
        }
        setFeedback({ type: 'success', message: msg });
      } else {
        const res = await placeSellOrder(payload);
        let msg = '';
        if (res.status === 'FILLED') {
          msg = `¡Venta exitosa! Vendiste ${res.executedQuantity} token${res.executedQuantity === 1 ? '' : 's'} de ${playerName}.`;
        } else if (res.status === 'PARTIALLY_FILLED') {
          msg = `Venta parcial: Vendiste ${res.executedQuantity} de ${res.quantity} tokens. Los ${res.remainingQuantity} restantes quedaron publicados en el mercado.`;
        } else if (res.status === 'PENDING') {
          msg = `Tu orden de venta fue publicada con éxito. Quedó abierta en el mercado a la espera de un comprador. Podés gestionarla en "Mi Patrimonio".`;
        } else {
          msg = 'Orden de venta procesada correctamente.';
        }
        setFeedback({ type: 'success', message: msg });
      }
      setQuantity('1');
      await loadPortfolio();
      if (showOrderBook) loadOrderBook();
      if (onTradeExecuted) onTradeExecuted();
    } catch (err: unknown) {
      if (err && typeof err === 'object' && 'response' in err) {
        const axiosErr = err as { response?: { status?: number; data?: { error?: string } } };
        if (axiosErr.response?.status === 401) {
          setFeedback({
            type: 'error',
            message: 'Tu sesión ha expirado. Por favor volvé a iniciar sesión.',
          });
          if (onOpenAuth) onOpenAuth();
        } else if (axiosErr.response?.status === 409) {
          setFeedback({
            type: 'error',
            message:
              side === 'BUY'
                ? 'No hay órdenes de venta en este momento para completar la compra inmediata (o todas las ofertas activas pertenecen a tu usuario).'
                : 'No hay órdenes de compra en este momento para completar la venta inmediata (o todas las ofertas activas pertenecen a tu usuario).',
          });
        } else if (axiosErr.response?.data?.error) {
          setFeedback({ type: 'error', message: axiosErr.response.data.error });
        } else {
          setFeedback({ type: 'error', message: 'Error al enviar la orden al mercado.' });
        }
      } else {
        setFeedback({ type: 'error', message: 'Error de conexión con el servidor.' });
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  const buyOrders = orderBook.filter((o) => o.side === 'BUY');
  const sellOrders = orderBook.filter((o) => o.side === 'SELL');

  return (
    <div className="player-trade">
      <div className="player-trade__header">
        <h3 className="player-trade__title">Mercado de Tokens</h3>
        <div className="player-trade__user-stats">
          <div className="player-trade__stats-row">
            <span>Saldo disponible: <strong>{formatCredits(availableCash, 2)} Créditos</strong></span>
            <span>
              Tokens en cartera: <strong>{totalOwnedTokens}</strong>
              {reservedTokens > 0 && (
                <span style={{ fontSize: '0.85em', opacity: 0.85, marginLeft: '4px' }}>
                  ({availableTokens} disp. / {reservedTokens} en orden)
                </span>
              )}
            </span>
          </div>
          <div className="player-trade__quote-box">
            <span>Cotización oficial:</span>
            <strong>{formatCredits(activeQuote, 4)} Créditos</strong>
          </div>
        </div>
      </div>

      <div className="player-trade__tabs">
        <button
          type="button"
          className={`player-trade__tab player-trade__tab--buy ${
            side === 'BUY' ? 'player-trade__tab--active' : ''
          }`}
          onClick={() => handleTabChange('BUY')}
        >
          Comprar
        </button>
        <button
          type="button"
          className={`player-trade__tab player-trade__tab--sell ${
            side === 'SELL' ? 'player-trade__tab--active' : ''
          }`}
          onClick={() => handleTabChange('SELL')}
        >
          Vender
        </button>
      </div>

      {feedback && (
        <div className={`player-trade__alert player-trade__alert--${feedback.type}`} role="alert">
          {feedback.message}
        </div>
      )}

      <form className="player-trade__form" onSubmit={handleSubmit}>
        <div className="player-trade__field">
          <label className="player-trade__label" htmlFor="trade-quantity">
            Cantidad de tokens
          </label>
          <div className="player-trade__input-row">
            <input
              id="trade-quantity"
              type="text"
              inputMode="numeric"
              className="player-trade__input"
              value={quantity}
              onChange={handleQuantityChange}
              placeholder="1"
              required
            />
            {side === 'BUY' && maxAffordable > 0 && (
              <button
                type="button"
                className="player-trade__max-btn"
                onClick={() => setQuantity(Math.max(1, maxAffordable).toString())}
              >
                Máx ({maxAffordable})
              </button>
            )}
            {side === 'SELL' && availableTokens > 0 && (
              <button
                type="button"
                className="player-trade__max-btn"
                onClick={() => setQuantity(availableTokens.toString())}
              >
                Máx ({availableTokens})
              </button>
            )}
          </div>
          <span className="player-trade__hint">
            {side === 'BUY'
              ? `Tope de compra: hasta ${MAX_PLAYER_SUPPLY} tokens (suministro emitido del jugador)`
              : reservedTokens > 0
              ? `Podés vender hasta ${availableTokens} tokens (${reservedTokens} reservados en órdenes abiertas)`
              : `Podés vender hasta ${availableTokens} tokens`}
          </span>
        </div>

        <div className="player-trade__field">
          <label className="player-trade__label">Tipo de Orden</label>
          <div className="player-trade__radios">
            <label className="player-trade__radio-label">
              <input
                type="radio"
                name="orderType"
                value="MARKET"
                checked={orderType === 'MARKET'}
                onChange={() => setOrderType('MARKET')}
              />
              Mercado (FOK)
            </label>
            <label className="player-trade__radio-label">
              <input
                type="radio"
                name="orderType"
                value="LIMIT"
                checked={orderType === 'LIMIT'}
                onChange={() => setOrderType('LIMIT')}
              />
              Precio Límite (GTC)
            </label>
          </div>

          <div className="player-trade__type-hint">
            {orderType === 'MARKET' ? (
              side === 'BUY' ? (
                <><strong>Inmediata:</strong> Comprás a la cotización oficial de <strong>{formatCredits(activeQuote, 4)} Créditos</strong>.</>
              ) : (
                <><strong>Inmediata:</strong> Vendés a la cotización oficial de <strong>{formatCredits(activeQuote, 4)} Créditos</strong>.</>
              )
            ) : (
              side === 'BUY' ? (
                <><strong>Precio tope:</strong> Definís el precio máximo a pagar. Si no hay ofertas inmediatas a ese valor, la orden queda abierta en el mercado hasta ejecutarse o cancelarse.</>
              ) : (
                <><strong>Precio piso:</strong> Definís el precio mínimo a recibir. Si no hay compradores inmediatos a ese valor, la orden queda abierta esperando en el mercado.</>
              )
            )}
          </div>
        </div>

        {orderType === 'LIMIT' && (
          <div className="player-trade__field">
            <label className="player-trade__label" htmlFor="trade-limit-price">
              Precio Límite (en Créditos)
            </label>
            <input
              id="trade-limit-price"
              type="text"
              inputMode="decimal"
              className="player-trade__input"
              value={limitPrice}
              onChange={handlePriceChange}
              placeholder="Ej: 1.0000"
              required
            />
            <span className="player-trade__hint">
              Máximo 4 decimales (ej: 1.5000). Cotización oficial: {formatCredits(activeQuote, 4)} Créditos
            </span>
          </div>
        )}

        <div className="player-trade__summary-box">
          <div className="player-trade__summary-row">
            <span>Precio unitario estimado:</span>
            <strong>{formatCredits(parsedPrice, 4)} Créditos</strong>
          </div>
          <div className="player-trade__summary-row player-trade__summary-row--total">
            <span>{side === 'BUY' ? 'Total a pagar:' : 'Total estimado a recibir:'}</span>
            <strong>{formatCredits(estimatedTotal, 4)} Créditos</strong>
          </div>
        </div>

        <button
          type="submit"
          className={`player-trade__submit-btn ${
            side === 'BUY' ? 'player-trade__submit-btn--buy' : 'player-trade__submit-btn--sell'
          }`}
          disabled={isSubmitting || (side === 'SELL' && availableTokens === 0)}
        >
          {isSubmitting
            ? 'Enviando orden...'
            : side === 'BUY'
            ? `Comprar ${parsedQty} token${parsedQty === 1 ? '' : 's'}`
            : availableTokens === 0
            ? reservedTokens > 0
              ? 'Tokens en órdenes abiertas'
              : 'No poseés tokens para vender'
            : `Vender ${parsedQty} token${parsedQty === 1 ? '' : 's'}`}
        </button>
      </form>

      <div className="player-trade__book-section">
        <button
          type="button"
          className="player-trade__book-toggle"
          onClick={() => {
            const next = !showOrderBook;
            setShowOrderBook(next);
            if (next) loadOrderBook();
          }}
          aria-expanded={showOrderBook}
        >
          <span>Cola de Órdenes del Mercado</span>
          <span>{showOrderBook ? 'Ocultar' : 'Ver cola'}</span>
        </button>

        {showOrderBook && (
          <div className="player-trade__book-content">
            {isLoadingBook ? (
              <p className="player-trade__book-empty">Cargando cola de órdenes...</p>
            ) : (
              <div className="player-trade__book-columns">
                <div className="player-trade__book-column">
                  <h4 className="player-trade__book-col-title player-trade__book-col-title--buy">
                    Compras ({buyOrders.length})
                  </h4>
                  {buyOrders.length === 0 ? (
                    <p className="player-trade__book-empty">Sin órdenes de compra</p>
                  ) : (
                    <ul className="player-trade__book-list">
                      {buyOrders.map((ord) => (
                        <li
                          key={ord.id}
                          className={`player-trade__book-item ${
                            user && ord.userId === user.id ? 'player-trade__book-item--mine' : ''
                          }`}
                        >
                          <span>
                            <strong>{formatCredits(ord.price, 4)}</strong> cr. × {ord.remainingQuantity}
                          </span>
                          {user && ord.userId === user.id && (
                            <span className="player-trade__book-badge">Tu orden</span>
                          )}
                        </li>
                      ))}
                    </ul>
                  )}
                </div>

                <div className="player-trade__book-column">
                  <h4 className="player-trade__book-col-title player-trade__book-col-title--sell">
                    Ventas ({sellOrders.length})
                  </h4>
                  {sellOrders.length === 0 ? (
                    <p className="player-trade__book-empty">Sin órdenes de venta</p>
                  ) : (
                    <ul className="player-trade__book-list">
                      {sellOrders.map((ord) => (
                        <li
                          key={ord.id}
                          className={`player-trade__book-item ${
                            user && ord.userId === user.id ? 'player-trade__book-item--mine' : ''
                          }`}
                        >
                          <span>
                            <strong>{formatCredits(ord.price, 4)}</strong> cr. × {ord.remainingQuantity}
                          </span>
                          {user && ord.userId === user.id && (
                            <span className="player-trade__book-badge">Tu orden</span>
                          )}
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};
