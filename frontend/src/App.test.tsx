import { cleanup, render, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

const { getHealth } = vi.hoisted(() => ({ getHealth: vi.fn() }));

vi.mock('./api/health', () => ({ getHealth }));

import App from './App';

afterEach(cleanup);

describe('App', () => {
  it('shows a placeholder while the backend has not answered yet', () => {
    // Una promesa que no se resuelve nunca deja al componente en su estado inicial.
    getHealth.mockReturnValue(new Promise(() => {}));

    const { container } = render(<App />);

    expect(container.textContent).toContain('Status: loading...');
  });

  it('shows the status reported by the backend', async () => {
    getHealth.mockResolvedValue({ status: 'healthy', timestamp: '2026-09-21T00:00:00Z' });

    const { container } = render(<App />);

    await waitFor(() => expect(container.textContent).toContain('Status: healthy'));
  });

  it('falls back to offline when the backend is unreachable', async () => {
    getHealth.mockRejectedValue(new Error('Network Error'));

    const { container } = render(<App />);

    await waitFor(() => expect(container.textContent).toContain('Status: offline'));
  });
});
