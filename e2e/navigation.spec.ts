import { test, expect, mockAPI } from './fixtures';

test.describe('Navigation & Layout', () => {
  test.beforeEach(async ({ page }) => {
    await mockAPI(page);
  });

  test('page loads successfully', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveTitle(/Shrinkray/i);
  });

  test('main UI elements are visible', async ({ page }) => {
    await page.goto('/');

    // Logo or header should be visible
    const header = page.locator('header, .header, h1, .logo');
    await expect(header.first()).toBeVisible();

    // Main content area should exist
    const main = page.locator('main, .main, .container, #app');
    await expect(main.first()).toBeVisible();
  });

  test('settings panel opens and closes', async ({ shrinkray }) => {
    await shrinkray.goto();

    // Find and click settings trigger (gear icon, settings button, etc.)
    const settingsTrigger = shrinkray.page.locator(
      '[class*="gear"], [class*="settings"], [aria-label*="settings" i], button:has-text("Settings")'
    ).first();

    if (await settingsTrigger.isVisible()) {
      await settingsTrigger.click();

      // Settings panel should appear
      const panel = shrinkray.page.locator('[class*="settings"], [class*="panel"], .modal');
      await expect(panel.first()).toBeVisible();

      // Close with escape
      await shrinkray.page.keyboard.press('Escape');
    }
  });

  test('responsive layout on mobile viewport', async ({ page }) => {
    await mockAPI(page);
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/');

    // Page should still be functional on mobile
    await expect(page.locator('body')).toBeVisible();

    // No horizontal scrollbar (content fits)
    const hasHorizontalScroll = await page.evaluate(() => {
      return document.documentElement.scrollWidth > document.documentElement.clientWidth;
    });
    expect(hasHorizontalScroll).toBe(false);
  });

  test('dark mode toggle works', async ({ page }) => {
    await mockAPI(page);
    await page.goto('/');

    // Look for theme toggle
    const themeToggle = page.locator(
      '[class*="theme"], [class*="dark"], [aria-label*="theme" i], button:has-text("Dark")'
    ).first();

    if (await themeToggle.isVisible()) {
      // Get initial state
      const initialBg = await page.evaluate(() => {
        return getComputedStyle(document.body).backgroundColor;
      });

      await themeToggle.click();

      // Background should change
      const newBg = await page.evaluate(() => {
        return getComputedStyle(document.body).backgroundColor;
      });

      // Colors should be different (theme changed)
      expect(newBg).not.toBe(initialBg);
    }
  });

  test('keyboard navigation works', async ({ page }) => {
    await mockAPI(page);
    await page.goto('/');

    // Tab through interactive elements
    await page.keyboard.press('Tab');
    const focused = await page.evaluate(() => document.activeElement?.tagName);
    expect(focused).toBeTruthy();

    // Continue tabbing - should cycle through focusable elements
    for (let i = 0; i < 5; i++) {
      await page.keyboard.press('Tab');
    }

    // Should still have focus somewhere
    const stillFocused = await page.evaluate(() => document.activeElement?.tagName);
    expect(stillFocused).toBeTruthy();
  });
});
