import { test, expect } from '@playwright/test';

const email = process.env.E2E_EMAIL;
const password = process.env.E2E_PASSWORD;

test.describe('auth and coupon flow', () => {
  test.skip(!email || !password, 'E2E_EMAIL and E2E_PASSWORD required for full flow');

  test('login, generate coupon, visit account', async ({ page }) => {
    await page.goto('/auth');
    await page.getByPlaceholder('Email').fill(email!);
    await page.getByPlaceholder('Mot de passe').fill(password!);
    await page.getByRole('button', { name: /Se connecter/i }).click();

    await expect(page).toHaveURL(/account|coupon/);

    await page.goto('/coupon');
    await expect(page.getByRole('heading', { name: /Coupon IA/i })).toBeVisible();

    const generateBtn = page.getByRole('button', { name: /Générer le coupon/i });
    await generateBtn.click();

    await expect(page.getByText(/Coupon généré|Aucune sélection/i)).toBeVisible({ timeout: 30000 });

    await page.goto('/account');
    await expect(page.getByRole('heading', { name: /Mon compte/i })).toBeVisible();
  });
});
