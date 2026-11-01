import React, { useState, useEffect, useCallback } from 'react';
import { useAuth } from '../../context/AuthContext';
import { getUserPortfolio, getUserTransactions, PortfolioResponseDTO, TransactionPageDTO } from '../../api/portfolio';
import { getOpenOrders, cancelOrder, OpenOrderDTO } from '../../api/orders';
import { parseCredits, formatCredits, translateOrderStatus } from '../../utils/format';
import './PortfolioPage.css';

interface PortfolioPageProps {
  onSelectPlayer: (playerId: number) => void;
  onBackToCatalog: () => void;
}

export const PortfolioPage: React.FC<PortfolioPageProps> = ({
  onSelectPlayer,
  onBackToCatalog,
}) => {
  const { user, isAuthenticated } = useAuth();
  const [activeTab, setActiveTab] = useState<'positions' | 'orders' | 'transactions'>('positions');
  const [portfolio, setPortfolio] = useState<PortfolioResponseDTO | null>(null);
  const [openOrders, setOpenOrders] = useState<OpenOrderDTO[]>([]);
  const [transactionsPage, setTransactionsPage] = useState<TransactionPageDTO | null>(null);
  const [txCurrentPage, setTxCurrentPage] = useState(1);
  const [isLoading, setIsLoading] = useState(true);
  const [feedback, setFeedback] = useState<{ type: 'success' | 'error'; message: string } | null>(null);

  const loadData = useCallback(async () => {
    if (!user) return;
    setIsLoading(true);
    try {
      const [portData, ordersData, txData] = await Promise.all([
        getUserPortfolio(user.id),
        getOpenOrders(),
        getUserTransactions(user.id, txCurrentPage, 10),
      ]);
      setPortfolio(portData);
      setOpenOrders(ordersData);
      setTransactionsPage(txData);
    } catch {
      setFeedback({ type: 'error', message: 'Error al cargar los datos de patrimonio.' });
    } finally {
      setIsLoading(false);
    }
  }, [user, txCurrentPage]);

  useEffect(() => {
    if (isAuthenticated && user) {
      loadData();
    }
  }, [isAuthenticated, user, loadData]);

  const handleCancelOrder = async (orderId: number) => {
    setFeedback(null);
    try {
      const res = await cancelOrder(orderId);
      let details = '';
      if (res.releasedCredits !== undefined && res.releasedCredits !== null) {
        details = ` Se liberaron ${formatCredits(res.releasedCredits, 4)} créditos al saldo disponible.`;
      } else if (res.releasedTokens !== undefined && res.releasedTokens !== null) {
        details = ` Se liberaron ${res.releasedTokens} tokens a la cartera.`;
      }
      setFeedback({
        type: 'success',
        message: `Orden #${orderId} cancelada exitosamente.${details}`,
      });
      await loadData();
    } catch {
      setFeedback({
        type: 'error',
        message: `No se pudo cancelar la orden #${orderId}. Quizás ya fue ejecutada.`,
      });
    }
  };

  if (!isAuthenticated || !user) {
    return (
      <main className="portfolio-page">
        <div className="portfolio-page__empty">
          <h2 className="portfolio-page__empty-title">Acceso restringido</h2>
          <p>Debes iniciar sesión para consultar tu patrimonio y órdenes de mercado.</p>
          <button
            type="button"
            className="portfolio-table__action-btn"
            onClick={onBackToCatalog}
          >
            ← Volver al Catálogo
          </button>
        </div>
      </main>
    );
  }

  const availableCash = parseCredits(portfolio?.availableCash);
  const reservedCash = parseCredits(portfolio?.reservedCash);
  const tokensValuation = parseCredits(portfolio?.tokensValuation);
  const totalValue = parseCredits(portfolio?.totalValue);
  const unrealizedResult = parseCredits(portfolio?.unrealizedResult);
  const positions = portfolio?.positions ?? [];

  return (
    <main className="portfolio-page">
      <header className="portfolio-page__header">
        <h1 className="portfolio-page__title">Mi Patrimonio</h1>
        <p className="portfolio-page__subtitle">
          Resumen financiero, posiciones en tokens y gestión de órdenes activas.
        </p>
      </header>

      {feedback && (
        <div className={`portfolio-page__alert portfolio-page__alert--${feedback.type}`} role="alert">
          {feedback.message}
        </div>
      )}

      <section className="portfolio-page__cards-grid" aria-label="Resumen de patrimonio">
        <div className="portfolio-card">
          <span className="portfolio-card__label">Saldo Disponible</span>
          <span className="portfolio-card__value">{formatCredits(availableCash, 2)} Créditos</span>
          <span className="portfolio-card__desc">Listo para nuevas operaciones</span>
        </div>

        <div className="portfolio-card">
          <span className="portfolio-card__label">Saldo Reservado</span>
          <span className="portfolio-card__value">{formatCredits(reservedCash, 2)} Créditos</span>
          <span className="portfolio-card__desc">Comprometido en órdenes de compra</span>
        </div>

        <div className="portfolio-card">
          <span className="portfolio-card__label">Valuación de Tokens</span>
          <span className="portfolio-card__value">{formatCredits(tokensValuation, 2)} Créditos</span>
          <span className="portfolio-card__desc">{positions.length} jugador{positions.length === 1 ? '' : 'es'} en cartera</span>
        </div>

        <div className="portfolio-card">
          <span className="portfolio-card__label">Patrimonio Total</span>
          <span className="portfolio-card__value">{formatCredits(totalValue, 2)} Créditos</span>
          <span
            className={`portfolio-card__badge ${
              unrealizedResult >= 0 ? 'portfolio-card__badge--positive' : 'portfolio-card__badge--negative'
            }`}
          >
            {unrealizedResult >= 0 ? '+' : ''}
            {formatCredits(unrealizedResult, 2)} Créditos ({totalValue > 0 ? ((unrealizedResult / (totalValue - unrealizedResult || 1)) * 100).toFixed(1) : 0}%)
          </span>
        </div>
      </section>

      <nav className="portfolio-page__tabs" aria-label="Secciones del patrimonio">
        <button
          type="button"
          className={`portfolio-page__tab ${activeTab === 'positions' ? 'portfolio-page__tab--active' : ''}`}
          onClick={() => setActiveTab('positions')}
        >
          Posiciones <span className="portfolio-page__tab-badge">{positions.length}</span>
        </button>
        <button
          type="button"
          className={`portfolio-page__tab ${activeTab === 'orders' ? 'portfolio-page__tab--active' : ''}`}
          onClick={() => setActiveTab('orders')}
        >
          Órdenes Abiertas <span className="portfolio-page__tab-badge">{openOrders.length}</span>
        </button>
        <button
          type="button"
          className={`portfolio-page__tab ${activeTab === 'transactions' ? 'portfolio-page__tab--active' : ''}`}
          onClick={() => setActiveTab('transactions')}
        >
          Historial de Operaciones
        </button>
      </nav>

      {activeTab === 'positions' && (
        <section className="portfolio-page__table-wrapper" aria-label="Tabla de posiciones">
          {positions.length === 0 ? (
            <div className="portfolio-page__empty">
              <p className="portfolio-page__empty-title">Aún no poseés tokens de jugadores</p>
              <p>Explorá el catálogo de jugadores y colocá tu primera orden de compra.</p>
              <button
                type="button"
                className="portfolio-table__action-btn"
                onClick={onBackToCatalog}
              >
                Ir al Catálogo
              </button>
            </div>
          ) : (
            <table className="portfolio-table">
              <thead>
                <tr>
                  <th>Jugador</th>
                  <th>Club</th>
                  <th>Posición</th>
                  <th>Cantidad</th>
                  <th>Precio Promedio</th>
                  <th>Cotización</th>
                  <th>Valuación</th>
                  <th>Rendimiento</th>
                  <th>Acción</th>
                </tr>
              </thead>
              <tbody>
                {positions.map((pos) => {
                  const unResult = parseCredits(pos.unrealizedResult);
                  return (
                    <tr key={pos.playerId}>
                      <td><strong>{pos.playerName}</strong></td>
                      <td>{pos.clubName}</td>
                      <td>{pos.position}</td>
                      <td>
                        <strong>{pos.quantity}</strong>
                        {pos.reservedQuantity && pos.reservedQuantity > 0 ? (
                          <span style={{ display: 'block', fontSize: '0.8em', color: '#666' }}>
                            ({pos.availableQuantity ?? (pos.quantity - pos.reservedQuantity)} disp. / {pos.reservedQuantity} en orden)
                          </span>
                        ) : null}
                      </td>
                      <td>{formatCredits(pos.averagePrice, 4)} Créditos</td>
                      <td>{formatCredits(pos.currentPrice, 4)} Créditos</td>
                      <td><strong>{formatCredits(pos.currentValuation, 4)} Créditos</strong></td>
                      <td>
                        <span
                          className={
                            unResult >= 0
                              ? 'portfolio-table__badge--positive'
                              : 'portfolio-table__badge--negative'
                          }
                        >
                          {unResult >= 0 ? '+' : ''}
                          {formatCredits(unResult, 4)} Créditos
                        </span>
                      </td>
                      <td>
                        <button
                          type="button"
                          className="portfolio-table__action-btn"
                          onClick={() => onSelectPlayer(pos.playerId)}
                        >
                          Operar
                        </button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          )}
        </section>
      )}

      {activeTab === 'orders' && (
        <section className="portfolio-page__table-wrapper" aria-label="Tabla de órdenes abiertas">
          {openOrders.length === 0 ? (
            <div className="portfolio-page__empty">
              <p className="portfolio-page__empty-title">No tenés órdenes abiertas activas</p>
              <p>Todas tus órdenes han sido ejecutadas o canceladas.</p>
            </div>
          ) : (
            <table className="portfolio-table">
              <thead>
                <tr>
                  <th>ID</th>
                  <th>Tipo</th>
                  <th>ID Jugador</th>
                  <th>Precio Límite</th>
                  <th>Total</th>
                  <th>Ejecutados</th>
                  <th>Restantes</th>
                  <th>Vigencia</th>
                  <th>Estado</th>
                  <th>Acción</th>
                </tr>
              </thead>
              <tbody>
                {openOrders.map((ord) => (
                  <tr key={ord.id}>
                    <td>#{ord.id}</td>
                    <td>
                      <span
                        className={`portfolio-table__badge ${
                          ord.side === 'BUY'
                            ? 'portfolio-table__badge--buy'
                            : 'portfolio-table__badge--sell'
                        }`}
                      >
                        {ord.side === 'BUY' ? 'COMPRA' : 'VENTA'}
                      </span>
                    </td>
                    <td>
                      <button
                        type="button"
                        className="portfolio-table__action-btn"
                        onClick={() => onSelectPlayer(ord.playerId)}
                      >
                        Jugador #{ord.playerId}
                      </button>
                    </td>
                    <td>{formatCredits(ord.price, 4)} Créditos</td>
                    <td>{ord.quantity}</td>
                    <td>{ord.executedQuantity}</td>
                    <td><strong>{ord.remainingQuantity}</strong></td>
                    <td>{ord.timeInForce}</td>
                    <td>{translateOrderStatus(ord.status)}</td>
                    <td>
                      <button
                        type="button"
                        className="portfolio-table__action-btn portfolio-table__action-btn--cancel"
                        onClick={() => handleCancelOrder(ord.id)}
                      >
                        Cancelar
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </section>
      )}

      {activeTab === 'transactions' && (
        <section className="portfolio-page__table-wrapper" aria-label="Historial de transacciones">
          {!transactionsPage || transactionsPage.items.length === 0 ? (
            <div className="portfolio-page__empty">
              <p className="portfolio-page__empty-title">No hay transacciones registradas</p>
              <p>Las compras y ventas ejecutadas aparecerán en esta tabla.</p>
            </div>
          ) : (
            <>
              <table className="portfolio-table">
                <thead>
                  <tr>
                    <th>Fecha y Hora</th>
                    <th>Operación</th>
                    <th>Jugador</th>
                    <th>Cantidad</th>
                    <th>Precio Unitario</th>
                    <th>Total</th>
                  </tr>
                </thead>
                <tbody>
                  {transactionsPage.items.map((tx) => (
                    <tr key={tx.id}>
                      <td>{new Date(tx.executedAt).toLocaleString()}</td>
                      <td>
                        <span
                          className={`portfolio-table__badge ${
                            tx.side === 'BUY'
                              ? 'portfolio-table__badge--buy'
                              : 'portfolio-table__badge--sell'
                          }`}
                        >
                          {tx.side === 'BUY' ? 'COMPRA' : 'VENTA'}
                        </span>
                      </td>
                      <td>
                        <strong>{tx.playerName}</strong>
                      </td>
                      <td>{tx.quantity}</td>
                      <td>{formatCredits(tx.price, 4)} Créditos</td>
                      <td><strong>{formatCredits(tx.totalAmount, 4)} Créditos</strong></td>
                    </tr>
                  ))}
                </tbody>
              </table>

              {transactionsPage.totalPages > 1 && (
                <div className="portfolio-page__pagination">
                  <span>
                    Página {transactionsPage.page} de {transactionsPage.totalPages} ({transactionsPage.total} operaciones)
                  </span>
                  <div style={{ display: 'flex', gap: '0.5rem' }}>
                    <button
                      type="button"
                      className="portfolio-page__page-btn"
                      disabled={txCurrentPage <= 1 || isLoading}
                      onClick={() => setTxCurrentPage((p) => Math.max(1, p - 1))}
                    >
                      Anterior
                    </button>
                    <button
                      type="button"
                      className="portfolio-page__page-btn"
                      disabled={txCurrentPage >= transactionsPage.totalPages || isLoading}
                      onClick={() => setTxCurrentPage((p) => p + 1)}
                    >
                      Siguiente
                    </button>
                  </div>
                </div>
              )}
            </>
          )}
        </section>
      )}
    </main>
  );
};
