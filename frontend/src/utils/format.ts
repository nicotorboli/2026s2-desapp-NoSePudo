export const formatDateTime = (value: string | null): string => {
  if (!value) {
    return '—';
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleString('es-ES');
};

export const formatDuration = (start: string, end: string | null): string => {
  if (!end) {
    return '—';
  }

  const startMs = new Date(start).getTime();
  const endMs = new Date(end).getTime();
  if (Number.isNaN(startMs) || Number.isNaN(endMs)) {
    return '—';
  }

  const seconds = Math.max(0, Math.round((endMs - startMs) / 1000));
  if (seconds < 60) {
    return `${seconds} s`;
  }

  const minutes = Math.floor(seconds / 60);
  const rest = seconds % 60;
  return `${minutes} min ${rest} s`;
};

export const parseCredits = (val: string | number | undefined | null): number => {
  if (val === undefined || val === null) return 0;
  if (typeof val === 'number') return val;
  const parsed = Number.parseFloat(val);
  return Number.isNaN(parsed) ? 0 : parsed;
};

export const formatCredits = (
  val: string | number | undefined | null,
  decimals = 4
): string => {
  const num = parseCredits(val);
  if (num === 0) return '0';
  const fixed = num.toFixed(decimals);
  return fixed.replace(/(\.\d*?[1-9])0+$/, '$1').replace(/\.0+$/, '');
};

export const translateOrderStatus = (status: string): string => {
  switch (status) {
    case 'PENDING':
      return 'Abierta';
    case 'PARTIALLY_FILLED':
      return 'Parcial';
    case 'FILLED':
      return 'Completada';
    case 'CANCELLED':
      return 'Cancelada';
    default:
      return status;
  }
};
