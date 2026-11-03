import React, { useCallback, useEffect, useState } from 'react';
import {
  activateStrategy,
  getActiveStrategyOptional,
  listStrategies,
  ScoringStrategyDTO,
} from '../../../api/admin';
import { translatePosition, translateStat } from '../../../utils/translations';
import { AdminMessageKind, ConfirmDialog, StatusMessage } from '../AdminFeedback/AdminFeedback';
import './StrategySection.css';

interface AdminMessage {
  kind: AdminMessageKind;
  text: string;
}

const WeightTable: React.FC<{ strategy: ScoringStrategyDTO }> = ({ strategy }) => {
  const positions = Object.entries(strategy.weights ?? {});

  if (positions.length === 0) {
    return <p className="strategy-section__meta">Sin pesos definidos.</p>;
  }

  return (
    <div className="strategy-section__weights">
      {positions.map(([position, stats]) => (
        <div key={position} className="strategy-section__position">
          <h4 className="strategy-section__position-title">{translatePosition(position)}</h4>
          <ul className="strategy-section__weight-list">
            {Object.entries(stats).map(([stat, weight]) => (
              <li key={stat} className="strategy-section__weight">
                <span>{translateStat(stat)}</span>
                <span className="strategy-section__weight-value">{weight.toFixed(3)}</span>
              </li>
            ))}
          </ul>
        </div>
      ))}
    </div>
  );
};

export const StrategySection: React.FC = () => {
  const [strategies, setStrategies] = useState<ScoringStrategyDTO[]>([]);
  const [activeId, setActiveId] = useState<number | null>(null);
  const [expandedId, setExpandedId] = useState<number | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [message, setMessage] = useState<AdminMessage | null>(null);
  const [confirmId, setConfirmId] = useState<number | null>(null);
  const [activatingId, setActivatingId] = useState<number | null>(null);

  const load = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const [list, active] = await Promise.all([
        listStrategies(),
        getActiveStrategyOptional(),
      ]);
      setStrategies(list);
      setActiveId(active?.id ?? null);
      setExpandedId((current) => current ?? list[0]?.id ?? null);
    } catch {
      setError('No se pudieron cargar las estrategias de scoring.');
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const handleActivate = async () => {
    if (confirmId === null) {
      return;
    }

    setActivatingId(confirmId);
    setMessage(null);
    try {
      const active = await activateStrategy(confirmId);
      setActiveId(active.id);
      setMessage({
        kind: 'success',
        text: `Estrategia "${active.name}" activada. Se recalcularon los scores.`,
      });
      setConfirmId(null);
    } catch {
      setMessage({ kind: 'error', text: 'No se pudo activar la estrategia.' });
    } finally {
      setActivatingId(null);
    }
  };

  return (
    <section className="strategy-section" id="admin-estrategias">
      <header className="strategy-section__header">
        <h2 className="strategy-section__title">Estrategias de scoring</h2>
        <p className="strategy-section__subtitle">
          Consultá los pesos de cada estrategia y activá la que quieras usar. Activar una
          estrategia recalcula los scores de todos los jugadores.
        </p>
      </header>

      {message && <StatusMessage kind={message.kind}>{message.text}</StatusMessage>}
      {error && <StatusMessage kind="error">{error}</StatusMessage>}

      {isLoading ? (
        <p className="strategy-section__meta">Cargando estrategias...</p>
      ) : strategies.length === 0 ? (
        <p className="strategy-section__meta">No hay estrategias disponibles.</p>
      ) : (
        <div className="strategy-section__list">
          {strategies.map((strategy) => {
            const isActive = strategy.id === activeId;
            const isExpanded = expandedId === strategy.id;

            return (
              <article
                key={strategy.id}
                className={`strategy-section__card ${
                  isActive ? 'strategy-section__card--active' : ''
                }`}
              >
                <header className="strategy-section__card-header">
                  <div>
                    <h3 className="strategy-section__card-title">{strategy.name}</h3>
                    <p className="strategy-section__meta">
                      ID {strategy.id} · α = {strategy.alpha.toFixed(3)}
                    </p>
                  </div>
                  {isActive && <span className="strategy-section__badge">Activa</span>}
                </header>

                <div className="strategy-section__actions">
                  <button
                    type="button"
                    className="strategy-section__button"
                    onClick={() => setExpandedId(isExpanded ? null : strategy.id)}
                  >
                    {isExpanded ? 'Ocultar pesos' : 'Ver pesos'}
                  </button>
                  <button
                    type="button"
                    className="strategy-section__button strategy-section__button--primary"
                    onClick={() => setConfirmId(strategy.id)}
                    disabled={isActive || activatingId !== null}
                  >
                    {isActive ? 'En uso' : 'Activar'}
                  </button>
                </div>

                {isExpanded && <WeightTable strategy={strategy} />}
              </article>
            );
          })}
        </div>
      )}

      <ConfirmDialog
        open={confirmId !== null}
        title="Activar estrategia"
        description="Activar una estrategia recalcula los scores de todos los jugadores. Puede tardar varios minutos."
        confirmLabel="Activar"
        busy={activatingId !== null}
        onConfirm={() => void handleActivate()}
        onCancel={() => setConfirmId(null)}
      />
    </section>
  );
};
