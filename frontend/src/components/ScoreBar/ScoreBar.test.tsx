import { describe, it, expect, afterEach } from 'vitest';
import { cleanup, render, screen } from '@testing-library/react';
import { ScoreBar } from './ScoreBar';

afterEach(cleanup);

describe('ScoreBar', () => {
  it('renders the percentage of a score', () => {
    render(<ScoreBar score={0.875} />);
    expect(screen.getByText('87.50%')).toBeDefined();
  });

  it('renders N/R when the score is missing', () => {
    render(<ScoreBar score={null} />);
    expect(screen.getByText('N/R')).toBeDefined();
  });
});
