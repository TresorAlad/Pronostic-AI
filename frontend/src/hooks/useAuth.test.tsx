import { describe, it, expect } from 'vitest';
import { userInitials, userLabel } from './useAuth';

describe('userLabel', () => {
  it('prefers display_name', () => {
    expect(userLabel({ id: '1', email: 'a@b.com', display_name: 'Alice Martin' })).toBe('Alice Martin');
  });

  it('falls back to email local part', () => {
    expect(userLabel({ id: '1', email: 'bob@example.com', display_name: '' })).toBe('bob');
  });
});

describe('userInitials', () => {
  it('uses two initials from display name', () => {
    expect(userInitials({ id: '1', email: 'a@b.com', display_name: 'Alice Martin' })).toBe('AM');
  });

  it('uses first two chars of email when no display name', () => {
    expect(userInitials({ id: '1', email: 'bob@example.com', display_name: '' })).toBe('BO');
  });
});
