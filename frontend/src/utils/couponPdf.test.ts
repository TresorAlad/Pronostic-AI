import { describe, expect, it } from 'vitest';
import { marketLabel } from './marketLabels';

describe('couponPdf labels', () => {
  it('never exposes raw keys in resolved labels', () => {
    const label = marketLabel('over_2_5');
    expect(label).not.toContain('over_2_5');
    expect(label).toContain('2,5');
  });
});
