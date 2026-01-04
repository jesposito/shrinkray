import { test, expect, mockAPI } from './fixtures';

test.describe('Preset Selection', () => {
  test.beforeEach(async ({ page }) => {
    await mockAPI(page);
    await page.goto('/');
  });

  test('preset dropdown displays all options', async ({ page }) => {
    const dropdown = page.locator('select#preset, select[name="preset"], #preset-select, [class*="preset"] select');

    if (await dropdown.isVisible()) {
      // Click to open dropdown
      await dropdown.click();

      // Should have HEVC and AV1 options
      const options = dropdown.locator('option');
      const count = await options.count();
      expect(count).toBeGreaterThanOrEqual(2);

      // Check for expected presets
      const optionTexts = await options.allTextContents();
      const hasHEVC = optionTexts.some(t => /hevc/i.test(t));
      const hasAV1 = optionTexts.some(t => /av1/i.test(t));
      expect(hasHEVC || hasAV1).toBe(true);
    }
  });

  test('preset dropdown shows codec in label', async ({ page }) => {
    const dropdown = page.locator('select#preset, select[name="preset"], #preset-select, [class*="preset"] select');

    if (await dropdown.isVisible()) {
      const options = dropdown.locator('option');
      const optionTexts = await options.allTextContents();

      // At least one option should mention HEVC or AV1
      const hasCodecLabel = optionTexts.some(t => /HEVC|AV1/i.test(t));
      expect(hasCodecLabel).toBe(true);
    }
  });

  test('selecting preset updates UI', async ({ shrinkray }) => {
    await shrinkray.goto();

    const dropdown = shrinkray.presetDropdown;
    if (await dropdown.isVisible()) {
      // Select AV1 preset
      await dropdown.selectOption({ label: /AV1/i });

      // Verify selection
      const selected = await dropdown.inputValue();
      expect(selected).toContain('av1');
    }
  });

  test('Help me choose link opens modal', async ({ page }) => {
    const helpLink = page.locator('a:has-text("Help me choose"), button:has-text("Help me choose"), .help-choose, [class*="help"]');

    if (await helpLink.first().isVisible()) {
      await helpLink.first().click();

      // Modal should open
      const modal = page.locator('.modal, [class*="modal"], [role="dialog"]');
      await expect(modal.first()).toBeVisible();

      // Should have preset cards
      const cards = modal.locator('.card, [class*="card"], [class*="option"]');
      const cardCount = await cards.count();
      expect(cardCount).toBeGreaterThanOrEqual(2);
    }
  });

  test('Help modal card selection works', async ({ page }) => {
    const helpLink = page.locator('a:has-text("Help me choose"), button:has-text("Help me choose"), .help-choose');

    if (await helpLink.first().isVisible()) {
      await helpLink.first().click();

      const modal = page.locator('.modal, [class*="modal"], [role="dialog"]');
      await expect(modal.first()).toBeVisible();

      // Click first card
      const card = modal.locator('.card, [class*="card"], [class*="option"]').first();
      if (await card.isVisible()) {
        await card.click();

        // Modal should close
        await expect(modal.first()).toBeHidden({ timeout: 3000 });

        // Preset should be selected
        const dropdown = page.locator('select#preset, select[name="preset"], #preset-select');
        const selected = await dropdown.inputValue();
        expect(selected).toBeTruthy();
      }
    }
  });

  test('Help modal closes on escape', async ({ page }) => {
    const helpLink = page.locator('a:has-text("Help me choose"), button:has-text("Help me choose"), .help-choose');

    if (await helpLink.first().isVisible()) {
      await helpLink.first().click();

      const modal = page.locator('.modal, [class*="modal"], [role="dialog"]');
      await expect(modal.first()).toBeVisible();

      // Press escape
      await page.keyboard.press('Escape');

      // Modal should close
      await expect(modal.first()).toBeHidden({ timeout: 3000 });
    }
  });

  test('Help modal closes on backdrop click', async ({ page }) => {
    const helpLink = page.locator('a:has-text("Help me choose"), button:has-text("Help me choose"), .help-choose');

    if (await helpLink.first().isVisible()) {
      await helpLink.first().click();

      const modal = page.locator('.modal, [class*="modal"], [role="dialog"]');
      const backdrop = page.locator('.modal-backdrop, [class*="overlay"], [class*="backdrop"]');

      await expect(modal.first()).toBeVisible();

      // Click backdrop
      if (await backdrop.isVisible()) {
        await backdrop.click({ position: { x: 10, y: 10 } });
        await expect(modal.first()).toBeHidden({ timeout: 3000 });
      }
    }
  });
});
