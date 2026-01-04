import { test, expect, mockAPI } from './fixtures';

test.describe('Preset Selection', () => {
  test.beforeEach(async ({ page }) => {
    await mockAPI(page);
    await page.goto('/');
  });

  test('preset dropdown displays options', async ({ page }) => {
    const dropdown = page.locator('#preset');
    await expect(dropdown).toBeVisible();

    // Should have multiple options
    const options = dropdown.locator('option');
    const count = await options.count();
    expect(count).toBeGreaterThanOrEqual(2);
  });

  test('preset dropdown shows codec in label', async ({ page }) => {
    const dropdown = page.locator('#preset');
    await expect(dropdown).toBeVisible();

    // Get all option text
    const options = dropdown.locator('option');
    const optionTexts = await options.allTextContents();

    // At least one option should mention HEVC or AV1
    const hasCodecLabel = optionTexts.some(t => /HEVC|AV1/i.test(t));
    expect(hasCodecLabel).toBe(true);
  });

  test('selecting preset changes value', async ({ shrinkray }) => {
    await shrinkray.goto();

    const dropdown = shrinkray.presetDropdown;
    await expect(dropdown).toBeVisible();

    // Get initial value
    const initialValue = await dropdown.inputValue();

    // Select a different preset
    const options = await dropdown.locator('option').all();
    if (options.length > 1) {
      const secondOption = await options[1].getAttribute('value');
      if (secondOption && secondOption !== initialValue) {
        await dropdown.selectOption(secondOption);
        const newValue = await dropdown.inputValue();
        expect(newValue).toBe(secondOption);
      }
    }
  });
});
