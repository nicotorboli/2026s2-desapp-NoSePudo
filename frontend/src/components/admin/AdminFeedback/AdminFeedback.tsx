import React from 'react';
import './AdminFeedback.css';

export type AdminMessageKind = 'success' | 'error' | 'info';

interface StatusMessageProps {
  kind: AdminMessageKind;
  children: React.ReactNode;
}

export const StatusMessage: React.FC<StatusMessageProps> = ({ kind, children }) => (
  <p
    className={`admin-status admin-status--${kind}`}
    role={kind === 'error' ? 'alert' : 'status'}
  >
    {children}
  </p>
);

interface ConfirmDialogProps {
  open: boolean;
  title: string;
  description: string;
  confirmLabel?: string;
  cancelLabel?: string;
  busy?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export const ConfirmDialog: React.FC<ConfirmDialogProps> = ({
  open,
  title,
  description,
  confirmLabel = 'Confirmar',
  cancelLabel = 'Cancelar',
  busy = false,
  onConfirm,
  onCancel,
}) => {
  if (!open) {
    return null;
  }

  return (
    <div className="admin-dialog" role="dialog" aria-modal="true" aria-label={title}>
      <div className="admin-dialog__panel">
        <h3 className="admin-dialog__title">{title}</h3>
        <p className="admin-dialog__description">{description}</p>
        <div className="admin-dialog__actions">
          <button
            type="button"
            className="admin-dialog__button admin-dialog__button--cancel"
            onClick={onCancel}
            disabled={busy}
          >
            {cancelLabel}
          </button>
          <button
            type="button"
            className="admin-dialog__button admin-dialog__button--confirm"
            onClick={onConfirm}
            disabled={busy}
          >
            {busy ? 'Procesando...' : confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
};
