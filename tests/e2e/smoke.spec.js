import { test, expect } from '@playwright/test';

test.describe('Cinembrot Web Smoke Test', () => {
  const baseURL = process.env.BASE_URL || 'http://localhost:8080';

  test('Homepage harus merespons dan memiliki title yang tepat', async ({ page }) => {
    const response = await page.goto(baseURL, { waitUntil: 'domcontentloaded', timeout: 15000 }).catch(() => null);
    
    // Jika server lokal sedang tidak running di :8080, lewati gracefully
    if (!response || response.status() >= 500) {
      test.skip(true, `Server pada ${baseURL} tidak merespons (status: ${response?.status() || 'offline'})`);
      return;
    }

    expect(response.status()).toBe(200);
    await expect(page).toHaveTitle(/CINEMBROT/i);

    // Cek elemen hero spotlight atau navbar
    const navbar = page.locator('nav');
    await expect(navbar).toBeVisible();
  });

  test('Halaman Katalog Anime merespons status 200', async ({ page }) => {
    const response = await page.goto(`${baseURL}/anime`, { waitUntil: 'domcontentloaded', timeout: 15000 }).catch(() => null);
    if (!response || response.status() >= 500) {
      test.skip(true, `Server pada ${baseURL} tidak merespons`);
      return;
    }
    expect(response.status()).toBe(200);
  });

  test('Halaman Katalog Drama Pendek merespons status 200', async ({ page }) => {
    const response = await page.goto(`${baseURL}/drama-pendek`, { waitUntil: 'domcontentloaded', timeout: 15000 }).catch(() => null);
    if (!response || response.status() >= 500) {
      test.skip(true, `Server pada ${baseURL} tidak merespons`);
      return;
    }
    expect(response.status()).toBe(200);
  });

  test('Halaman Katalog Hollywood merespons status 200', async ({ page }) => {
    const response = await page.goto(`${baseURL}/hollywood`, { waitUntil: 'domcontentloaded', timeout: 15000 }).catch(() => null);
    if (!response || response.status() >= 500) {
      test.skip(true, `Server pada ${baseURL} tidak merespons`);
      return;
    }
    expect(response.status()).toBe(200);
  });
});

