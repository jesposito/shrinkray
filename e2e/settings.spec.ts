import { test, expect, mockAPI } from './fixtures';

test.describe('Settings Panel', () => {
  test.beforeEach(async ({ page }) => {
    await mockAPI(page);
    await page.goto('/');
  });

  test('settings button exists', async ({ page }) => {
    const settingsBtn = page.locator('#settings-btn');
    await expect(settingsBtn).toBeVisible();
  });

  test('settings panel opens on click', async ({ shrinkray }) => {
    await shrinkray.goto();
    await shrinkray.openSettings();
    await expect(shrinkray.settingsPanel).toBeVisible();
  });

  test('settings panel has worker count setting', async ({ shrinkray }) => {
    await shrinkray.goto();
    await shrinkray.openSettings();

    // Look for workers input
    const workersInput = shrinkray.page.locator('#setting-workers, input[name="workers"]');
    await expect(workersInput.first()).toBeVisible();
  });

  test('settings panel has CPU fallback toggle', async ({ shrinkray }) => {
    await shrinkray.goto();
    await shrinkray.openSettings();

    // Look for CPU fallback toggle
    const fallbackToggle = shrinkray.page.locator('#setting-allow-software-fallback, :text("CPU encode fallback")');
    await expect(fallbackToggle.first()).toBeVisible();
  });

  test('settings panel closes on escape', async ({ shrinkray }) => {
    await shrinkray.goto();
    await shrinkray.openSettings();
    await expect(shrinkray.settingsPanel).toBeVisible();

    await shrinkray.page.keyboard.press('Escape');
    await expect(shrinkray.settingsPanel).toBeHidden();
  });
});
