import { describe, expect, it } from 'vitest';
import { marketLabel, MARKET_LABELS, shortMarketLabel } from './marketLabels';

describe('marketLabels buts', () => {
  it('has btts_no label', () => {
    expect(MARKET_LABELS.btts_no).toBe('Les deux équipes ne marquent pas');
  });

  it('short label for btts is french buts copy', () => {
    expect(shortMarketLabel('btts')).toBe('2 marquent');
  });

  it('uses french decimal comma for over under', () => {
    expect(marketLabel('over_2_5')).toContain('2,5');
    expect(marketLabel('over_2_5')).toContain('Plus de');
  });

  it('away win accord', () => {
    expect(marketLabel('away_win')).toBe('Victoire extérieure');
  });
});
