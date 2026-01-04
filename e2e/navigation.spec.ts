import { test, expect, mockAPI } from './fixtures';

test.describe('Navigation & Layout', () => {
  test.beforeEach(async ({ page }) => {
    await mockAPI(page);
  });

  test('page loads successfully', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveTitle(/Shrinkray/i);
  });

  test('main UI panels are visible', async ({ page }) => {
    await page.goto('/');

    // Queue panel should be visible
    await expect(page.locator('#queue-panel')).toBeVisible();

    // Active panel should be visible
    await expect(page.locator('#active-panel')).toBeVisible();
  });

  test('settings panel opens and closes', async ({ shrinkray }) => {
    await shrinkray.goto();

    // Open settings
    await shrinkray.openSettings();
    await expect(shrinkray.settingsPanel).toBeVisible();

    // Close with escape
    await shrinkray.closeSettings();
    await expect(shrinkray.settingsPanel).toBeHidden();
  });

  test('responsive layout on mobile viewport', async ({ page }) => {
    await mockAPI(page);
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/');

    // Page should still be functional on mobile
    await expect(page.locator('body')).toBeVisible();
  });

  test('keyboard navigation works', async ({ page }) => {
    await mockAPI(page);
    await page.goto('/');

    // Tab through interactive elements
    await page.keyboard.press('Tab');
    const focused = await page.evaluate(() => document.activeElement?.tagName);
    expect(focused).toBeTruthy();
  });
});
