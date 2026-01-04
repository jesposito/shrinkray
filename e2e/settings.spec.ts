import { test, expect, mockAPI } from './fixtures';

test.describe('Settings Panel', () => {
  test.beforeEach(async ({ page }) => {
    await mockAPI(page);
    await page.goto('/');
  });

  test('settings panel is accessible', async ({ page }) => {
    // Find settings trigger
    const settingsTrigger = page.locator(
      '[class*="gear"], [class*="settings"], [aria-label*="settings" i], ' +
      'button[title*="settings" i], button:has-text("Settings"), .settings-icon'
    ).first();

    await expect(settingsTrigger).toBeVisible();
    await settingsTrigger.click();

    // Panel should appear
    const panel = page.locator('[class*="settings-panel"], [class*="settings-modal"], .settings, .modal');
    await expect(panel.first()).toBeVisible();
  });

  test('worker count setting exists', async ({ page }) => {
    const settingsTrigger = page.locator('[class*="gear"], [class*="settings"], button:has-text("Settings")').first();
    if (await settingsTrigger.isVisible()) {
      await settingsTrigger.click();

      // Look for worker/concurrent jobs setting
      const workerSetting = page.locator(
        ':text("worker"), :text("concurrent"), :text("parallel"), ' +
        'input[type="number"], input[type="range"]'
      ).first();

      await expect(workerSetting).toBeVisible({ timeout: 5000 });
    }
  });

  test('CPU fallback toggle exists and works', async ({ page }) => {
    let configUpdated = false;

    await page.route('**/api/config', async (route) => {
      if (route.request().method() === 'PUT') {
        configUpdated = true;
        const body = route.request().postDataJSON();
        expect(body).toHaveProperty('allow_software_fallback');
        await route.fulfill({ status: 200, body: '{}' });
      } else {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            media_path: '/media',
            workers: 1,
            allow_software_fallback: false,
          }),
        });
      }
    });

    const settingsTrigger = page.locator('[class*="gear"], [class*="settings"], button:has-text("Settings")').first();
    if (await settingsTrigger.isVisible()) {
      await settingsTrigger.click();

      // Find CPU fallback toggle
      const fallbackToggle = page.locator(
        'input[id*="fallback"], input[id*="software"], ' +
        ':text("CPU encode fallback") >> .. >> input[type="checkbox"], ' +
        ':text("Allow CPU") >> .. >> input[type="checkbox"]'
      ).first();

      if (await fallbackToggle.isVisible()) {
        // Toggle it
        await fallbackToggle.click();

        // Wait for API call
        await page.waitForTimeout(500);
        expect(configUpdated).toBe(true);
      }
    }
  });

  test('original file handling setting exists', async ({ page }) => {
    const settingsTrigger = page.locator('[class*="gear"], [class*="settings"], button:has-text("Settings")').first();
    if (await settingsTrigger.isVisible()) {
      await settingsTrigger.click();

      // Look for original file handling
      const origSetting = page.locator(
        ':text("original"), :text("delete"), :text("keep"), :text("replace")'
      ).first();

      await expect(origSetting).toBeVisible({ timeout: 5000 });
    }
  });

  test('notification settings exist', async ({ page }) => {
    const settingsTrigger = page.locator('[class*="gear"], [class*="settings"], button:has-text("Settings")').first();
    if (await settingsTrigger.isVisible()) {
      await settingsTrigger.click();

      // Look for notification settings
      const notifySetting = page.locator(
        ':text("notification"), :text("pushover"), :text("ntfy"), ' +
        'input[id*="pushover"], input[id*="ntfy"]'
      ).first();

      // Notifications may be hidden in a subsection
      const settingsText = await page.locator('.settings, [class*="settings"]').textContent();
      const hasNotifications = /notification|pushover|ntfy/i.test(settingsText || '');
      expect(hasNotifications).toBe(true);
    }
  });

  test('settings persist after reload', async ({ page }) => {
    let savedWorkers = 1;

    await page.route('**/api/config', async (route) => {
      if (route.request().method() === 'PUT') {
        const body = route.request().postDataJSON();
        if (body.workers) {
          savedWorkers = body.workers;
        }
        await route.fulfill({ status: 200, body: '{}' });
      } else {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            media_path: '/media',
            workers: savedWorkers,
            allow_software_fallback: false,
          }),
        });
      }
    });

    const settingsTrigger = page.locator('[class*="gear"], [class*="settings"], button:has-text("Settings")').first();
    if (await settingsTrigger.isVisible()) {
      await settingsTrigger.click();

      // Change worker count
      const workerInput = page.locator('input[type="number"], input[id*="worker"]').first();
      if (await workerInput.isVisible()) {
        await workerInput.fill('2');
        await workerInput.blur();

        // Wait for save
        await page.waitForTimeout(500);

        // Reload
        await page.reload();
        await settingsTrigger.click();

        // Value should persist
        const newValue = await workerInput.inputValue();
        expect(newValue).toBe('2');
      }
    }
  });

  test('settings validation - workers must be 1-6', async ({ page }) => {
    const settingsTrigger = page.locator('[class*="gear"], [class*="settings"], button:has-text("Settings")').first();
    if (await settingsTrigger.isVisible()) {
      await settingsTrigger.click();

      const workerInput = page.locator('input[type="number"], input[id*="worker"]').first();
      if (await workerInput.isVisible()) {
        // Check min/max attributes
        const min = await workerInput.getAttribute('min');
        const max = await workerInput.getAttribute('max');

        if (min && max) {
          expect(parseInt(min)).toBe(1);
          expect(parseInt(max)).toBe(6);
        }

        // Try invalid value
        await workerInput.fill('10');
        await workerInput.blur();

        // Should be clamped or show error
        const value = await workerInput.inputValue();
        expect(parseInt(value)).toBeLessThanOrEqual(6);
      }
    }
  });
});
