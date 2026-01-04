import { test, expect, mockAPI } from './fixtures';

test.describe('Job Queue', () => {
  test.beforeEach(async ({ page }) => {
    await mockAPI(page);
    await page.goto('/');
  });

  test('queue panel is visible', async ({ page }) => {
    const queuePanel = page.locator('#queue-panel');
    await expect(queuePanel).toBeVisible();
  });

  test('active panel is visible', async ({ page }) => {
    const activePanel = page.locator('#active-panel');
    await expect(activePanel).toBeVisible();
  });

  test('empty queue shows message', async ({ page }) => {
    // Look for empty state message
    const emptyMessage = page.locator('.queue-empty, :text("No jobs")');
    await expect(emptyMessage.first()).toBeVisible();
  });

  test('queue has tab buttons', async ({ page }) => {
    // Should have Queue and Active tabs/buttons
    const queueTab = page.locator('button:has-text("Queue"), .tab:has-text("Queue")');
    await expect(queueTab.first()).toBeVisible();
  });
});
