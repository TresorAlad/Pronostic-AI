import { describe, expect, it } from 'vitest';
import { todayLocalISO } from './date';

describe('todayLocalISO', () => {
  it('formats local calendar date', () => {
    const d = new Date(2026, 8, 7, 23, 30);
    expect(todayLocalISO(d)).toBe('2026-09-07');
  });
});
