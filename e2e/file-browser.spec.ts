import { test, expect, mockAPI } from './fixtures';

test.describe('File Browser', () => {
  test.beforeEach(async ({ page }) => {
    await mockAPI(page);
    await page.goto('/');
  });

  test('file browser panel exists', async ({ page }) => {
    // The file browser area should exist
    const browser = page.locator('#file-browser, .browser-panel, .file-list');
    await expect(browser.first()).toBeVisible();
  });

  test('preset dropdown has options', async ({ page }) => {
    const dropdown = page.locator('#preset');
    await expect(dropdown).toBeVisible();

    // Should have options
    const options = dropdown.locator('option');
    const count = await options.count();
    expect(count).toBeGreaterThanOrEqual(1);
  });

  test('start button exists', async ({ page }) => {
    const startBtn = page.locator('#start-btn');
    await expect(startBtn).toBeVisible();
  });
});
