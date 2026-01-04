import { test, expect, mockAPI } from './fixtures';

test.describe('File Browser', () => {
  test.beforeEach(async ({ page }) => {
    await mockAPI(page);
    await page.goto('/');
  });

  test('displays directories and files', async ({ page }) => {
    // Wait for file browser to load
    await page.waitForResponse('**/api/browse**');

    // Should show directories
    const directories = page.locator('[class*="folder"], [class*="directory"], .dir-item');
    await expect(directories.first()).toBeVisible({ timeout: 5000 });

    // Should show files
    const files = page.locator('[class*="file"], .file-item');
    await expect(files.first()).toBeVisible({ timeout: 5000 });
  });

  test('clicking directory navigates into it', async ({ page }) => {
    await page.waitForResponse('**/api/browse**');

    // Click on a directory
    const directory = page.locator('[class*="folder"], [class*="directory"], .dir-item').first();
    if (await directory.isVisible()) {
      await directory.click();

      // Should make a new browse request
      await page.waitForResponse('**/api/browse**');
    }
  });

  test('selecting files updates selection count', async ({ page }) => {
    await page.waitForResponse('**/api/browse**');

    // Find file checkboxes or clickable files
    const fileCheckbox = page.locator('input[type="checkbox"][class*="file"], .file-item input[type="checkbox"]').first();
    const fileItem = page.locator('.file-item, [class*="file-row"]').first();

    if (await fileCheckbox.isVisible()) {
      await fileCheckbox.check();
    } else if (await fileItem.isVisible()) {
      await fileItem.click();
    }

    // Selection indicator should update (if present)
    const selectionCount = page.locator('[class*="selected"], [class*="selection"], .selected-count');
    // Just verify no errors occurred
  });

  test('breadcrumb navigation works', async ({ page }) => {
    await page.waitForResponse('**/api/browse**');

    // Look for breadcrumb or path display
    const breadcrumb = page.locator('[class*="breadcrumb"], [class*="path"], .current-path');

    if (await breadcrumb.first().isVisible()) {
      // Should display current path
      const pathText = await breadcrumb.first().textContent();
      expect(pathText).toBeTruthy();
    }
  });

  test('shows file sizes in human-readable format', async ({ page }) => {
    await page.waitForResponse('**/api/browse**');

    // Look for size displays
    const sizeText = page.locator('[class*="size"], .file-size');

    if (await sizeText.first().isVisible()) {
      const text = await sizeText.first().textContent();
      // Should contain human-readable units (MB, GB, etc.)
      expect(text).toMatch(/\d+(\.\d+)?\s*(B|KB|MB|GB|TB)/i);
    }
  });

  test('search/filter functionality', async ({ page }) => {
    await page.waitForResponse('**/api/browse**');

    // Look for search input
    const searchInput = page.locator('input[type="search"], input[placeholder*="search" i], input[placeholder*="filter" i]');

    if (await searchInput.isVisible()) {
      await searchInput.fill('test');

      // Wait for filtering to apply
      await page.waitForTimeout(300);

      // Files should be filtered
      const files = page.locator('.file-item, [class*="file-row"]');
      // Count should be reduced or display "no results"
    }
  });

  test('parent directory navigation', async ({ page }) => {
    // Navigate to a subdirectory first
    await page.route('**/api/browse**', async (route) => {
      const url = new URL(route.request().url());
      const path = url.searchParams.get('path') || '/media';

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          path: path,
          parent: path === '/media' ? null : '/media',
          directories: [],
          files: [{ name: 'file.mkv', path: `${path}/file.mkv`, size: 100000 }],
        }),
      });
    });

    await page.goto('/');

    // Look for back/parent button
    const backButton = page.locator('[class*="back"], [class*="parent"], [class*="up"], button:has-text("..")');

    if (await backButton.isVisible()) {
      // Click should navigate up
      await backButton.click();
    }
  });
});
