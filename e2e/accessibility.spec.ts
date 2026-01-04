import { test, expect, mockAPI } from './fixtures';

test.describe('Accessibility', () => {
  test.beforeEach(async ({ page }) => {
    await mockAPI(page);
    await page.goto('/');
  });

  test('page has title', async ({ page }) => {
    await expect(page).toHaveTitle(/Shrinkray/i);
  });

  test('buttons have accessible text', async ({ page }) => {
    const buttons = await page.locator('button:visible').all();

    for (const button of buttons.slice(0, 5)) {
      const text = await button.textContent();
      const ariaLabel = await button.getAttribute('aria-label');
      const title = await button.getAttribute('title');

      // Should have some accessible name
      const hasName = (text && text.trim().length > 0) || ariaLabel || title;
      expect(hasName).toBeTruthy();
    }
  });

  test('form inputs are labeled', async ({ page }) => {
    const selects = await page.locator('select:visible').all();

    for (const select of selects) {
      const id = await select.getAttribute('id');
      const ariaLabel = await select.getAttribute('aria-label');

      // Should have id for label association or aria-label
      expect(id || ariaLabel).toBeTruthy();
    }
  });

  test('page has lang attribute', async ({ page }) => {
    const lang = await page.locator('html').getAttribute('lang');
    expect(lang).toBeTruthy();
  });

  test('no auto-playing media', async ({ page }) => {
    const videos = await page.locator('video[autoplay]').count();
    const audios = await page.locator('audio[autoplay]').count();

    expect(videos).toBe(0);
    expect(audios).toBe(0);
  });

  test('escape closes modals', async ({ shrinkray }) => {
    await shrinkray.goto();
    await shrinkray.openSettings();
    await expect(shrinkray.settingsPanel).toBeVisible();

    await shrinkray.page.keyboard.press('Escape');
    await expect(shrinkray.settingsPanel).toBeHidden();
  });
});
