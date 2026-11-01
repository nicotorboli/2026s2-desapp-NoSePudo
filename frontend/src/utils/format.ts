export const parseCredits = (val: string | number | undefined | null): number => {
  if (val === undefined || val === null) return 0;
  if (typeof val === 'number') return val;
  const parsed = parseFloat(val);
  return isNaN(parsed) ? 0 : parsed;
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

