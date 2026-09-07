import { test, expect } from '@playwright/test';

test('landing loads', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByRole('heading', { name: /Pronostics football/i })).toBeVisible();
});

test('auth page loads', async ({ page }) => {
  await page.goto('/auth');
  await expect(page.getByPlaceholder('Email')).toBeVisible();
});
