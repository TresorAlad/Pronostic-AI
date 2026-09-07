import { describe, expect, it } from 'vitest';
import { marketCategory, marketLabel } from './marketLabels';

describe('marketLabels', () => {
  it('returns French label for known market', () => {
    expect(marketLabel('home_win')).toBe('Victoire domicile');
    expect(marketLabel('over_2_5')).toBe('Plus de 2,5 buts');
  });

  it('falls back for unknown market', () => {
    const label = marketLabel('custom_market_xyz');
    expect(label.length).toBeGreaterThan(0);
  });

  it('categorizes corners market', () => {
    expect(marketCategory('over_corners_9_5')).toBe('Corners');
  });
});
