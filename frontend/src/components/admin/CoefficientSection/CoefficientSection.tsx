import React, { useEffect, useState } from 'react';
import { getLeagueCoefficientOptional, setLeagueCoefficient } from '../../../api/admin';
import { LEAGUES } from '../../../utils/leagues';
import { formatDateTime } from '../../../utils/format';
import { AdminMessageKind, StatusMessage } from '../AdminFeedback/AdminFeedback';
import './CoefficientSection.css';

interface CoefficientRow {
  code: string;
  name: string;
  coefficient: number | null;
  effectiveFrom: string | null;
  input: string;
  isLoading: boolean;
}

interface AdminMessage {
  kind: AdminMessageKind;
  text: string;
}

const initialRows = (): CoefficientRow[] =>
  LEAGUES.map((league) => ({
    ...league,
    coefficient: null,
    effectiveFrom: null,
    input: '',
    isLoading: true,
  }));

export const CoefficientSection: React.FC = () => {
  const [rows, setRows] = useState<CoefficientRow[]>(initialRows);
  const [savingCode, setSavingCode] = useState<string | null>(null);
  const [message, setMessage] = useState<AdminMessage | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      const results = await Promise.allSettled(
        LEAGUES.map((league) => getLeagueCoefficientOptional(league.code)),
      );
      if (cancelled) {
        return;
      }

      let hadError = false;
      const next = LEAGUES.map((league, index) => {
        const result = results[index];
        if (result.status === 'fulfilled') {
          const dto = result.value;
          return {
            ...league,
            coefficient: dto?.coefficient ?? null,
            effectiveFrom: dto?.effectiveFrom ?? null,
            input: dto ? String(dto.coefficient) : '',
            isLoading: false,
          };
        }

        hadError = true;
        return {
          ...league,
          coefficient: null,
          effectiveFrom: null,
          input: '',
          isLoading: false,
        };
      });

      setRows(next);
      setLoadError(hadError ? 'No se pudieron cargar algunos coeficientes.' : null);
    };

    void load();
    return () => {
      cancelled = true;
    };
  }, []);

  const updateInput = (code: string, input: string) => {
    setRows((prev) => prev.map((row) => (row.code === code ? { ...row, input } : row)));
  };

  const handleSave = async (row: CoefficientRow) => {
    const value = Number(row.input);
    if (!Number.isFinite(value) || value <= 0) {
      setMessage({ kind: 'error', text: 'El coeficiente debe ser un número mayor que cero.' });
      return;
    }

    setSavingCode(row.code);
    setMessage(null);
    try {
      const updated = await setLeagueCoefficient(row.code, value);
      setRows((prev) =>
        prev.map((current) =>
          current.code === row.code
            ? {
                ...current,
                coefficient: updated.coefficient,
                effectiveFrom: updated.effectiveFrom,
                input: String(updated.coefficient),
              }
            : current,
        ),
      );
      setMessage({ kind: 'success', text: `Coeficiente de ${row.name} actualizado.` });
    } catch {
      setMessage({ kind: 'error', text: `No se pudo actualizar el coeficiente de ${row.name}.` });
    } finally {
      setSavingCode(null);
    }
  };

  return (
    <section className="coefficient-section" id="admin-coeficientes">
      <header className="coefficient-section__header">
        <h2 className="coefficient-section__title">Coeficientes por liga</h2>
        <p className="coefficient-section__subtitle">
          El coeficiente se aplica al leer el score. Cambiarlo no dispara un recálculo.
        </p>
      </header>

      {message && <StatusMessage kind={message.kind}>{message.text}</StatusMessage>}
      {loadError && <StatusMessage kind="error">{loadError}</StatusMessage>}

      <div className="coefficient-section__table-wrapper">
        <table className="coefficient-section__table">
          <thead>
            <tr>
              <th scope="col">Liga</th>
              <th scope="col">Coeficiente actual</th>
              <th scope="col">Vigente desde</th>
              <th scope="col">Nuevo coeficiente</th>
              <th scope="col">Acción</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => (
              <tr key={row.code}>
                <td>
                  <span className="coefficient-section__league">{row.name}</span>
                  <span className="coefficient-section__code">{row.code}</span>
                </td>
                <td>
                  {row.isLoading
                    ? '...'
                    : row.coefficient === null
                      ? 'Sin definir'
                      : row.coefficient.toFixed(4)}
                </td>
                <td>{formatDateTime(row.effectiveFrom)}</td>
                <td>
                  <input
                    className="coefficient-section__input"
                    type="number"
                    min="0.0001"
                    step="0.01"
                    value={row.input}
                    onChange={(event) => updateInput(row.code, event.target.value)}
                    disabled={row.isLoading || savingCode !== null}
                  />
                </td>
                <td>
                  <button
                    type="button"
                    className="coefficient-section__button"
                    onClick={() => void handleSave(row)}
                    disabled={row.isLoading || savingCode !== null}
                  >
                    {savingCode === row.code ? 'Guardando...' : 'Guardar'}
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
};
