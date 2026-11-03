import React, { useCallback, useEffect, useState } from 'react';
import {
  createSyncRun,
  getSyncRun,
  listSyncRuns,
  listSyncSchedules,
  SyncRunDTO,
  SyncRunStatus,
  SyncScheduleDTO,
  SyncType,
  updateSyncSchedule,
} from '../../../api/admin';
import { AdminMessageKind, ConfirmDialog, StatusMessage } from '../AdminFeedback/AdminFeedback';
import { formatDateTime, formatDuration } from '../../../utils/format';
import './SyncSection.css';

const POLL_INTERVAL_MS = 2000;
const MIN_INTERVAL_MINUTES = 15;

const SYNC_LABELS: Record<SyncType, string> = {
  players: 'catálogo',
  statistics: 'estadísticas',
};

const STATUS_LABELS: Record<SyncRunStatus, string> = {
  running: 'En curso',
  succeeded: 'Completada',
  partial: 'Parcial',
  failed: 'Fallida',
  skipped: 'Omitida',
};

interface ScheduleForm {
  enabled: boolean;
  intervalMinutes: number;
}

interface AdminMessage {
  kind: AdminMessageKind;
  text: string;
}

const runResult = (run: SyncRunDTO): string => {
  if (run.type === 'players') {
    return `Procesados ${run.totalProcessed} · actualizados ${run.totalUpdated} · desactivados ${run.totalDeactivated}`;
  }
  return `Procesados ${run.totalProcessed} · matcheados ${run.totalMatched} · sin match ${run.totalUnmatched} · insertados ${run.totalInserted}`;
};

export const SyncSection: React.FC = () => {
  const [schedules, setSchedules] = useState<SyncScheduleDTO[]>([]);
  const [forms, setForms] = useState<Record<string, ScheduleForm>>({});
  const [isLoadingSchedules, setIsLoadingSchedules] = useState(true);
  const [schedulesError, setSchedulesError] = useState<string | null>(null);

  const [runs, setRuns] = useState<SyncRunDTO[]>([]);
  const [isLoadingRuns, setIsLoadingRuns] = useState(true);
  const [runsError, setRunsError] = useState<string | null>(null);

  const [activeRun, setActiveRun] = useState<SyncRunDTO | null>(null);
  const [confirmType, setConfirmType] = useState<SyncType | null>(null);
  const [busyType, setBusyType] = useState<SyncType | null>(null);
  const [savingType, setSavingType] = useState<SyncType | null>(null);
  const [message, setMessage] = useState<AdminMessage | null>(null);

  const loadSchedules = useCallback(async () => {
    setIsLoadingSchedules(true);
    setSchedulesError(null);
    try {
      const data = await listSyncSchedules();
      setSchedules(data);

      const nextForms: Record<string, ScheduleForm> = {};
      for (const schedule of data) {
        nextForms[schedule.type] = {
          enabled: schedule.enabled,
          intervalMinutes: schedule.intervalMinutes,
        };
      }
      setForms(nextForms);
    } catch {
      setSchedulesError('No se pudieron cargar las sincronizaciones programadas.');
    } finally {
      setIsLoadingSchedules(false);
    }
  }, []);

  const loadRuns = useCallback(async () => {
    setIsLoadingRuns(true);
    setRunsError(null);
    try {
      const page = await listSyncRuns({ limit: 20 });
      setRuns(page.items);
    } catch {
      setRunsError('No se pudo cargar el historial de ejecuciones.');
    } finally {
      setIsLoadingRuns(false);
    }
  }, []);

  useEffect(() => {
    void loadSchedules();
    void loadRuns();
  }, [loadSchedules, loadRuns]);

  const runningRunId = activeRun?.status === 'running' ? activeRun.id : null;

  useEffect(() => {
    if (runningRunId === null) {
      return;
    }

    let cancelled = false;
    const timer = window.setInterval(() => {
      void getSyncRun(runningRunId)
        .then((run) => {
          if (cancelled) {
            return;
          }
          setActiveRun(run);
          if (run.status !== 'running') {
            void loadRuns();
          }
        })
        .catch(() => {
          if (!cancelled) {
            setMessage({ kind: 'error', text: 'No se pudo actualizar el estado del run.' });
          }
        });
    }, POLL_INTERVAL_MS);

    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [runningRunId, loadRuns]);

  const updateForm = (type: SyncType, patch: Partial<ScheduleForm>) => {
    setForms((prev) => {
      const current = prev[type] ?? { enabled: false, intervalMinutes: MIN_INTERVAL_MINUTES };
      return { ...prev, [type]: { ...current, ...patch } };
    });
  };

  const handleSave = async (type: SyncType) => {
    const form = forms[type];
    if (!form) {
      return;
    }
    if (!Number.isFinite(form.intervalMinutes) || form.intervalMinutes < MIN_INTERVAL_MINUTES) {
      setMessage({ kind: 'error', text: 'El intervalo mínimo es de 15 minutos.' });
      return;
    }

    setSavingType(type);
    setMessage(null);
    try {
      const updated = await updateSyncSchedule(type, form);
      setSchedules((prev) => prev.map((schedule) => (schedule.type === updated.type ? updated : schedule)));
      setForms((prev) => ({
        ...prev,
        [updated.type]: { enabled: updated.enabled, intervalMinutes: updated.intervalMinutes },
      }));
      setMessage({ kind: 'success', text: `Frecuencia de ${SYNC_LABELS[type]} actualizada.` });
    } catch {
      setMessage({ kind: 'error', text: `No se pudo actualizar la frecuencia de ${SYNC_LABELS[type]}.` });
    } finally {
      setSavingType(null);
    }
  };

  const handleTrigger = async () => {
    if (!confirmType) {
      return;
    }

    const type = confirmType;
    setBusyType(type);
    setMessage(null);
    try {
      const run = await createSyncRun(type);
      setActiveRun(run);
      setConfirmType(null);
      setMessage({
        kind: run.status === 'skipped' ? 'info' : 'success',
        text:
          run.status === 'skipped'
            ? `Ya hay una sincronización de ${SYNC_LABELS[type]} en curso.`
            : `Sincronización de ${SYNC_LABELS[type]} iniciada.`,
      });
      void loadRuns();
    } catch {
      setMessage({ kind: 'error', text: `No se pudo iniciar la sincronización de ${SYNC_LABELS[type]}.` });
    } finally {
      setBusyType(null);
    }
  };

  return (
    <section className="sync-section" id="admin-sync">
      <header className="sync-section__header">
        <h2 className="sync-section__title">Sincronizaciones</h2>
        <p className="sync-section__subtitle">
          Administrá la frecuencia de cada sincronización y ejecutala cuando necesites. Las
          ejecuciones corren en segundo plano.
        </p>
      </header>

      {message && <StatusMessage kind={message.kind}>{message.text}</StatusMessage>}
      {schedulesError && <StatusMessage kind="error">{schedulesError}</StatusMessage>}

      {isLoadingSchedules ? (
        <p className="sync-section__loading">Cargando sincronizaciones programadas...</p>
      ) : schedules.length === 0 ? (
        <p className="sync-section__meta">No hay sincronizaciones configuradas.</p>
      ) : (
        <div className="sync-section__grid">
          {schedules.map((schedule) => {
            const form =
              forms[schedule.type] ?? {
                enabled: schedule.enabled,
                intervalMinutes: schedule.intervalMinutes,
              };
            const isSaving = savingType === schedule.type;
            const isRunning = activeRun?.type === schedule.type && activeRun.status === 'running';

            return (
              <article key={schedule.type} className="sync-section__card">
                <header className="sync-section__card-header">
                  <h3 className="sync-section__card-title">
                    Sincronización de {SYNC_LABELS[schedule.type]}
                  </h3>
                  <span
                    className={`sync-section__badge ${
                      form.enabled ? 'sync-section__badge--on' : 'sync-section__badge--off'
                    }`}
                  >
                    {form.enabled ? 'Habilitada' : 'Deshabilitada'}
                  </span>
                </header>

                <label className="sync-section__check">
                  <input
                    type="checkbox"
                    checked={form.enabled}
                    onChange={(event) => updateForm(schedule.type, { enabled: event.target.checked })}
                  />
                  Ejecución automática
                </label>

                <label className="sync-section__field">
                  Intervalo (minutos)
                  <input
                    type="number"
                    min={MIN_INTERVAL_MINUTES}
                    step={MIN_INTERVAL_MINUTES}
                    value={form.intervalMinutes}
                    onChange={(event) =>
                      updateForm(schedule.type, { intervalMinutes: Number(event.target.value) })
                    }
                  />
                </label>

                <p className="sync-section__meta">
                  Próxima ejecución: {formatDateTime(schedule.nextRunAt)}
                </p>

                <div className="sync-section__actions">
                  <button
                    type="button"
                    className="sync-section__button sync-section__button--primary"
                    onClick={() => void handleSave(schedule.type)}
                    disabled={isSaving}
                  >
                    {isSaving ? 'Guardando...' : 'Guardar frecuencia'}
                  </button>
                  <button
                    type="button"
                    className="sync-section__button"
                    onClick={() => setConfirmType(schedule.type)}
                    disabled={busyType !== null || isRunning}
                  >
                    {isRunning ? 'Ejecutando...' : 'Ejecutar ahora'}
                  </button>
                </div>
              </article>
            );
          })}
        </div>
      )}

      {activeRun && (
        <article className="sync-section__run">
          <h3 className="sync-section__run-title">
            Ejecución #{activeRun.id} · {SYNC_LABELS[activeRun.type]}
          </h3>
          <p className="sync-section__meta">
            Estado: <strong>{STATUS_LABELS[activeRun.status]}</strong> · Disparo:{' '}
            {activeRun.trigger === 'cron' ? 'programado' : 'manual'}
          </p>

          {activeRun.status === 'running' &&
            (activeRun.progress ? (
              <div className="sync-section__progress">
                <progress
                  value={activeRun.progress.current}
                  max={activeRun.progress.total || undefined}
                />
                <span>
                  {activeRun.progress.message} ({activeRun.progress.current}/
                  {activeRun.progress.total})
                </span>
              </div>
            ) : (
              <p className="sync-section__meta">En curso...</p>
            ))}

          {activeRun.status !== 'running' && (
            <>
              <p className="sync-section__meta">{runResult(activeRun)}</p>
              {activeRun.message && <p className="sync-section__meta">{activeRun.message}</p>}
              {activeRun.error && <StatusMessage kind="error">{activeRun.error}</StatusMessage>}
            </>
          )}
        </article>
      )}

      <article className="sync-section__history">
        <header className="sync-section__history-header">
          <h3 className="sync-section__history-title">Historial reciente</h3>
          <button
            type="button"
            className="sync-section__button"
            onClick={() => void loadRuns()}
            disabled={isLoadingRuns}
          >
            Actualizar
          </button>
        </header>

        {runsError && <StatusMessage kind="error">{runsError}</StatusMessage>}

        {isLoadingRuns ? (
          <p className="sync-section__loading">Cargando historial...</p>
        ) : runs.length === 0 ? (
          <p className="sync-section__meta">Sin ejecuciones registradas.</p>
        ) : (
          <div className="sync-section__table-wrapper">
            <table className="sync-section__table">
              <thead>
                <tr>
                  <th scope="col">Tipo</th>
                  <th scope="col">Disparo</th>
                  <th scope="col">Estado</th>
                  <th scope="col">Inicio</th>
                  <th scope="col">Duración</th>
                  <th scope="col">Resultado</th>
                  <th scope="col">Mensaje</th>
                </tr>
              </thead>
              <tbody>
                {runs.map((run) => (
                  <tr key={run.id}>
                    <td>{SYNC_LABELS[run.type]}</td>
                    <td>{run.trigger === 'cron' ? 'Programado' : 'Manual'}</td>
                    <td>
                      <span className={`sync-section__status sync-section__status--${run.status}`}>
                        {STATUS_LABELS[run.status]}
                      </span>
                    </td>
                    <td>{formatDateTime(run.startedAt)}</td>
                    <td>{formatDuration(run.startedAt, run.finishedAt)}</td>
                    <td>{runResult(run)}</td>
                    <td className="sync-section__message">{run.error || run.message || '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </article>

      <ConfirmDialog
        open={confirmType !== null}
        title="Ejecutar sincronización"
        description={
          confirmType
            ? `Se ejecutará la sincronización de ${SYNC_LABELS[confirmType]} en segundo plano. Puede tardar varios minutos.`
            : ''
        }
        confirmLabel="Ejecutar"
        busy={busyType !== null}
        onConfirm={() => void handleTrigger()}
        onCancel={() => setConfirmType(null)}
      />
    </section>
  );
};
