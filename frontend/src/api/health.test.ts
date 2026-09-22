import { describe, expect, it, vi } from 'vitest';

const { axiosCreate, clientGet } = vi.hoisted(() => {
  const clientGet = vi.fn();
  return { clientGet, axiosCreate: vi.fn(() => ({ get: clientGet })) };
});

vi.mock('axios', () => ({ default: { create: axiosCreate } }));

import { getHealth } from './health';

describe('getHealth', () => {
  it('targets the backend under the /api prefix', async () => {
    // El cliente se arma al importar el modulo, y vitest limpia el historial de
    // los mocks antes de cada test: hay que reimportarlo para observar la llamada.
    vi.resetModules();
    await import('./health');

    expect(axiosCreate).toHaveBeenCalledWith({ baseURL: '/api' });
  });

  it('requests /health and returns the payload sent by the backend', async () => {
    const payload = { status: 'healthy', timestamp: '2026-09-21T00:00:00Z' };
    clientGet.mockResolvedValue({ data: payload });

    const health = await getHealth();

    expect(clientGet).toHaveBeenCalledWith('/health');
    expect(health).toEqual(payload);
  });

  it('propagates the error when the backend is unreachable', async () => {
    clientGet.mockRejectedValue(new Error('Network Error'));

    await expect(getHealth()).rejects.toThrow('Network Error');
  });
});
