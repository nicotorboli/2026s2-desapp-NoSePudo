import React, { useCallback, useEffect, useState } from 'react';
import {
  getPlayerPrice,
  getPlayerQuotes,
  PlayerPriceDTO,
  PlayerQuoteDTO,
} from '../../api/players';
import { formatCredits, formatDateTime } from '../../utils/format';
import './PlayerQuote.css';

interface PlayerQuoteProps {
  playerId: number;
}

export const PlayerQuote: React.FC<PlayerQuoteProps> = ({ playerId }) => {
  const [quotes, setQuotes] = useState<PlayerQuoteDTO[]>([]);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [date, setDate] = useState<string>('');
  const [dateResult, setDateResult] = useState<PlayerPriceDTO | null>(null);
  const [isSearching, setIsSearching] = useState<boolean>(false);
  const [searchError, setSearchError] = useState<string | null>(null);

  useEffect(() => {
    let isMounted = true;
    setIsLoading(true);

    getPlayerQuotes(playerId)
      .then((data) => {
        if (isMounted) {
          setQuotes(data);
        }
      })
      .catch(() => {
        if (isMounted) {
          setQuotes([]);
        }
      })
      .finally(() => {
        if (isMounted) {
          setIsLoading(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [playerId]);

  const handleSearch = useCallback(
    async (event: React.FormEvent) => {
      event.preventDefault();
      if (!date) {
        return;
      }

      setIsSearching(true);
      setSearchError(null);
      setDateResult(null);

      try {
        const result = await getPlayerPrice(playerId, date);
        setDateResult(result);
      } catch {
        setSearchError('No hay cotización para esa fecha.');
      } finally {
        setIsSearching(false);
      }
    },
    [date, playerId]
  );

  const current = quotes[0] ?? null;

  return (
    <section className="player-quote" aria-label="Cotización del jugador">
      <h2 className="player-quote__title">Cotización</h2>

      <div className="player-quote__current">
        <span className="player-quote__label">Cotización actual</span>
        {current === null ? (
          <span className="player-quote__empty">Sin cotización</span>
        ) : (
          <span className="player-quote__price">{`${formatCredits(current.price)} Créditos`}</span>
        )}
      </div>

      <div className="player-quote__history-block">
        <h3 className="player-quote__subtitle">Historial de cotizaciones</h3>
        {isLoading ? (
          <p className="player-quote__empty">Cargando cotizaciones...</p>
        ) : quotes.length === 0 ? (
          <p className="player-quote__empty">Sin cotizaciones registradas</p>
        ) : (
          <table className="player-quote__history">
            <thead>
              <tr>
                <th scope="col">Fecha</th>
                <th scope="col">Precio</th>
              </tr>
            </thead>
            <tbody>
              {quotes.map((quote, index) => (
                <tr key={`${quote.createdAt}-${index}`}>
                  <td>{formatDateTime(quote.createdAt)}</td>
                  <td>{`${formatCredits(quote.price)} Créditos`}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <form className="player-quote__search" onSubmit={handleSearch}>
        <label className="player-quote__search-label" htmlFor="quote-date">
          Buscar cotización por fecha
        </label>
        <div className="player-quote__search-row">
          <input
            id="quote-date"
            type="date"
            className="player-quote__date"
            value={date}
            onChange={(event) => setDate(event.target.value)}
          />
          <button
            type="submit"
            className="player-quote__search-button"
            disabled={!date || isSearching}
          >
            {isSearching ? 'Buscando...' : 'Consultar'}
          </button>
        </div>
      </form>

      {searchError && (
        <p className="player-quote__error" role="alert">
          {searchError}
        </p>
      )}

      {dateResult && (
        <div className="player-quote__result">
          <span className="player-quote__label">Cotización al {date}</span>
          <span className="player-quote__price">{`${formatCredits(dateResult.price)} Créditos`}</span>
        </div>
      )}
    </section>
  );
};
